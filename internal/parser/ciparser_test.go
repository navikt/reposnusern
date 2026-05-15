package parser_test

import (
	"github.com/jonmartinstorm/reposnusern/internal/parser"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseCIConfig", func() {
	DescribeTable("CI config parsing correctly detects antipatterns",
		func(content string, expected parser.CIFeatures) {
			if expected.SecretNames == nil {
				expected.SecretNames = []string{}
			}
			result := parser.ParseCIConfig(content)
			Expect(result).To(Equal(expected))
		},

		Entry("npm install is detected",
			`run: npm install`,
			parser.CIFeatures{UsesNpmInstall: true},
		),

		Entry("pnpm install --frozen-lockfile is not detected as npm install",
			`run: "pnpm install --frozen-lockfile --ignore-scripts"`,
			parser.CIFeatures{},
		),

		Entry("chained commands like npm install must also be found ",
			`run: "echo 'hello world'&&npm install"`,
			parser.CIFeatures{UsesNpmInstall: true},
		),

		Entry("chained commands like npm install must also be found ",
			`run: "echo 'hello world';npm install"`,
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

		Entry("npm-family package publishing is detected",
			`name: Release
on: [push]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - run: npm publish`,
			parser.CIFeatures{UsesPackagePublish: true},
		),

		Entry("yarn npm publish in chained commands is detected",
			`name: Release
on: [push]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - run: npm test && yarn npm publish`,
			parser.CIFeatures{UsesPackagePublish: true},
		),

		Entry("publish dry run is not detected as package publish",
			`name: Release
on: [push]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - run: npm publish --dry-run`,
			parser.CIFeatures{},
		),

		Entry("npm run publish script does not count as direct package publish",
			`name: Release
on: [push]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - run: npm run publish`,
			parser.CIFeatures{},
		),

		Entry("pull_request_target scalar trigger is detected",
			`name: CI
on: pull_request_target
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok`,
			parser.CIFeatures{UsesPullRequestTarget: true},
		),

		Entry("pull_request_target in trigger list is detected",
			`name: CI
on: [push, pull_request_target]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok`,
			parser.CIFeatures{UsesPullRequestTarget: true},
		),

		Entry("pull_request_target mapping trigger is detected",
			`name: CI
on:
  pull_request_target:
    types: [opened, synchronize]
  workflow_dispatch:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok`,
			parser.CIFeatures{UsesPullRequestTarget: true},
		),

		Entry("unquoted on key still detects pull_request_target",
			`name: CI
on:
  pull_request_target:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok`,
			parser.CIFeatures{UsesPullRequestTarget: true},
		),

		Entry("pull_request_target outside top-level triggers is ignored",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: pull_request_target check
        if: github.event_name == 'pull_request_target'
        run: echo ok`,
			parser.CIFeatures{},
		),

		Entry("static secret names are extracted from expressions",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    env:
      API_TOKEN: ${{ secrets.API_TOKEN }}
      SECONDARY_TOKEN: ${{ secrets['SECONDARY_TOKEN'] }}
    steps:
      - run: echo "${{ secrets.API_TOKEN }}"`,
			parser.CIFeatures{SecretNames: []string{"API_TOKEN", "SECONDARY_TOKEN"}},
		),

		Entry("dynamic secret lookups are skipped",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo "${{ secrets[matrix.secret_name] }}"
      - run: echo "${{ secrets[inputs.secret_name] }}"`,
			parser.CIFeatures{},
		),

		Entry("secret references in comments are ignored",
			`name: CI
on: [push]
# ${{ secrets.FAKE_SECRET }}
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok`,
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
      - name: Publish
        env:
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
          RELEASE_TOKEN: ${{ secrets["RELEASE_TOKEN"] }}
        run: echo "ship it"
      - name: Fix perms
        run: sudo chmod +x ./run.sh`,
			parser.CIFeatures{
				UsesNpmInstall:               true,
				UsesYarnInstallWithoutFrozen: true,
				UsesPipInstallWithoutNoCache: true,
				UsesPipInstallWithoutHashes:  true,
				UsesCurlBashPipe:             true,
				UsesSudo:                     true,
				UsesPackagePublish:           false,
				SecretNames:                  []string{"NPM_TOKEN", "RELEASE_TOKEN"},
			},
		),

		Entry("ci antipatterns should not trigger on step names",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: npm install 
        run: echo 'hello world' 
      - name: Install js deps
        run: yarn install
      - name: Setup python
        run: pip install requests
      - name: Fetch tool
        run: curl https://example.com/setup.sh | bash
      - name: Publish
        env:
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
        run: echo "ship it"
      - name: Fix perms
        run: sudo chmod +x ./run.sh`,
			parser.CIFeatures{
				UsesNpmInstall:               false,
				UsesYarnInstallWithoutFrozen: true,
				UsesPipInstallWithoutNoCache: true,
				UsesPipInstallWithoutHashes:  true,
				UsesCurlBashPipe:             true,
				UsesSudo:                     true,
				UsesPackagePublish:           false,
				SecretNames:                  []string{"NPM_TOKEN"},
			},
		),
	)
})
