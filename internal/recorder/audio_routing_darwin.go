//go:build darwin

package recorder

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation

#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <string.h>
#include <stdlib.h>

static AudioDeviceID getCurrentOutputDevice() {
	AudioDeviceID deviceID = kAudioDeviceUnknown;
	UInt32 size = sizeof(deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
	return deviceID;
}

static OSStatus setOutputDevice(AudioDeviceID deviceID) {
	UInt32 size = sizeof(deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	return AudioObjectSetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, size, &deviceID);
}

static char* coreAudioDeviceName(AudioDeviceID deviceID) {
	CFStringRef name = NULL;
	UInt32 size = sizeof(name);
	AudioObjectPropertyAddress addr = {
		kAudioObjectPropertyName,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	OSStatus status = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &name);
	if (status != 0 || name == NULL) return NULL;
	char* buf = (char*)malloc(256);
	CFStringGetCString(name, buf, 256, kCFStringEncodingUTF8);
	CFRelease(name);
	return buf;
}

static AudioDeviceID findDeviceContaining(const char* substring) {
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	UInt32 size = 0;
	if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size) != 0) {
		return kAudioDeviceUnknown;
	}
	int n = (int)(size / sizeof(AudioDeviceID));
	AudioDeviceID* devices = (AudioDeviceID*)malloc(size);
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, devices) != 0) {
		free(devices);
		return kAudioDeviceUnknown;
	}
	AudioDeviceID result = kAudioDeviceUnknown;
	for (int i = 0; i < n; i++) {
		char* name = coreAudioDeviceName(devices[i]);
		if (name != NULL) {
			if (strstr(name, substring) != NULL) {
				result = devices[i];
				free(name);
				break;
			}
			free(name);
		}
	}
	free(devices);
	return result;
}

static CFStringRef getDeviceUID(AudioDeviceID deviceID) {
	CFStringRef uid = NULL;
	UInt32 size = sizeof(uid);
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyDeviceUID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	OSStatus status = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &uid);
	if (status != 0 || uid == NULL) return NULL;
	return uid;
}

static AudioObjectID getCoreAudioPluginID() {
	CFStringRef bundleID = CFSTR("com.apple.audio.CoreAudio");
	AudioValueTranslation translation;
	translation.mInputData = &bundleID;
	translation.mInputDataSize = sizeof(bundleID);
	AudioObjectID pluginID = kAudioObjectUnknown;
	translation.mOutputData = &pluginID;
	translation.mOutputDataSize = sizeof(pluginID);
	UInt32 size = sizeof(translation);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyPlugInForBundleID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	OSStatus status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &translation);
	if (status != 0) return kAudioObjectUnknown;
	return pluginID;
}

