package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Set at link time by GoReleaser and local build scripts.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func init() {
	enrichFromBuildInfo()
}

func enrichFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if isUnset(Version) {
		if module := moduleVersion(info); module != "" {
			Version = module
		}
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.tag":
			if isUnset(Version) {
				if tag := normalizeTag(setting.Value); tag != "" {
					Version = tag
				}
			}
		case "vcs.revision":
			if isUnknown(Commit) {
				Commit = shortSHA(setting.Value)
			}
		case "vcs.time":
			if isUnknown(Date) {
				Date = setting.Value
			}
		}
	}
}

func isUnset(value string) bool {
	return value == "" || value == "dev"
}

func isUnknown(value string) bool {
	return value == "" || value == "unknown" || value == "none"
}

func moduleVersion(info *debug.BuildInfo) string {
	value := strings.TrimSpace(info.Main.Version)
	if value == "" || value == "(devel)" {
		return ""
	}
	return normalizeTag(value)
}

func normalizeTag(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "refs/tags/")
	value = strings.TrimPrefix(value, "v")
	return value
}

func shortSHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func Label() string {
	if isUnset(Version) {
		return "dev"
	}
	return Version
}

func Summary() string {
	return fmt.Sprintf("duck %s", Label())
}

func Details() string {
	return fmt.Sprintf(
		"duck %s\ncommit: %s\nbuild: %s\ngo: %s\nos/arch: %s/%s",
		Label(),
		Commit,
		Date,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

func MatchesTag(tag string) bool {
	if isUnset(Version) {
		return false
	}
	return normalizeTag(tag) == normalizeTag(Version)
}
