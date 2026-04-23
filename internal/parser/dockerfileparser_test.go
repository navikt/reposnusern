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

		Entry("FROM --platform keeps image and tag parsing correct",
			`FROM --platform=$BUILDPLATFORM golang:1.22 AS builder`,
			parser.DockerfileFeatures{
				BaseImage:     "golang",
				BaseTag:       "1.22",
				UsesLatestTag: false,
			},
		),

		Entry("Registry port is not mistaken for image tag delimiter",
			`FROM ghcr.io:443/navikt/app:1.2.3`,
			parser.DockerfileFeatures{
				BaseImage:     "ghcr.io:443/navikt/app",
				BaseTag:       "1.2.3",
				UsesLatestTag: false,
			},
		),

		Entry("Digest reference does not imply latest",
			`FROM alpine@sha256:deadbeef`,
			parser.DockerfileFeatures{
				BaseImage:     "alpine",
				BaseTag:       "",
				UsesLatestTag: false,
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
				BaseImage:        "alpine",
				BaseTag:          "latest",
				UsesLatestTag:    true,
				HasLabelMetadata: true,
				HasExpose:        true,
				HasHealthcheck:   true,
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

		Entry("COPY with source named add is NOT flagged as ADD instruction",
			`FROM debian
COPY add /opt/`,
			parser.DockerfileFeatures{
				BaseImage:          "debian",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				UsesAddInstruction: false,
			},
		),

		Entry("npm install in RUN is flagged",
			`FROM node:18
RUN npm install`,
			parser.DockerfileFeatures{
				BaseImage:      "node",
				BaseTag:        "18",
				UsesNpmInstall: true,
			},
		),

		Entry("npm ci with --ignore-scripts is NOT flagged",
			`FROM node:18
RUN npm ci --ignore-scripts`,
			parser.DockerfileFeatures{
				BaseImage: "node",
				BaseTag:   "18",
			},
		),

		Entry("npm ci without --ignore-scripts is flagged",
			`FROM node:18
RUN npm ci`,
			parser.DockerfileFeatures{
				BaseImage:                     "node",
				BaseTag:                       "18",
				UsesNpmCiWithoutIgnoreScripts: true,
			},
		),

		Entry("yarn install without --frozen-lockfile is flagged",
			`FROM node:18
RUN yarn install`,
			parser.DockerfileFeatures{
				BaseImage:                    "node",
				BaseTag:                      "18",
				UsesYarnInstallWithoutFrozen: true,
			},
		),

		Entry("yarn install with --frozen-lockfile is NOT flagged",
			`FROM node:18
RUN yarn install --frozen-lockfile`,
			parser.DockerfileFeatures{
				BaseImage: "node",
				BaseTag:   "18",
			},
		),

		Entry("pip install without --no-cache-dir is flagged",
			`FROM python:3.12
RUN pip install requests`,
			parser.DockerfileFeatures{
				BaseImage:                    "python",
				BaseTag:                      "3.12",
				UsesPipInstallWithoutNoCache: true,
				UsesPipInstallWithoutHashes:  true,
			},
		),

		Entry("pip install with --no-cache-dir is NOT flagged",
			`FROM python:3.12
RUN pip install --no-cache-dir requests`,
			parser.DockerfileFeatures{
				BaseImage:                   "python",
				BaseTag:                     "3.12",
				UsesPipInstallWithoutHashes: true,
			},
		),

		Entry("pip install with --require-hashes is NOT flagged for hashes",
			`FROM python:3.12
RUN pip install --require-hashes -r requirements.txt`,
			parser.DockerfileFeatures{
				BaseImage:                    "python",
				BaseTag:                      "3.12",
				UsesPipInstallWithoutNoCache: true,
			},
		),

		Entry("pip install with both mitigations is NOT flagged",
			`FROM python:3.12
RUN pip install --no-cache-dir --require-hashes -r requirements.txt`,
			parser.DockerfileFeatures{
				BaseImage: "python",
				BaseTag:   "3.12",
			},
		),

		Entry("curl piped to bash in RUN is flagged",
			`FROM ubuntu
RUN curl https://get.example.com/install.sh | bash`,
			parser.DockerfileFeatures{
				BaseImage:          "ubuntu",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				InstallsCurlOrWget: true,
				UsesCurlBashPipe:   true,
			},
		),

		Entry("curl to output file is NOT flagged as pipe",
			`FROM ubuntu
RUN curl https://example.com/file.txt -o /tmp/file.txt`,
			parser.DockerfileFeatures{
				BaseImage:          "ubuntu",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				InstallsCurlOrWget: true,
			},
		),

		Entry("Comment lines do not trigger feature flags",
			`FROM alpine
# ENV SECRET_TOKEN=abc123
# RUN npm install
# chmod 777 /tmp/file`,
			parser.DockerfileFeatures{
				BaseImage:     "alpine",
				BaseTag:       "latest",
				UsesLatestTag: true,
			},
		),

		Entry("Multiline RUN keeps pip mitigation detection on the joined instruction",
			`FROM python:3.12
RUN pip install \
  --no-cache-dir \
  --require-hashes \
  -r requirements.txt`,
			parser.DockerfileFeatures{
				BaseImage: "python",
				BaseTag:   "3.12",
			},
		),

		Entry("COPY with flags still detects sensitive paths",
			`FROM alpine AS base
COPY --from=builder .ssh /root/.ssh`,
			parser.DockerfileFeatures{
				BaseImage:          "alpine",
				BaseTag:            "latest",
				UsesLatestTag:      true,
				HasCopySensitive:   true,
				UsesMultistage:     false,
				UsesAddInstruction: false,
			},
		),

		Entry("Unresolved variable FROM falls through to the first parseable external stage",
			`ARG BASE_IMAGE
FROM ${BASE_IMAGE} AS dynamic
FROM alpine:3.20`,
			parser.DockerfileFeatures{
				BaseImage:     "alpine",
				BaseTag:       "3.20",
				UsesLatestTag: false,
			},
		),
	)

	DescribeTable("Dockerfile parsing produces correct stage metadata",
		func(content string, expected []parser.DockerStageMeta) {
			_, stages := parser.ParseDockerfile(content)
			Expect(stages).To(Equal(expected))
		},

		Entry("Platform flag is ignored for stage source parsing",
			`FROM --platform=$BUILDPLATFORM golang:1.22 AS builder`,
			[]parser.DockerStageMeta{
				{StageIndex: 0, BaseImage: "golang", BaseTag: "1.22"},
			},
		),

		Entry("Registry ports are preserved in stage base image",
			`FROM ghcr.io:443/navikt/app:1.2.3`,
			[]parser.DockerStageMeta{
				{StageIndex: 0, BaseImage: "ghcr.io:443/navikt/app", BaseTag: "1.2.3"},
			},
		),

		Entry("Digest references produce empty base tag",
			`FROM alpine@sha256:deadbeef`,
			[]parser.DockerStageMeta{
				{StageIndex: 0, BaseImage: "alpine", BaseTag: ""},
			},
		),

		Entry("Alias-derived stages are not persisted as external stages",
			`FROM alpine:3.20 AS base
FROM base AS final`,
			[]parser.DockerStageMeta{
				{StageIndex: 0, BaseImage: "alpine", BaseTag: "3.20"},
			},
		),

		Entry("Unresolved variable FROM stages are skipped instead of emitting misleading metadata",
			`ARG BASE_IMAGE
FROM ${BASE_IMAGE} AS dynamic
FROM alpine:3.20`,
			[]parser.DockerStageMeta{
				{StageIndex: 0, BaseImage: "alpine", BaseTag: "3.20"},
			},
		),
	)
})
