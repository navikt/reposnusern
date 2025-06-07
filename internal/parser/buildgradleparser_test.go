package parser

import (
	"testing"
)

func TestParseGradleGroovy(t *testing.T) {
	input := []byte(`
		ext.avroVersion = '1.8.2'
		ext.pluginVersion = "0.14.0"

		dependencies {
			implementation "org.apache.avro:avro:${avroVersion}"
			classpath "com.commercehub.gradle.plugin:gradle-avro-plugin:${pluginVersion}"
		}`)

	expected := []Dependency{
		{Group: "org.apache.avro", Name: "avro", Version: "1.8.2", Type: "gradle", Path: "build.gradle"},
		{Group: "com.commercehub.gradle.plugin", Name: "gradle-avro-plugin", Version: "0.14.0", Type: "gradle", Path: "build.gradle"},
	}

	parser := GradleGroovyParser{}
	deps, err := parser.ParseFile("build.gradle", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d deps, got %d", len(expected), len(deps))
	}

	for i := range deps {
		if deps[i] != expected[i] {
			t.Errorf("mismatch at %d:\n  got  %+v\n  want %+v", i, deps[i], expected[i])
		}
	}
}
