package parser

import (
	"testing"
)

func TestParseSinglePomFile(t *testing.T) {
	xmlData := []byte(`
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>demo</artifactId>
  <version>1.2.3</version>
  <properties>
    <some.version>4.5.6</some.version>
  </properties>
  <dependencies>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>lib-a</artifactId>
      <version>${some.version}</version>
    </dependency>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>lib-b</artifactId>
      <version>${project.version}</version>
    </dependency>
  </dependencies>
</project>
`)

	deps, err := ParseSinglePomFile(xmlData)
	if err != nil {
		t.Fatalf("ParseSinglePomFile failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	expected := map[string]string{
		"lib-a": "4.5.6",
		"lib-b": "1.2.3",
	}
	for _, dep := range deps {
		want := expected[dep.Name]
		if dep.Version != want {
			t.Errorf("dependency %s: expected version %s, got %s", dep.Name, want, dep.Version)
		}
	}
}

func TestParseRepoPomFiles(t *testing.T) {
	files := map[string][]byte{
		"a.xml": []byte(`
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <version>1.0.0</version>
  <properties>
    <lib.version>9.9.9</lib.version>
  </properties>
  <dependencies>
    <dependency>
      <groupId>org.repo</groupId>
      <artifactId>core</artifactId>
      <version>${lib.version}</version>
    </dependency>
  </dependencies>
</project>`),
		"b.xml": []byte(`
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <version>2.0.0</version>
  <dependencies>
    <dependency>
      <groupId>org.repo</groupId>
      <artifactId>extra</artifactId>
      <version>${lib.version}</version>
    </dependency>
  </dependencies>
</project>`),
	}

	deps, err := ParseRepoPomFiles(files)
	if err != nil {
		t.Fatalf("ParseRepoPomFiles failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	expected := map[string]string{
		"core":  "9.9.9",
		"extra": "9.9.9",
	}
	for _, dep := range deps {
		want := expected[dep.Name]
		if dep.Version != want {
			t.Errorf("dependency %s: expected version %s, got %s", dep.Name, want, dep.Version)
		}
		if dep.Path == "" {
			t.Errorf("dependency %s: missing path metadata", dep.Name)
		}
	}
}

func TestParseSinglePomFile_EdgeCases(t *testing.T) {
	xmlData := []byte(`
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>demo</artifactId>
  <version>${a}${b}</version>
  <properties>
    <a>1.2</a>
    <!-- b er ikke definert -->
    <known>3.3.3</known>
  </properties>
  <dependencies>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>with-version</artifactId>
      <version>${known}</version>
    </dependency>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>no-version</artifactId>
    </dependency>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>unknown-var</artifactId>
      <version>${missing}</version>
    </dependency>
    <dependency>
      <groupId>org.test</groupId>
      <artifactId>combined-vars</artifactId>
      <version>${a}${b}</version>
    </dependency>
  </dependencies>
</project>
`)

	deps, err := ParseSinglePomFile(xmlData)
	if err != nil {
		t.Fatalf("ParseSinglePomFile failed: %v", err)
	}

	expected := map[string]string{
		"with-version":  "3.3.3",
		"unknown-var":   "unparsed-version",
		"combined-vars": "unparsed-version",
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	for _, dep := range deps {
		want, ok := expected[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency: %s", dep.Name)
			continue
		}
		if dep.Version != want {
			t.Errorf("dependency %s: expected version %s, got %s", dep.Name, want, dep.Version)
		}
	}
}
