// Package testkit builds small, synthetic modpack instances on disk for tests.
package testkit

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Files maps archive or folder paths to their content.
type Files map[string]string

// Instance is a modpack folder under a temporary directory.
type Instance struct {
	t    testing.TB
	Root string
}

// NewInstance creates an empty instance with a mods/ folder.
func NewInstance(t testing.TB) *Instance {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Instance{t: t, Root: root}
}

// Jar writes a zip archive with the given files at rel (relative to Root).
func (in *Instance) Jar(rel string, files Files) string {
	in.t.Helper()
	path := filepath.Join(in.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		in.t.Fatal(err)
	}
	if err := os.WriteFile(path, ZipBytes(in.t, files), 0o644); err != nil {
		in.t.Fatal(err)
	}
	return path
}

// Dir writes the files as a folder tree at rel (relative to Root).
func (in *Instance) Dir(rel string, files Files) string {
	in.t.Helper()
	base := filepath.Join(in.Root, filepath.FromSlash(rel))
	for name, content := range files {
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			in.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			in.t.Fatal(err)
		}
	}
	return base
}

// ZipBytes builds a zip archive in memory.
func ZipBytes(t testing.TB, files Files) []byte {
	t.Helper()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[n])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// ModsToml returns a minimal Forge mods.toml for one mod targeting mcRange.
func ModsToml(modID, name, mcRange string) string {
	return `modLoader="javafml"
loaderVersion="[47,)"
license="MIT"
[[mods]]
modId="` + modID + `"
displayName="` + name + `"
version="1.0.0"
[[dependencies.` + modID + `]]
modId="minecraft"
mandatory=true
versionRange="` + mcRange + `"
ordering="NONE"
side="BOTH"
`
}
