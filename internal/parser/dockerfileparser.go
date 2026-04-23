package parser

import (
	"regexp"
	"strings"
)

type DockerfileFeatures struct {
	BaseImage                     string
	BaseTag                       string
	UsesLatestTag                 bool
	HasUserInstruction            bool
	HasCopySensitive              bool
	HasPackageInstalls            bool
	UsesMultistage                bool
	HasHealthcheck                bool
	UsesAddInstruction            bool
	HasLabelMetadata              bool
	HasExpose                     bool
	HasEntrypointOrCmd            bool
	InstallsCurlOrWget            bool
	InstallsBuildTools            bool
	HasAptGetClean                bool
	WorldWritable                 bool
	HasSecretsInEnvOrArg          bool
	UsesNpmInstall                bool
	UsesNpmCiWithoutIgnoreScripts bool
	UsesYarnInstallWithoutFrozen  bool
	UsesPipInstallWithoutNoCache  bool
	UsesPipInstallWithoutHashes   bool
	UsesCurlBashPipe              bool
}

type DockerStageMeta struct {
	StageIndex int
	BaseImage  string
	BaseTag    string
}

type dockerInstruction struct {
	keyword string
	value   string
}

type fromInstruction struct {
	alias      string
	baseImage  string
	baseTag    string
	isAlias    bool
	unresolved bool
	parseable  bool
}

func ParseDockerfile(content string) (DockerfileFeatures, []DockerStageMeta) {
	var features DockerfileFeatures
	var stages []DockerStageMeta
	globalArgs := map[string]string{}
	knownAliases := map[string]struct{}{}
	stageIndex := 0
	seenFrom := false

	for _, instruction := range parseDockerInstructions(content) {
		switch instruction.keyword {
		case "arg":
			lowerValue := strings.ToLower(instruction.value)
			if strings.Contains(lowerValue, "password") || strings.Contains(lowerValue, "token") || strings.Contains(lowerValue, "secret") {
				features.HasSecretsInEnvOrArg = true
			}
			if seenFrom {
				continue
			}
			name, defaultValue, hasDefault := parseArgInstruction(instruction.value)
			if name == "" || !hasDefault {
				continue
			}
			resolvedValue, _ := resolveArgReferences(defaultValue, globalArgs)
			globalArgs[name] = resolvedValue
		case "from":
			seenFrom = true
			parsed := parseFromInstruction(instruction.value, knownAliases, globalArgs)
			if parsed.alias != "" {
				knownAliases[strings.ToLower(parsed.alias)] = struct{}{}
			}
			if parsed.isAlias || parsed.baseImage == "" {
				continue
			}

			if parsed.parseable && parsed.baseTag == "latest" {
				features.UsesLatestTag = true
			}
			if features.BaseImage == "" {
				features.BaseImage = parsed.baseImage
				features.BaseTag = parsed.baseTag
			}

			stages = append(stages, DockerStageMeta{
				StageIndex: stageIndex,
				BaseImage:  parsed.baseImage,
				BaseTag:    parsed.baseTag,
			})
			stageIndex++
		case "user":
			features.HasUserInstruction = true
		case "label":
			features.HasLabelMetadata = true
		case "expose":
			features.HasExpose = true
		case "entrypoint", "cmd":
			features.HasEntrypointOrCmd = true
		case "healthcheck":
			features.HasHealthcheck = true
		case "copy", "add":
			lowerValue := strings.ToLower(instruction.value)
			if instruction.keyword == "add" {
				features.UsesAddInstruction = true
			}
			if strings.Contains(lowerValue, ".ssh") || strings.Contains(lowerValue, "id_rsa") || strings.Contains(lowerValue, "secrets") {
				features.HasCopySensitive = true
			}
		case "run":
			lowerValue := strings.ToLower(instruction.value)
			if strings.Contains(lowerValue, "apt-get install") ||
				strings.Contains(lowerValue, "apk add") ||
				strings.Contains(lowerValue, "yum install") ||
				strings.Contains(lowerValue, "dnf install") {
				features.HasPackageInstalls = true
			}
			if strings.Contains(lowerValue, "curl") || strings.Contains(lowerValue, "wget") {
				features.InstallsCurlOrWget = true
			}
			if strings.Contains(lowerValue, "gcc") || strings.Contains(lowerValue, "make") || strings.Contains(lowerValue, "build-essential") {
				features.InstallsBuildTools = true
			}
			if strings.Contains(lowerValue, "apt-get clean") {
				features.HasAptGetClean = true
			}
			if strings.Contains(lowerValue, "chmod 777") {
				features.WorldWritable = true
			}
			if isNpmInstall(lowerValue) {
				features.UsesNpmInstall = true
			}
			if isNpmCiWithoutIgnoreScripts(lowerValue) {
				features.UsesNpmCiWithoutIgnoreScripts = true
			}
			if isYarnInstallWithoutFrozen(lowerValue) {
				features.UsesYarnInstallWithoutFrozen = true
			}
			if isPipInstallWithoutNoCache(lowerValue) {
				features.UsesPipInstallWithoutNoCache = true
			}
			if isPipInstallWithoutHashes(lowerValue) {
				features.UsesPipInstallWithoutHashes = true
			}
			if isCurlBashPipe(lowerValue) {
				features.UsesCurlBashPipe = true
			}
		case "env":
			lowerValue := strings.ToLower(instruction.value)
			if strings.Contains(lowerValue, "password") || strings.Contains(lowerValue, "token") || strings.Contains(lowerValue, "secret") {
				features.HasSecretsInEnvOrArg = true
			}
		}
	}

	features.UsesMultistage = len(stages) > 1
	return features, stages
}

