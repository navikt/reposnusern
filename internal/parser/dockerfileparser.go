package parser

import (
	"strings"
)

type DockerfileFeatures struct {
	BaseImage          string
	BaseTag            string
	UsesLatestTag      bool
	HasUserInstruction bool
	HasCopySensitive   bool
	HasPackageInstalls bool
	UsesMultistage     bool
}

func ParseDockerfile(content string) DockerfileFeatures {
	lines := strings.Split(content, "\n")
	var features DockerfileFeatures
	stageCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))

		if strings.HasPrefix(line, "from ") {
			stageCount++
			parts := strings.Fields(line)
			if len(parts) >= 2 && features.BaseImage == "" {
				// FROM ubuntu:20.04 AS builder → parts[1] = ubuntu:20.04
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

		if strings.HasPrefix(line, "user ") {
			features.HasUserInstruction = true
		}

		if strings.HasPrefix(line, "copy ") || strings.HasPrefix(line, "add ") {
			// Naiv sensitiv path-match (kan utvides)
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
	}

	features.UsesMultistage = stageCount > 1
	return features
}
