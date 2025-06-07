package parser

import (
	"testing"
)

func TestParseYarnLock(t *testing.T) {
	input := []byte(`
"@babel/code-frame@^7.0.0":
  version "7.10.4"
  resolved "https://registry.yarnpkg.com/@babel/code-frame/-/code-frame-7.10.4.tgz"
  integrity sha512-blabla

"debug@^4.1.1", "debug@^4.3.1":
  version "4.3.4"
  resolved "https://registry.yarnpkg.com/debug/-/debug-4.3.4.tgz"
  integrity sha512-moreblabla
`)

	expected := []Dependency{
		{Name: "@babel/code-frame", Version: "7.10.4", Type: "yarn", Path: "yarn.lock"},
		{Name: "debug", Version: "4.3.4", Type: "yarn", Path: "yarn.lock"},
	}

	deps, err := YarnLockParser{}.ParseFile("yarn.lock", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d deps, got %d", len(expected), len(deps))
	}

	for i := range deps {
		if deps[i] != expected[i] {
			t.Errorf("mismatch at %d: got %+v, want %+v", i, deps[i], expected[i])
		}
	}
}
