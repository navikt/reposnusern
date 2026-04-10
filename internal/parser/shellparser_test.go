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
