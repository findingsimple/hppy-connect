package main

import "github.com/findingsimple/hppy-connect/cmd/hppycli/cmd"

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, buildDate)
	cmd.Execute()
}