// createMultiOutputDevice creates a CoreAudio Multi-Output aggregate device that
// sends audio simultaneously to masterDevID (speakers/headphones) and slaveDevID
// (BlackHole). uid must be a unique C string per session to avoid collisions with
// stale devices from crashed runs. Returns kAudioDeviceUnknown on failure.
static AudioDeviceID createMultiOutputDevice(AudioDeviceID masterDevID, AudioDeviceID slaveDevID, const char* uid) {
	AudioObjectID pluginID = getCoreAudioPluginID();
	if (pluginID == kAudioObjectUnknown) return kAudioDeviceUnknown;

	CFStringRef masterUID = getDeviceUID(masterDevID);
	CFStringRef slaveUID  = getDeviceUID(slaveDevID);
	if (!masterUID || !slaveUID) {
		if (masterUID) CFRelease(masterUID);
		if (slaveUID)  CFRelease(slaveUID);
		return kAudioDeviceUnknown;
	}

	// Sub-device dictionaries
	CFStringRef subUIDKey = CFStringCreateWithCString(NULL, kAudioSubDeviceUIDKey, kCFStringEncodingUTF8);

	CFMutableDictionaryRef masterSubDict = CFDictionaryCreateMutable(NULL, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(masterSubDict, subUIDKey, masterUID);

	CFMutableDictionaryRef slaveSubDict = CFDictionaryCreateMutable(NULL, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(slaveSubDict, subUIDKey, slaveUID);
	CFRelease(subUIDKey);

	CFMutableArrayRef subDevList = CFArrayCreateMutable(NULL, 2, &kCFTypeArrayCallBacks);
	CFArrayAppendValue(subDevList, masterSubDict);
	CFArrayAppendValue(subDevList, slaveSubDict);
	CFRelease(masterSubDict);
	CFRelease(slaveSubDict);

	// Aggregate device description
	CFStringRef nameKey    = CFStringCreateWithCString(NULL, kAudioAggregateDeviceNameKey,          kCFStringEncodingUTF8);
	CFStringRef uidKey     = CFStringCreateWithCString(NULL, kAudioAggregateDeviceUIDKey,           kCFStringEncodingUTF8);
	CFStringRef subDevKey  = CFStringCreateWithCString(NULL, kAudioAggregateDeviceSubDeviceListKey, kCFStringEncodingUTF8);
	CFStringRef masterKey  = CFStringCreateWithCString(NULL, kAudioAggregateDeviceMainSubDeviceKey, kCFStringEncodingUTF8);
	CFStringRef stackedKey = CFStringCreateWithCString(NULL, kAudioAggregateDeviceIsStackedKey,     kCFStringEncodingUTF8);

	CFMutableDictionaryRef aggDesc = CFDictionaryCreateMutable(NULL, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef aggUID  = CFStringCreateWithCString(NULL, uid, kCFStringEncodingUTF8);
	CFDictionarySetValue(aggDesc, nameKey,    CFSTR("Meet Summarize Multi-Output"));
	CFDictionarySetValue(aggDesc, uidKey,     aggUID);
	CFRelease(aggUID);
	CFDictionarySetValue(aggDesc, subDevKey,  subDevList);
	CFDictionarySetValue(aggDesc, masterKey,  masterUID);
	// kCFBooleanTrue = stacked = Multi-Output mode (audio sent to ALL sub-devices simultaneously)
	CFDictionarySetValue(aggDesc, stackedKey, kCFBooleanTrue);

	CFRelease(nameKey);
	CFRelease(uidKey);
	CFRelease(subDevKey);
	CFRelease(masterKey);
	CFRelease(stackedKey);
	CFRelease(subDevList);
	CFRelease(masterUID);
	CFRelease(slaveUID);

	AudioObjectPropertyAddress createAddr = {
		kAudioPlugInCreateAggregateDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};

	AudioDeviceID newDeviceID = kAudioDeviceUnknown;
	UInt32 outSize = sizeof(newDeviceID);

	OSStatus status = AudioObjectGetPropertyData(
		pluginID,
		&createAddr,
		sizeof(aggDesc),
		&aggDesc,
		&outSize,
		&newDeviceID
	);

	CFRelease(aggDesc);

	if (status != 0) return kAudioDeviceUnknown;
	return newDeviceID;
}

static OSStatus destroyAggregateDevice(AudioDeviceID deviceID) {
	AudioObjectID pluginID = getCoreAudioPluginID();
	if (pluginID == kAudioObjectUnknown) return -1;
	AudioObjectPropertyAddress destroyAddr = {
		kAudioPlugInDestroyAggregateDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	return AudioObjectSetPropertyData(pluginID, &destroyAddr, 0, NULL, sizeof(deviceID), &deviceID);
}

// getDeviceMute returns 1 if the device output is muted, 0 otherwise.
static int getDeviceMute(AudioDeviceID deviceID) {
	UInt32 mute = 0;
	UInt32 size = sizeof(mute);
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyMute,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};
	if (!AudioObjectHasProperty(deviceID, &addr)) return 0;
	AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &mute);
	return (int)mute;
}

// setDeviceMute sets the mute state on the device output scope.
static void setDeviceMute(AudioDeviceID deviceID, int mute) {
	UInt32 m = (UInt32)mute;
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyMute,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};
	if (!AudioObjectHasProperty(deviceID, &addr)) return;
	Boolean settable = false;
	AudioObjectIsPropertySettable(deviceID, &addr, &settable);
	if (settable) AudioObjectSetPropertyData(deviceID, &addr, 0, NULL, sizeof(m), &m);
}

// getDeviceVolume returns the current volume scalar (0.0–1.0) on the master
// output channel, or -1.0 if the property is unavailable.
static Float32 getDeviceVolume(AudioDeviceID deviceID) {
	Float32 vol = -1.0f;
	UInt32 size = sizeof(vol);
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};
	if (!AudioObjectHasProperty(deviceID, &addr)) return -1.0f;
	AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &vol);
	return vol;
}

