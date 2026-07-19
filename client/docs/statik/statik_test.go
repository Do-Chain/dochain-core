package statik

import (
	"io"
	"strings"
	"testing"

	statikfs "github.com/rakyll/statik/fs"
	"gopkg.in/yaml.v2"
)

func TestEmbeddedSwaggerUsesDoChainBranding(t *testing.T) {
	assets, err := statikfs.NewWithNamespace("dochaind")
	if err != nil {
		t.Fatalf("load embedded Swagger assets: %v", err)
	}

	for path, expected := range map[string]string{
		"/index.html":         `href="do-chain-theme.css"`,
		"/do-chain-theme.css": "--do-accent: #a855f7",
	} {
		asset, err := assets.Open(path)
		if err != nil {
			t.Fatalf("open embedded %s: %v", path, err)
		}
		contents, err := io.ReadAll(asset)
		asset.Close()
		if err != nil {
			t.Fatalf("read embedded %s: %v", path, err)
		}
		if !strings.Contains(string(contents), expected) {
			t.Fatalf("embedded %s is missing %q", path, expected)
		}
	}

	swagger, err := assets.Open("/swagger.yaml")
	if err != nil {
		t.Fatalf("open embedded swagger.yaml: %v", err)
	}
	defer swagger.Close()

	contents, err := io.ReadAll(swagger)
	if err != nil {
		t.Fatalf("read embedded swagger.yaml: %v", err)
	}

	document := string(contents)
	for _, expected := range []string{
		"title: Do-Chain - gRPC Gateway docs",
		"description: REST interface for Do-Chain",
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("embedded swagger.yaml is missing %q", expected)
		}
	}

	if strings.Contains(document, "title: Terra Classic - gRPC Gateway docs") {
		t.Fatal("embedded swagger.yaml still uses the Terra Classic title")
	}

	var specification struct {
		Swagger string                 `yaml:"swagger"`
		Paths   map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(contents, &specification); err != nil {
		t.Fatalf("parse embedded swagger.yaml: %v", err)
	}
	if specification.Swagger != "2.0" {
		t.Fatalf("unexpected Swagger version %q", specification.Swagger)
	}
	if _, ok := specification.Paths["/do/market/v1beta1/params"]; !ok {
		t.Fatal("embedded swagger.yaml is missing the Do Chain market API")
	}
	for path := range specification.Paths {
		if strings.HasPrefix(path, "/terra/") {
			t.Fatalf("embedded swagger.yaml still exposes legacy Terra route %q", path)
		}
	}
}
