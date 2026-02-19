package main

import "github.com/falkomer/meet-summarize/cmd"

// Version is set at build time via -ldflags "-X main.Version=x.y.z".
var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
