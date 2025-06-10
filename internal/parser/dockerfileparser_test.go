package parser

import (
	"testing"
)

func TestParseDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected DockerfileFeatures
	}{
		{
			name: "Basic Dockerfile with latest tag",
			content: `
FROM ubuntu:latest
USER appuser
COPY . /app
RUN apt-get update && apt-get install -y curl
ENV SECRET_TOKEN=abc123
EXPOSE 8080
ENTRYPOINT ["./start.sh"]
`,
			expected: DockerfileFeatures{
				BaseImage:            "ubuntu",
				BaseTag:              "latest",
				UsesLatestTag:        true,
				HasUserInstruction:   true,
				HasCopySensitive:     false,
				HasPackageInstalls:   true,
				UsesMultistage:       false,
				HasHealthcheck:       false,
				UsesAddInstruction:   false,
				HasLabelMetadata:     false,
				HasExpose:            true,
				HasEntrypointOrCmd:   true,
				InstallsCurlOrWget:   true,
				InstallsBuildTools:   false,
				HasAptGetClean:       false,
				WorldWritable:        false,
				HasSecretsInEnvOrArg: true,
			},
		},
		{
			name: "Multistage with sensitive copy",
			content: `
FROM golang:1.20 AS builder
COPY .ssh /root/.ssh
FROM alpine
COPY --from=builder /app /app
`,
			expected: DockerfileFeatures{
				BaseImage:            "golang",
				BaseTag:              "1.20",
				UsesLatestTag:        false,
				HasUserInstruction:   false,
				HasCopySensitive:     true,
				HasPackageInstalls:   false,
				UsesMultistage:       true,
				HasHealthcheck:       false,
				UsesAddInstruction:   false,
				HasLabelMetadata:     false,
				HasExpose:            false,
				HasEntrypointOrCmd:   false,
				InstallsCurlOrWget:   false,
				InstallsBuildTools:   false,
				HasAptGetClean:       false,
				WorldWritable:        false,
				HasSecretsInEnvOrArg: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDockerfile(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseDockerfile_AdditionalFeatures(t *testing.T) {
	content := `
FROM debian
USER appuser
ADD file.tar.gz /app
RUN apt-get update && apt-get install -y gcc make curl && apt-get clean
`
	f := ParseDockerfile(content)

	if !f.HasUserInstruction {
		t.Errorf("expected HasUserInstruction true")
	}
	if !f.UsesAddInstruction {
		t.Errorf("expected UsesAddInstruction true")
	}
	if !f.HasPackageInstalls {
		t.Errorf("expected HasPackageInstalls true")
	}
	if !f.InstallsCurlOrWget {
		t.Errorf("expected InstallsCurlOrWget true")
	}
	if !f.InstallsBuildTools {
		t.Errorf("expected InstallsBuildTools true")
	}
	if !f.HasAptGetClean {
		t.Errorf("expected HasAptGetClean true")
	}
}
