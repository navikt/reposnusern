package parser_test

import (
	"github.com/jonmartinstorm/reposnusern/internal/parser"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseCIConfig", func() {
	DescribeTable("CI config parsing detects antipatterns",
		func(content string, expected parser.CIFeatures) {
			result := parser.ParseCIConfig(content)
			Expect(result).To(Equal(expected))
		},

		Entry("npm install is detected",
			`run: npm install`,
			parser.CIFeatures{UsesNpmInstall: true},
		),

		Entry("npm ci with --ignore-scripts is NOT flagged as npm install",
			`run: npm ci --ignore-scripts`,
			parser.CIFeatures{},
		),

		Entry("npm ci without --ignore-scripts is flagged",
			`run: npm ci`,
			parser.CIFeatures{UsesNpmCiWithoutIgnoreScripts: true},
		),

		Entry("npm ci with --ignore-scripts is NOT flagged",
			`run: npm ci --ignore-scripts`,
			parser.CIFeatures{},
		),

		Entry("npm ci not present means UsesNpmCiWithoutIgnoreScripts is false",
			`run: echo hello`,
			parser.CIFeatures{},
		),

		Entry("yarn install without --frozen-lockfile is flagged",
			`run: yarn install`,
			parser.CIFeatures{UsesYarnInstallWithoutFrozen: true},
		),

		Entry("yarn install with --frozen-lockfile is NOT flagged",
			`run: yarn install --frozen-lockfile`,
			parser.CIFeatures{},
		),

		Entry("yarn install not present means UsesYarnInstallWithoutFrozen is false",
			`run: echo hello`,
			parser.CIFeatures{},
		),

		Entry("pip install without --require-hashes is flagged",
			`run: pip install -r requirements.txt`,
			parser.CIFeatures{UsesPipInstallWithoutNoCache: true, UsesPipInstallWithoutHashes: true},
		),

		Entry("pip install with --require-hashes is NOT flagged for hashes",
			`run: pip install --no-cache-dir --require-hashes -r requirements.txt`,
			parser.CIFeatures{},
		),

		Entry("pip3 install without --require-hashes is flagged",
			`run: pip3 install flask`,
			parser.CIFeatures{UsesPipInstallWithoutNoCache: true, UsesPipInstallWithoutHashes: true},
		),

		Entry("pip install with --no-cache-dir is NOT flagged for cache, but is flagged for missing hashes",
			`run: pip install --no-cache-dir requests`,
			parser.CIFeatures{UsesPipInstallWithoutHashes: true},
		),

		Entry("curl piped to bash is flagged",
			`run: curl https://example.com/install.sh | bash`,
			parser.CIFeatures{UsesCurlBashPipe: true},
		),

		Entry("curl to output file is NOT flagged",
			`run: curl https://example.com/file.txt -o out.txt`,
			parser.CIFeatures{},
		),

		Entry("sudo apt-get install is flagged",
			`run: sudo apt-get install -y git`,
			parser.CIFeatures{UsesSudo: true},
		),

		Entry("step with no sudo is not flagged",
			`run: apt-get install -y git`,
			parser.CIFeatures{},
		),

		Entry("realistic GitHub Actions YAML with multiple antipatterns",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install deps
        run: npm install
      - name: Install js deps
        run: yarn install
      - name: Setup python
        run: pip install requests
      - name: Fetch tool
        run: curl https://example.com/setup.sh | bash
      - name: Fix perms
        run: sudo chmod +x ./run.sh`,
			parser.CIFeatures{
				UsesNpmInstall:               true,
				UsesYarnInstallWithoutFrozen: true,
				UsesPipInstallWithoutNoCache: true,
				UsesPipInstallWithoutHashes:  true,
				UsesCurlBashPipe:             true,
				UsesSudo:                     true,
			},
		),
	)
})
