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
	DescribeTable("Basic parsing",
		func(content string, expected parser.DockerfileFeatures) {
			result := parser.ParseDockerfile(content)
			Expect(result).To(Equal(expected))
		},

		Entry("Basic Dockerfile with latest tag",
			`FROM ubuntu:latest
USER appuser
COPY . /app
RUN apt-get update && apt-get install -y curl
ENV SECRET_TOKEN=abc123
EXPOSE 8080
ENTRYPOINT ["./start.sh"]
`,
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

		Entry("Multistage with sensitive copy",
			`FROM golang:1.20 AS builder
COPY .ssh /root/.ssh
FROM alpine
COPY --from=builder /app /app
`,
			parser.DockerfileFeatures{
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
		),
	)

	It("should detect advanced features like add, curl, build tools, and apt clean", func() {
		content := `
FROM debian
USER appuser
ADD file.tar.gz /app
RUN apt-get update && apt-get install -y gcc make curl && apt-get clean
`
		f := parser.ParseDockerfile(content)

		Expect(f.HasUserInstruction).To(BeTrue())
		Expect(f.UsesAddInstruction).To(BeTrue())
		Expect(f.HasPackageInstalls).To(BeTrue())
		Expect(f.InstallsCurlOrWget).To(BeTrue())
		Expect(f.InstallsBuildTools).To(BeTrue())
		Expect(f.HasAptGetClean).To(BeTrue())
	})

	It("should detect label, expose, healthcheck, and world writable", func() {
		content := `
FROM alpine
LABEL version="1.0"
EXPOSE 443
HEALTHCHECK CMD curl -f http://localhost || exit 1
RUN chmod 777 /tmp/file
`
		f := parser.ParseDockerfile(content)

		Expect(f.HasLabelMetadata).To(BeTrue())
		Expect(f.HasExpose).To(BeTrue())
		Expect(f.HasHealthcheck).To(BeTrue())
		Expect(f.WorldWritable).To(BeTrue())
	})
})