func parseDockerInstructions(content string) []dockerInstruction {
	lines := strings.Split(content, "\n")
	var instructions []dockerInstruction
	var current []string

	flushCurrent := func() {
		if len(current) == 0 {
			return
		}

		joined := strings.Join(current, " ")
		current = nil

		fields := strings.Fields(joined)
		if len(fields) == 0 {
			return
		}

		keyword := strings.ToLower(fields[0])
		value := strings.TrimSpace(joined[len(fields[0]):])
		instructions = append(instructions, dockerInstruction{
			keyword: keyword,
			value:   value,
		})
	}

	for _, rawLine := range lines {
		trimmedLine := strings.TrimSpace(rawLine)
		if len(current) == 0 && (trimmedLine == "" || strings.HasPrefix(trimmedLine, "#")) {
			continue
		}

		line := strings.TrimRight(rawLine, " \t\r")
		continuation := strings.HasSuffix(line, "\\")
		segment := strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		if segment != "" {
			current = append(current, segment)
		}

		if continuation {
			continue
		}

		flushCurrent()
	}

	flushCurrent()
	return instructions
}

func parseFromInstruction(value string, knownAliases map[string]struct{}, globalArgs map[string]string) fromInstruction {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return fromInstruction{}
	}

	i := 0
	for i < len(fields) && strings.HasPrefix(fields[i], "--") {
		i++
	}
	if i >= len(fields) {
		return fromInstruction{}
	}

	imageRef := fields[i]
	i++
	resolvedRef, unresolved := resolveArgReferences(imageRef, globalArgs)

	var alias string
	if i+1 < len(fields) && strings.EqualFold(fields[i], "as") {
		alias = fields[i+1]
	}

	if _, ok := knownAliases[strings.ToLower(resolvedRef)]; ok {
		return fromInstruction{
			alias:   alias,
			isAlias: true,
		}
	}

	if unresolved {
		return fromInstruction{
			alias:      alias,
			baseImage:  resolvedRef,
			baseTag:    "",
			unresolved: true,
		}
	}

	baseImage, baseTag := splitDockerImageReference(resolvedRef)
	return fromInstruction{
		alias:     alias,
		baseImage: baseImage,
		baseTag:   baseTag,
		parseable: true,
	}
}

func parseArgInstruction(value string) (string, string, bool) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", "", false
	}

	assignment := fields[0]
	if eq := strings.Index(assignment, "="); eq >= 0 {
		return assignment[:eq], assignment[eq+1:], true
	}

	return assignment, "", false
}

var argReferencePattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

func resolveArgReferences(value string, args map[string]string) (string, bool) {
	unresolved := false
	resolved := argReferencePattern.ReplaceAllStringFunc(value, func(match string) string {
		name := strings.TrimPrefix(match, "$")
		name = strings.TrimPrefix(name, "{")
		name = strings.TrimSuffix(name, "}")

		argValue, ok := args[name]
		if !ok {
			unresolved = true
			return match
		}
		return argValue
	})

	return resolved, unresolved
}

func splitDockerImageReference(ref string) (string, string) {
	if ref == "" {
		return "", ""
	}

	if at := strings.Index(ref, "@"); at >= 0 {
		return ref[:at], ""
	}

	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		return ref[:lastColon], ref[lastColon+1:]
	}

	return ref, "latest"
}
