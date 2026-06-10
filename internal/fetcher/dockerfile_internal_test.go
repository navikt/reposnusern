package fetcher

import "testing"

func TestIsDockerfile(t *testing.T) {
	testCases := map[string]struct {
		filename string
		want     bool
	}{
		"plain dockerfile": {
			filename: "Dockerfile",
			want:     true,
		},
		"dockerfile with suffix": {
			filename: "deploy/Dockerfile.prod",
			want:     true,
		},
		"prefixed dockerfile name": {
			filename: "docker/backend.Dockerfile",
			want:     true,
		},
		"dockerignore is excluded": {
			filename: ".dockerignore",
			want:     false,
		},
		"kotlin test file is excluded": {
			filename: "src/test/kotlin/no/nav/data/DockerfileFeaturesTest.kt",
			want:     false,
		},
	}

	for name, tc := range testCases {
		if got := isDockerfile(tc.filename); got != tc.want {
			t.Fatalf("%s: isDockerfile(%q) = %t, want %t", name, tc.filename, got, tc.want)
		}
	}
}
