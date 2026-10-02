// Package resources reads the files of a modpack (vanilla jar, mod jars,
// datapacks, script folders) and merges them into one index with the same
// "last one wins" rule the game uses.
package resources

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackKind says where a pack comes from. Packs of a higher kind override
// lower ones, mirroring the game's load order.
type PackKind int

const (
	KindVanilla PackKind = iota
	KindMod
	KindDatapack
	KindScript
)

func (k PackKind) String() string {
	switch k {
	case KindVanilla:
		return "vanilla"
	case KindMod:
		return "mod"
	case KindDatapack:
		return "datapack"
	case KindScript:
		return "script"
	default:
		return "desconocido"
	}
}

// Pack is a source of game files with paths like "data/minecraft/loot_tables/…".
type Pack interface {
	Name() string
	Kind() PackKind
	// Files lists every regular file path, using "/" as separator.
	Files() []string
	Open(path string) (io.ReadCloser, error)
}

// ReadFile reads a whole file from a pack.
func ReadFile(p Pack, path string) ([]byte, error) {
	rc, err := p.Open(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// ZipPack is a pack backed by a jar or zip archive.
type ZipPack struct {
	name  string
	kind  PackKind
	zr    *zip.Reader
	files map[string]*zip.File
	order []string
	close func() error
}

// OpenZipPack opens a jar or zip file from disk.
func OpenZipPack(path string, kind PackKind) (*ZipPack, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %s: %w", path, err)
	}
	p := newZipPack(filepath.Base(path), kind, &rc.Reader)
	p.close = rc.Close
	return p, nil
}

// NewZipPackFromBytes opens an archive held in memory (used for jar-in-jar).
func NewZipPackFromBytes(name string, kind PackKind, data []byte) (*ZipPack, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %s: %w", name, err)
	}
	return newZipPack(name, kind, zr), nil
}

func newZipPack(name string, kind PackKind, zr *zip.Reader) *ZipPack {
	p := &ZipPack{name: name, kind: kind, zr: zr, files: map[string]*zip.File{}}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean := strings.TrimPrefix(f.Name, "/")
		if _, dup := p.files[clean]; dup {
			continue
		}
		p.files[clean] = f
		p.order = append(p.order, clean)
	}
	sort.Strings(p.order)
	return p
}

func (p *ZipPack) Name() string    { return p.name }
func (p *ZipPack) Kind() PackKind  { return p.kind }
func (p *ZipPack) Files() []string { return p.order }

func (p *ZipPack) Open(path string) (io.ReadCloser, error) {
	f, ok := p.files[path]
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return f.Open()
}

// Close releases the underlying file, if any.
func (p *ZipPack) Close() error {
	if p.close != nil {
		return p.close()
	}
	return nil
}

// DirPack is a pack backed by a folder, such as an unzipped datapack or the
// kubejs folder.
type DirPack struct {
	name  string
	kind  PackKind
	root  string
	order []string
}

// OpenDirPack indexes every file under root.
func OpenDirPack(root string, kind PackKind) (*DirPack, error) {
	p := &DirPack{name: filepath.Base(root), kind: kind, root: root}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			p.order = append(p.order, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la carpeta %s: %w", root, err)
	}
	sort.Strings(p.order)
	return p, nil
}

// WithName overrides the display name of the pack.
func (p *DirPack) WithName(name string) *DirPack { p.name = name; return p }

func (p *DirPack) Name() string    { return p.name }
func (p *DirPack) Kind() PackKind  { return p.kind }
func (p *DirPack) Files() []string { return p.order }

func (p *DirPack) Open(path string) (io.ReadCloser, error) {
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("ruta no válida: %s", path)
	}
	return os.Open(filepath.Join(p.root, filepath.FromSlash(path)))
}

// MappedPack exposes individual files from anywhere on disk under virtual
// pack paths (used for the launcher's hashed asset objects).
type MappedPack struct {
	name  string
	kind  PackKind
	files map[string]string // pack path -> real path
	order []string
}

// NewMappedPack creates a pack from a map of pack paths to real files.
func NewMappedPack(name string, kind PackKind, files map[string]string) *MappedPack {
	p := &MappedPack{name: name, kind: kind, files: files}
	for k := range files {
		p.order = append(p.order, k)
	}
	sort.Strings(p.order)
	return p
}

func (p *MappedPack) Name() string    { return p.name }
func (p *MappedPack) Kind() PackKind  { return p.kind }
func (p *MappedPack) Files() []string { return p.order }

func (p *MappedPack) Open(path string) (io.ReadCloser, error) {
	real, ok := p.files[path]
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return os.Open(real)
}