// setDeviceVolume sets the volume scalar on the master output channel.
// Does nothing if the property is not settable.
static void setDeviceVolume(AudioDeviceID deviceID, Float32 vol) {
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioDevicePropertyScopeOutput,
		kAudioObjectPropertyElementMain
	};
	if (!AudioObjectHasProperty(deviceID, &addr)) return;
	Boolean settable = false;
	AudioObjectIsPropertySettable(deviceID, &addr, &settable);
	if (settable) AudioObjectSetPropertyData(deviceID, &addr, 0, NULL, sizeof(vol), &vol);
}

// ensureAudioFlows unmutes and sets volume to 1.0 on the device so that all
// sub-devices (including BlackHole) receive the full audio signal regardless
// of whether the system was muted or the volume was at zero before recording.
static void ensureAudioFlows(AudioDeviceID deviceID) {
	// Unmute output
	setDeviceMute(deviceID, 0);

	// Set volume to 1.0 on channel 0 (master) and channels 1–2 if available.
	for (UInt32 ch = 0; ch <= 2; ch++) {
		AudioObjectPropertyAddress volAddr = {
			kAudioDevicePropertyVolumeScalar,
			kAudioDevicePropertyScopeOutput,
			ch
		};
		if (!AudioObjectHasProperty(deviceID, &volAddr)) continue;
		Boolean settable = false;
		AudioObjectIsPropertySettable(deviceID, &volAddr, &settable);
		if (!settable) continue;
		Float32 vol = 1.0f;
		AudioObjectSetPropertyData(deviceID, &volAddr, 0, NULL, sizeof(vol), &vol);
	}
}
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

// routeSystemAudioToCapture creates a temporary Multi-Output device that routes
// system audio to both the original output (speakers/headphones) and BlackHole
// simultaneously. ffmpeg then captures from BlackHole as usual.
//
// Returns a restore function that must be called when recording stops.
func routeSystemAudioToCapture() (func(), error) {
	originalID := C.getCurrentOutputDevice()

	// Save original mute state to restore after recording.
	// macOS may transfer the active device's mute/volume onto a newly selected
	// output, so we always restore explicitly.
	originalMute := C.getDeviceMute(originalID)

	// Unmute the original device BEFORE creating the Multi-Output aggregate so
	// that the aggregate inherits an unmuted state from the master sub-device.
	// We do NOT change volume here to avoid affecting the user's headphone level.
	C.setDeviceMute(originalID, 0)

	sub := C.CString("BlackHole")
	defer C.free(unsafe.Pointer(sub))

	blackholeID := C.findDeviceContaining(sub)
	if blackholeID == C.kAudioDeviceUnknown {
		return nil, fmt.Errorf("BlackHole device not found via CoreAudio")
	}

	// Use a timestamp-based UID so every recording session creates a fresh
	// aggregate device. Stale devices from crashed previous runs used a fixed
	// UID, causing CoreAudio to refuse creating a new one with the same UID.
	uid := fmt.Sprintf("com.meet-summarize.multi-output.%d", time.Now().UnixNano())
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))

	// Create Multi-Output: original speakers + BlackHole
	multiOutputID := C.createMultiOutputDevice(originalID, blackholeID, cUID)
	if multiOutputID == C.kAudioDeviceUnknown {
		return nil, fmt.Errorf("failed to create Multi-Output aggregate device")
	}

	// Let the audio subsystem initialize the new device
	time.Sleep(200 * time.Millisecond)

	if status := C.setOutputDevice(multiOutputID); status != 0 {
		C.destroyAggregateDevice(multiOutputID)
		return nil, fmt.Errorf("CoreAudio setOutputDevice failed: %d", int(status))
	}

	// Unmute the aggregate so BlackHole receives the audio signal.
	// Volume is intentionally not forced to 1.0 here to preserve the user's
	// headphone level. The aggregate inherits the original device's unmuted state.
	C.setDeviceMute(multiOutputID, 0)

	// Give CoreAudio time to route audio before ffmpeg opens BlackHole
	time.Sleep(200 * time.Millisecond)

	restore := func() {
		_ = C.setOutputDevice(originalID)
		// Restore the original mute state.
		C.setDeviceMute(originalID, originalMute)
		time.Sleep(100 * time.Millisecond)
		_ = C.destroyAggregateDevice(multiOutputID)
	}
	return restore, nil
}

// Silence unused import warning — unsafe is used in the CGo call above.
var _ = unsafe.Pointer(nil)
