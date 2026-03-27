package parser

import (
	"strings"
)

type CIFeatures struct {
	UsesNpmInstall                bool
	UsesNpmCiWithoutIgnoreScripts bool
	UsesYarnInstallWithoutFrozen  bool
	UsesPipInstallWithoutNoCache  bool
	UsesPipInstallWithoutHashes   bool
	UsesCurlBashPipe              bool
	UsesSudo                      bool
}

// ParseCIConfig scans CI YAML content for known antipatterns and returns a
// CIFeatures struct with a boolean flag per detected antipattern.
func ParseCIConfig(content string) CIFeatures {
	var f CIFeatures

	lines := strings.Split(content, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(strings.ToLower(raw))

		if isNpmInstall(line) {
			f.UsesNpmInstall = true
		}
		if isNpmCiWithoutIgnoreScripts(line) {
			f.UsesNpmCiWithoutIgnoreScripts = true
		}
		if isYarnInstallWithoutFrozen(line) {
			f.UsesYarnInstallWithoutFrozen = true
		}
		if isPipInstallWithoutNoCache(line) {
			f.UsesPipInstallWithoutNoCache = true
		}
		if isPipInstallWithoutHashes(line) {
			f.UsesPipInstallWithoutHashes = true
		}
		if isCurlBashPipe(line) {
			f.UsesCurlBashPipe = true
		}
		if isSudo(line) {
			f.UsesSudo = true
		}
	}

	return f
}
