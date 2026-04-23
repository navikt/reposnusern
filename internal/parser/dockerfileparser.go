package parser

import (
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
}

func ParseDockerfile(content string) (DockerfileFeatures, []DockerStageMeta) {
	var features DockerfileFeatures
	var stages []DockerStageMeta
	knownAliases := map[string]struct{}{}
	stageIndex := 0

	for _, instruction := range parseDockerInstructions(content) {
		switch instruction.keyword {
		case "from":
			parsed := parseFromInstruction(instruction.value, knownAliases)
			if parsed.alias != "" {
				knownAliases[strings.ToLower(parsed.alias)] = struct{}{}
			}
			if parsed.isAlias || parsed.unresolved || parsed.baseImage == "" {
				continue
			}

			if parsed.baseTag == "latest" {
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
		case "env", "arg":
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

func parseFromInstruction(value string, knownAliases map[string]struct{}) fromInstruction {
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

	var alias string
	if i+1 < len(fields) && strings.EqualFold(fields[i], "as") {
		alias = fields[i+1]
	}

	if strings.Contains(imageRef, "$") {
		return fromInstruction{
			alias:      alias,
			unresolved: true,
		}
	}

	if _, ok := knownAliases[strings.ToLower(imageRef)]; ok {
		return fromInstruction{
			alias:   alias,
			isAlias: true,
		}
	}

	baseImage, baseTag := splitDockerImageReference(imageRef)
	return fromInstruction{
		alias:     alias,
		baseImage: baseImage,
		baseTag:   baseTag,
	}
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
