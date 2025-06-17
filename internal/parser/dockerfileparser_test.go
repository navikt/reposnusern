package parser_test

import (
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/parser"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dockerfile Parser Suite")
}

var _ = Describe("ParseDockerfile", func() {
	DescribeTable("Dockerfile parsing produces correct features",
		func(content string, expected parser.DockerfileFeatures) {
			result, _ := parser.ParseDockerfile(content)
			Expect(result).To(Equal(expected))
		},

		Entry("Basic latest image and user/copy/install",
			`FROM ubuntu:latest
USER appuser
COPY . /app
RUN apt-get update && apt-get install -y curl
ENV SECRET_TOKEN=abc123
EXPOSE 8080
ENTRYPOINT ["./start.sh"]`,
			parser.DockerfileFeatures{
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
		),

		Entry("Multistage build with alias and sensitive copy",
			`FROM golang:1.19 AS builder
COPY .ssh /root/.ssh
FROM alpine
COPY --from=builder /app /app`,
			parser.DockerfileFeatures{
				BaseImage:            "golang",
				BaseTag:              "1.19",
				UsesLatestTag:        true,
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
		),

		Entry("Complex RUN with build tools and curl",
			`FROM debian
RUN apt-get update && apt-get install -y gcc make curl && apt-get clean`,
			parser.DockerfileFeatures{
				BaseImage:            "debian",
				BaseTag:              "latest",
				UsesLatestTag:        true,
				HasPackageInstalls:   true,
				UsesMultistage:       false,
				HasHealthcheck:       false,
				UsesAddInstruction:   false,
				HasLabelMetadata:     false,
				HasExpose:            false,
				HasEntrypointOrCmd:   false,
				InstallsCurlOrWget:   true,
				InstallsBuildTools:   true,
				HasAptGetClean:       true,
				WorldWritable:        false,
				HasSecretsInEnvOrArg: false,
			},
		),

		Entry("ARG secret triggers detection",
			`FROM alpine
ARG SECRET_TOKEN`,
			parser.DockerfileFeatures{
				BaseImage:            "alpine",
				BaseTag:              "latest",
				UsesLatestTag:        true,
				HasSecretsInEnvOrArg: true,
			},
		),

		Entry("ENV secret triggers detection",
			`FROM alpine
ENV DB_PASSWORD=supersecret`,
			parser.DockerfileFeatures{
				BaseImage:            "alpine",
				BaseTag:              "latest",
				UsesLatestTag:        true,
				HasSecretsInEnvOrArg: true,
			},
		),

		Entry("World writable detected with chmod 777",
			`FROM busybox
RUN chmod 777 /data/file`,
			parser.DockerfileFeatures{
				BaseImage:     "busybox",
				BaseTag:       "latest",
				UsesLatestTag: true,
				WorldWritable: true,
			},
		),

		Entry("Label, expose and healthcheck",
			`FROM alpine
LABEL version="1.0"
EXPOSE 443
HEALTHCHECK CMD curl -f http://localhost || exit 1`,
			parser.DockerfileFeatures{
				BaseImage:          "alpine",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				HasLabelMetadata:   true,
				HasExpose:          true,
				HasHealthcheck:     true,
				InstallsCurlOrWget: true,
			},
		),

		Entry("Add instruction is detected",
			`FROM debian
ADD file.tar.gz /opt/`,
			parser.DockerfileFeatures{
				BaseImage:          "debian",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				UsesAddInstruction: true,
			},
		),
	)
})
