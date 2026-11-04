package cli

import (
	"fmt"
	"io"
	"runtime/debug"
)

const (
	unknownCommit     = "sconosciuto"
	unknownCommitDate = "sconosciuta"
)

// printVersion prints the commit and commit date embedded in the binary at
// build time, together with the binary name.
func printVersion(w io.Writer) {
	commit, date := buildVersion()
	if commit == "" {
		commit = unknownCommit
	}
	if date == "" {
		date = unknownCommitDate
	}
	fmt.Fprintln(w, "skills-stack")
	fmt.Fprintf(w, "commit: %s\n", commit)
	fmt.Fprintf(w, "data commit: %s\n", date)
}

// buildVersion reads the VCS metadata that the Go toolchain records in the
// build info when the binary is built from a Git work tree.
func buildVersion() (commit, date string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	return versionFromSettings(info.Settings)
}

func versionFromSettings(settings []debug.BuildSetting) (commit, date string) {
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.time":
			date = setting.Value
		}
	}
	return commit, date
}
