package parser

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("extractRunLines", func() {
	DescribeTable("extracts only shell lines from run: fields",
		func(content string, expected []string) {
			result := extractRunLines(content)
			Expect(result).To(Equal(expected))
		},

		Entry("inline run: value",
			`      - run: npm install`,
			[]string{"npm install"},
		),

		Entry("inline run: with quotes stripped",
			`      - run: "npm install --save-dev"`,
			[]string{"npm install --save-dev"},
		),

		Entry("block scalar run: |",
			`    - name: Install deps
      run: |
        npm install
        pip install -r requirements.txt`,
			[]string{"npm install", "pip install -r requirements.txt"},
		),

		Entry("block scalar run: >",
			`    - run: >
        yarn install
        --frozen-lockfile`,
			[]string{"yarn install", "--frozen-lockfile"},
		),

		Entry("name: key is not extracted",
			`    - name: npm install
      run: echo hello`,
			[]string{"echo hello"},
		),

		Entry("mixed workflow — only run: lines returned",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: npm install
        run: echo 'nothing here'
      - name: Real install
        run: |
          npm ci --ignore-scripts
          pip install --no-cache-dir -r requirements.txt`,
			[]string{"echo 'nothing here'", "npm ci --ignore-scripts", "pip install --no-cache-dir -r requirements.txt"},
		),

		Entry("no run: fields returns empty slice",
			`name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3`,
			nil,
		),
	)
})

var _ = Describe("shell command detection", func() {
	DescribeTable("checks mitigation flags per command segment",
		func(line string, npmCi, yarnInstall, pipNoCache, pipHashes bool) {
			Expect(isNpmCiWithoutIgnoreScripts(line)).To(Equal(npmCi))
			Expect(isYarnInstallWithoutFrozen(line)).To(Equal(yarnInstall))
			Expect(isPipInstallWithoutNoCache(line)).To(Equal(pipNoCache))
			Expect(isPipInstallWithoutHashes(line)).To(Equal(pipHashes))
		},
		Entry("npm ci is still flagged when another segment has ignore-scripts",
			"npm install --ignore-scripts && npm ci",
			true, false, false, false,
		),
		Entry("yarn install is still flagged when another segment has frozen-lockfile",
			"echo --frozen-lockfile && yarn install",
			false, true, false, false,
		),
		Entry("pip install is still flagged when another segment has no-cache-dir",
			"echo --no-cache-dir && pip install requests",
			false, false, true, true,
		),
		Entry("pip install with both mitigations in the same segment is not flagged",
			"pip install --no-cache-dir --require-hashes -r requirements.txt",
			false, false, false, false,
		),
	)

	DescribeTable("detects npx execution",
		func(line string, expected bool) {
			Expect(isNpxUsage(line)).To(Equal(expected))
		},
		Entry("simple npx command", "npx tsx script.ts", true),
		Entry("npx after chained build", "npm test && npx eslint .", true),
		Entry("echoing npx is ignored", "echo npx", false),
		Entry("quoted npx string is ignored", "echo \"npx tsx script.ts\"", false),
		Entry("pnpx is ignored", "pnpx prisma generate", false),
	)

	DescribeTable("detects sudo across chained commands",
		func(line string, expected bool) {
			Expect(isSudo(line)).To(Equal(expected))
		},
		Entry("sudo at start", "sudo apt-get install -y git", true),
		Entry("sudo after semicolon", "echo hi;sudo apt-get install -y git", true),
		Entry("sudo after logical and without extra spaces", "echo hi&&sudo apt-get install -y git", true),
		Entry("sudo after logical or", "false||sudo apt-get install -y git", true),
		Entry("sudo after pipe", "cat file |sudo tee /tmp/file", true),
		Entry("no sudo present", "apt-get install -y git", false),
	)

	DescribeTable("detects npm-family package publishing",
		func(line string, expected bool) {
			Expect(isPackagePublish(line)).To(Equal(expected))
		},
		Entry("npm publish", "npm publish", true),
		Entry("pnpm publish", "pnpm publish --access public", true),
		Entry("yarn publish", "yarn publish", true),
		Entry("yarn npm publish", "yarn npm publish", true),
		Entry("publish after chained build", "npm run build && npm publish", true),
		Entry("pnpm recursive publish", "pnpm -r publish", false),
		Entry("npm publish dry run is ignored", "npm publish --dry-run", false),
		Entry("npm run publish script is ignored", "npm run publish", false),
		Entry("echoing npm publish is ignored", "echo \"npm publish\"", false),
	)
})
