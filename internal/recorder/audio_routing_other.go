//go:build !darwin

package recorder

// routeSystemAudioToCapture is a no-op on non-macOS platforms.
func routeSystemAudioToCapture() (func(), error) {
	return func() {}, nil
}
