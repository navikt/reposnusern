package parser

import (
	"testing"
)

func TestParseGradleKTS(t *testing.T) {
	input := `
val junitVersion = "5.9.1"
val log4jVersion by project

dependencies {
    implementation("org.junit.jupiter:junit-jupiter-api:$junitVersion")
    implementation("org.apache.logging.log4j:log4j-core:$log4jVersion")
    testImplementation("org.assertj:assertj-core:3.24.2")
}
`
	deps, err := ParseGradleKTS([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []Dependency{
		{Name: "junit-jupiter-api", Group: "org.junit.jupiter", Version: "5.9.1"},
		{Name: "log4j-core", Group: "org.apache.logging.log4j", Version: "unparsed-version"},
		{Name: "assertj-core", Group: "org.assertj", Version: "3.24.2"},
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d dependencies, got %d", len(expected), len(deps))
	}

	for i, dep := range deps {
		if dep.Name != expected[i].Name || dep.Group != expected[i].Group || dep.Version != expected[i].Version {
			t.Errorf("unexpected dep at index %d: got %+v, expected %+v", i, dep, expected[i])
		}
	}
}

func TestParseSingleBGKFile(t *testing.T) {
	input := `
val kotlinVersion = "1.9.0"

dependencies {
    implementation("org.jetbrains.kotlin:kotlin-stdlib:$kotlinVersion")
}`

	deps, err := ParseSingleBGKFile("some/path/build.gradle.kts", []byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.Name != "kotlin-stdlib" || dep.Group != "org.jetbrains.kotlin" || dep.Version != "1.9.0" {
		t.Errorf("unexpected dependency: %+v", dep)
	}
	if dep.Type != "gradle" {
		t.Errorf("expected Type 'gradle', got '%s'", dep.Type)
	}
	if dep.Path != "some/path/build.gradle.kts" {
		t.Errorf("unexpected Path: got '%s'", dep.Path)
	}
}
