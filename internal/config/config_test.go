package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	f, err := os.Open(filepath.Join(RootPath, "config", "application.yml"))
	if err != nil {
		t.Fatalf("error opening config file: %+v", err)
	}
	defer f.Close()

	v := &configContainer{}
	if err := yaml.NewDecoder(preprocess(f)).Decode(v); err != nil {
		t.Fatalf("error unmarshaling: %+v", err)
	}
}
