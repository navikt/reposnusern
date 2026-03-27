package parser

import "strings"

// splitRunSegments splits a shell command line on `&&`, `||`, `;` and `|` so
// that each individual command can be checked independently.
func splitRunSegments(line string) []string {
replacer := strings.NewReplacer("&&", "\x00", "||", "\x00", ";", "\x00", "|", "\x00")
return strings.Split(replacer.Replace(line), "\x00")
}

// containsCmd returns true if segment contains cmd as a standalone command:
// cmd must be followed by a space or end-of-string.
func containsCmd(segment, cmd string) bool {
idx := strings.Index(segment, cmd)
if idx < 0 {
return false
}
end := idx + len(cmd)
return end == len(segment) || segment[end] == ' '
}

// isNpmInstall detects bare `npm install` (not `npm install-ci-test` or similar).
func isNpmInstall(line string) bool {
for _, segment := range splitRunSegments(line) {
if containsCmd(strings.TrimSpace(segment), "npm install") {
return true
}
}
return false
}

// isNpmCiWithoutIgnoreScripts returns true only when `npm ci` is present but
// `--ignore-scripts` is absent. Returns false when npm ci is not used at all.
func isNpmCiWithoutIgnoreScripts(line string) bool {
hasNpmCi := false
for _, segment := range splitRunSegments(line) {
if containsCmd(strings.TrimSpace(segment), "npm ci") {
hasNpmCi = true
break
}
}
if !hasNpmCi {
return false
}
return !strings.Contains(line, "--ignore-scripts")
}

// isYarnInstallWithoutFrozen detects `yarn install` without `--frozen-lockfile`.
func isYarnInstallWithoutFrozen(line string) bool {
hasYarnInstall := false
for _, segment := range splitRunSegments(line) {
if containsCmd(strings.TrimSpace(segment), "yarn install") {
hasYarnInstall = true
break
}
}
if !hasYarnInstall {
return false
}
return !strings.Contains(line, "--frozen-lockfile")
}

// isPipInstallWithoutNoCache detects `pip install` (or `pip3 install`) without
// the `--no-cache-dir` flag.
func isPipInstallWithoutNoCache(line string) bool {
hasPipInstall := false
for _, segment := range splitRunSegments(line) {
s := strings.TrimSpace(segment)
if containsCmd(s, "pip install") || containsCmd(s, "pip3 install") {
hasPipInstall = true
break
}
}
if !hasPipInstall {
return false
}
return !strings.Contains(line, "--no-cache-dir")
}

// isPipInstallWithoutHashes detects `pip install` (or `pip3 install`) without
// the `--require-hashes` flag.
func isPipInstallWithoutHashes(line string) bool {
hasPipInstall := false
for _, segment := range splitRunSegments(line) {
s := strings.TrimSpace(segment)
if containsCmd(s, "pip install") || containsCmd(s, "pip3 install") {
hasPipInstall = true
break
}
}
if !hasPipInstall {
return false
}
return !strings.Contains(line, "--require-hashes")
}

// isCurlBashPipe detects `curl ... | bash` or `wget ... | sh`.
func isCurlBashPipe(line string) bool {
hasCurlOrWget := strings.Contains(line, "curl ") || strings.Contains(line, "wget ")
if !hasCurlOrWget {
return false
}
return strings.Contains(line, "| bash") ||
strings.Contains(line, "| sh") ||
strings.Contains(line, "|bash") ||
strings.Contains(line, "|sh")
}

// isSudo detects usage of sudo.
func isSudo(line string) bool {
return line == "sudo" ||
strings.HasPrefix(line, "sudo ") ||
strings.Contains(line, " sudo ") ||
strings.Contains(line, "&& sudo ") ||
strings.Contains(line, "| sudo ")
}
