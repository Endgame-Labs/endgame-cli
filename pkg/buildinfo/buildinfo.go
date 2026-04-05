package buildinfo

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	Version = "v0.2.0"
	Commit  = "unknown"
	Date    = "unknown"
)

func Summary() string {
	version, commit, date := resolved()
	return fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

func resolved() (string, string, string) {
	version := Version
	commit := Commit
	date := Date

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, date
	}

	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	if commit == "unknown" {
		if vcsRevision := setting(info, "vcs.revision"); vcsRevision != "" {
			commit = shortCommit(vcsRevision)
		}
	}

	if date == "unknown" {
		if vcsTime := setting(info, "vcs.time"); vcsTime != "" {
			date = vcsTime
		}
	}

	return version, commit, date
}

func setting(info *debug.BuildInfo, key string) string {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

func shortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}
