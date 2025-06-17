package parser

import (
	"strings"
)

type DockerfileFeatures struct {
	BaseImage            string
	BaseTag              string
	UsesLatestTag        bool
	HasUserInstruction   bool
	HasCopySensitive     bool
	HasPackageInstalls   bool
	UsesMultistage       bool
	HasHealthcheck       bool
	UsesAddInstruction   bool
	HasLabelMetadata     bool
	HasExpose            bool
	HasEntrypointOrCmd   bool
	InstallsCurlOrWget   bool
	InstallsBuildTools   bool
	HasAptGetClean       bool
	WorldWritable        bool
	HasSecretsInEnvOrArg bool
}

type DockerStageMeta struct {
	StageIndex int
	BaseImage  string
	BaseTag    string
}

func ParseDockerfile(content string) (DockerfileFeatures, []DockerStageMeta) {
	lines := strings.Split(content, "\n")

	var features DockerfileFeatures
	var stages []DockerStageMeta
	knownAliases := map[string]bool{}
	stageIndex := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.ToLower(rawLine))

		// === FROM parsing ===
		if strings.HasPrefix(strings.ToLower(line), "from ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				image := parts[1]
				imageLower := strings.ToLower(image)

				// FROM ... AS alias
				if len(parts) >= 4 && strings.ToLower(parts[2]) == "as" {
					alias := parts[3]
					knownAliases[alias] = true
				}

				// Skip if the "image" is an alias
				if knownAliases[imageLower] || strings.HasPrefix(image, "${") {
					continue
				}

				baseImage := image
				baseTag := "latest"
				if strings.Contains(image, ":") {
					split := strings.SplitN(image, ":", 2)
					baseImage = split[0]
					baseTag = split[1]
				}

				if baseTag == "latest" {
					features.UsesLatestTag = true
				}

				// Sett første base-image i DockerfileFeatures
				if features.BaseImage == "" {
					features.BaseImage = baseImage
					features.BaseTag = baseTag
				}

				stages = append(stages, DockerStageMeta{
					StageIndex: stageIndex,
					BaseImage:  baseImage,
					BaseTag:    baseTag,
				})
				stageIndex++
			}
			continue
		}

		// === Feature flags ===
		switch {
		case strings.HasPrefix(line, "user "):
			features.HasUserInstruction = true
		case strings.HasPrefix(line, "label "):
			features.HasLabelMetadata = true
		case strings.HasPrefix(line, "expose "):
			features.HasExpose = true
		case strings.HasPrefix(line, "entrypoint"), strings.HasPrefix(line, "cmd"):
			features.HasEntrypointOrCmd = true
		case strings.HasPrefix(line, "healthcheck"):
			features.HasHealthcheck = true
		case strings.HasPrefix(line, "copy "), strings.HasPrefix(line, "add "):
			if strings.Contains(line, "add ") {
				features.UsesAddInstruction = true
			}
			if strings.Contains(line, ".ssh") || strings.Contains(line, "id_rsa") || strings.Contains(line, "secrets") {
				features.HasCopySensitive = true
			}
		}

		if strings.Contains(line, "apt-get install") ||
			strings.Contains(line, "apk add") ||
			strings.Contains(line, "yum install") ||
			strings.Contains(line, "dnf install") {
			features.HasPackageInstalls = true
		}
		if strings.Contains(line, "curl") || strings.Contains(line, "wget") {
			features.InstallsCurlOrWget = true
		}
		if strings.Contains(line, "gcc") || strings.Contains(line, "make") || strings.Contains(line, "build-essential") {
			features.InstallsBuildTools = true
		}
		if strings.Contains(line, "apt-get clean") {
			features.HasAptGetClean = true
		}
		if strings.Contains(line, "chmod 777") {
			features.WorldWritable = true
		}
		if (strings.Contains(line, "env ") || strings.Contains(line, "arg ")) &&
			(strings.Contains(line, "password") || strings.Contains(line, "token") || strings.Contains(line, "secret")) {
			features.HasSecretsInEnvOrArg = true
		}
	}

	features.UsesMultistage = len(stages) > 1
	return features, stages
}
