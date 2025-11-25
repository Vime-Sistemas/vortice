package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv_ReadsFile(t *testing.T) {
	// create temp dir
	dir, err := ioutil.TempDir("", "envtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(dir)

	// write .env.local
	content := "FOO=bar\nBACKEND_URLS=http://localhost:9999"
	if err := ioutil.WriteFile(filepath.Join(dir, ".env.local"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	p := LoadEnv()
	if p == "" {
		t.Fatalf("expected env file to be loaded, got empty path")
	}

	if os.Getenv("FOO") != "bar" {
		t.Fatalf("expected FOO=bar, got %s", os.Getenv("FOO"))
	}
}
