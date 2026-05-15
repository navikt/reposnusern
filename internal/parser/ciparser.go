package parser

import (
	"io"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	githubExpressionPattern = regexp.MustCompile(`(?s)\$\{\{.*?\}\}`)
	secretDotPattern        = regexp.MustCompile(`\bsecrets\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	secretBracketPattern    = regexp.MustCompile(`\bsecrets\s*\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`)
)

type CIFeatures struct {
	UsesNpmInstall                bool
	UsesNpmCiWithoutIgnoreScripts bool
	UsesYarnInstallWithoutFrozen  bool
	UsesPipInstallWithoutNoCache  bool
	UsesPipInstallWithoutHashes   bool
	UsesCurlBashPipe              bool
	UsesSudo                      bool
	UsesPullRequestTarget         bool
	SecretNames                   []string
}

// extractRunLines parses a GitHub Actions workflow YAML (as raw text) and
// returns only the shell lines found inside `run:` fields. Inline values
// (run: cmd) are returned as a single entry; block scalars (run: |) are
// returned one line per continuation line. This prevents false positives from
// step `name:` fields that happen to contain command-like strings.
func extractRunLines(content string) []string {
	lines := strings.Split(content, "\n")
	var result []string

	inBlock := false
	blockIndent := 0

	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))

		if inBlock {
			if trimmed == "" {
				continue
			}
			if indent <= blockIndent {
				inBlock = false
				// fall through — this line may itself be a new `run:` key
			} else {
				result = append(result, trimmed)
				continue
			}
		}

		// Strip YAML list-item marker so both `run:` and `- run:` are matched.
		normalised := strings.TrimPrefix(trimmed, "- ")
		lower := strings.ToLower(normalised)
		if !strings.HasPrefix(lower, "run:") {
			continue
		}

		rest := strings.TrimSpace(normalised[len("run:"):])
		// Strip block scalar indicators (|, >, |-,  >-, |2, etc.)
		if rest == "" || rest == "|" || rest == ">" ||
			strings.HasPrefix(rest, "|-") || strings.HasPrefix(rest, ">-") ||
			strings.HasPrefix(rest, "|2") {
			inBlock = true
			blockIndent = indent
			continue
		}

		// Inline value — strip surrounding quotes only when they match.
		if len(rest) >= 2 && ((rest[0] == '"' && rest[len(rest)-1] == '"') ||
			(rest[0] == '\'' && rest[len(rest)-1] == '\'')) {
			rest = rest[1 : len(rest)-1]
		}
		result = append(result, rest)
	}

	return result
}

// ParseCIConfig scans CI YAML content for known antipatterns and returns a
// CIFeatures struct with a boolean flag per detected antipattern.
func ParseCIConfig(content string) CIFeatures {
	var f CIFeatures

	lines := extractRunLines(content)
	for _, raw := range lines {
		line := strings.ToLower(raw)

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

	f.UsesPullRequestTarget = hasPullRequestTargetTrigger(content)
	f.SecretNames = extractSecretNames(content)

	return f
}

func hasPullRequestTargetTrigger(content string) bool {
	decoder := yaml.NewDecoder(strings.NewReader(content))

	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return false
		}

		if hasPullRequestTargetInDocument(&doc) {
			return true
		}
	}

	return false
}

func hasPullRequestTargetInDocument(doc *yaml.Node) bool {
	root := dereferenceAlias(doc)
	if root == nil {
		return false
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = dereferenceAlias(root.Content[0])
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		if !strings.EqualFold(root.Content[i].Value, "on") {
			continue
		}

		return hasPullRequestTargetEvent(root.Content[i+1])
	}

	return false
}

func hasPullRequestTargetEvent(node *yaml.Node) bool {
	node = dereferenceAlias(node)
	if node == nil {
		return false
	}

	switch node.Kind {
	case yaml.ScalarNode:
		return strings.EqualFold(node.Value, "pull_request_target")
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if hasPullRequestTargetEvent(child) {
				return true
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if strings.EqualFold(node.Content[i].Value, "pull_request_target") {
				return true
			}
		}
	}

	return false
}

func dereferenceAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return node.Alias
	}
	return node
}

func extractSecretNames(content string) []string {
	decoder := yaml.NewDecoder(strings.NewReader(content))
	namesByKey := make(map[string]string)

	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return []string{}
		}
		collectSecretNames(&doc, namesByKey)
	}

	if len(namesByKey) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(namesByKey))
	for _, name := range namesByKey {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		return strings.ToUpper(names[i]) < strings.ToUpper(names[j])
	})

	return names
}

func collectSecretNames(node *yaml.Node, namesByKey map[string]string) {
	if node == nil {
		return
	}

	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		for _, expr := range githubExpressionPattern.FindAllString(node.Value, -1) {
			for _, match := range secretDotPattern.FindAllStringSubmatch(expr, -1) {
				addSecretName(namesByKey, match[1])
			}
			for _, match := range secretBracketPattern.FindAllStringSubmatch(expr, -1) {
				addSecretName(namesByKey, match[1])
			}
		}
	}

	for _, child := range node.Content {
		collectSecretNames(child, namesByKey)
	}
}

func addSecretName(namesByKey map[string]string, name string) {
	if name == "" {
		return
	}

	key := strings.ToUpper(name)
	if _, exists := namesByKey[key]; exists {
		return
	}

	namesByKey[key] = name
}
