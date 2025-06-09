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

func ParseDockerfile(content string) DockerfileFeatures {
	lines := strings.Split(content, "\n")
	var features DockerfileFeatures
	stageCount := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.ToLower(rawLine))

		// Base image og multistage
		if strings.HasPrefix(line, "from ") {
			stageCount++
			parts := strings.Fields(line)
			if len(parts) >= 2 && features.BaseImage == "" {
				image := parts[1]
				if strings.Contains(image, ":") {
					split := strings.SplitN(image, ":", 2)
					features.BaseImage = split[0]
					features.BaseTag = split[1]
					features.UsesLatestTag = (split[1] == "latest")
				} else {
					features.BaseImage = image
					features.BaseTag = "latest"
					features.UsesLatestTag = true
				}
			}
		}

		// USER
		if strings.HasPrefix(line, "user ") {
			features.HasUserInstruction = true
		}

		// COPY eller ADD
		if strings.HasPrefix(line, "copy ") || strings.HasPrefix(line, "add ") {
			if strings.Contains(line, "add ") {
				features.UsesAddInstruction = true
			}
			if strings.Contains(line, ".ssh") || strings.Contains(line, "id_rsa") || strings.Contains(line, "secrets") {
				features.HasCopySensitive = true
			}
		}

		// Install-pakker
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

		// ENV / ARG secrets
		if strings.Contains(line, "env ") || strings.Contains(line, "arg ") {
			if strings.Contains(line, "password") || strings.Contains(line, "token") || strings.Contains(line, "secret") {
				features.HasSecretsInEnvOrArg = true
			}
		}

		if strings.HasPrefix(line, "label ") {
			features.HasLabelMetadata = true
		}
		if strings.HasPrefix(line, "expose ") {
			features.HasExpose = true
		}
		if strings.HasPrefix(line, "entrypoint") || strings.HasPrefix(line, "cmd") {
			features.HasEntrypointOrCmd = true
		}
		if strings.HasPrefix(line, "healthcheck ") {
			features.HasHealthcheck = true
		}
	}

	features.UsesMultistage = stageCount > 1
	return features
}
