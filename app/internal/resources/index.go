package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

// Entry is one file of one pack.
type Entry struct {
	Pack Pack
	Path string
}

func (e Entry) String() string { return e.Pack.Name() + "!/" + e.Path }

// Read returns the content of the entry.
func (e Entry) Read() ([]byte, error) { return ReadFile(e.Pack, e.Path) }

// ReadJSON decodes the entry into v.
func (e Entry) ReadJSON(v any) error {
	data, err := e.Read()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(stripBOM(data), v); err != nil {
		return fmt.Errorf("%s: %w", e, err)
	}
	return nil
}

func stripBOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
}

// Index is the merged, version-normalised view of every pack.
type Index struct {
	layout   *Layout
	data     map[string]map[domain.ResourceID][]Entry // type -> id -> entries, lowest priority first
	lang     map[string][]Entry                       // language code -> lang files, lowest priority first
	packs    []Pack
	override map[string]int // type -> number of ids overridden by a later pack
}

// NewIndex merges the packs, given from lowest to highest priority.
func NewIndex(layout *Layout, packs []Pack) *Index {
	ix := &Index{
		layout:   layout,
		data:     map[string]map[domain.ResourceID][]Entry{},
		lang:     map[string][]Entry{},
		packs:    packs,
		override: map[string]int{},
	}
	for _, p := range packs {
		for _, path := range p.Files() {
			if code, ok := langCode(path); ok {
				ix.lang[code] = append(ix.lang[code], Entry{Pack: p, Path: path})
				continue
			}
			typ, ns, id, ok := layout.classify(path)
			if !ok {
				continue
			}
			rid := domain.ResourceID{Namespace: ns, Path: id}
			byID := ix.data[typ]
			if byID == nil {
				byID = map[domain.ResourceID][]Entry{}
				ix.data[typ] = byID
			}
			if len(byID[rid]) > 0 {
				ix.override[typ]++
			}
			byID[rid] = append(byID[rid], Entry{Pack: p, Path: path})
		}
	}
	return ix
}

// langCode matches "assets/<ns>/lang/<code>.json".
func langCode(path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, "assets/")
	if !ok {
		return "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[1] != "lang" || !strings.HasSuffix(parts[2], ".json") {
		return "", false
	}
	return strings.ToLower(strings.TrimSuffix(parts[2], ".json")), true
}

// Layout returns the layout used to normalise the packs.
func (ix *Index) Layout() *Layout { return ix.layout }

// Packs returns the merged packs, lowest priority first.
func (ix *Index) Packs() []Pack { return ix.packs }

// IDs lists every id of a type, sorted.
func (ix *Index) IDs(typ string) []domain.ResourceID {
	byID := ix.data[typ]
	out := make([]domain.ResourceID, 0, len(byID))
	for id := range byID {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

// Lookup returns the effective (highest priority) file of an id.
func (ix *Index) Lookup(typ string, id domain.ResourceID) (Entry, bool) {
	entries := ix.data[typ][id]
	if len(entries) == 0 {
		return Entry{}, false
	}
	return entries[len(entries)-1], true
}

// All returns every file of an id, lowest priority first. Tags need them all
// because they merge instead of overriding.
func (ix *Index) All(typ string, id domain.ResourceID) []Entry {
	return ix.data[typ][id]
}

// Count returns how many ids a type has.
func (ix *Index) Count(typ string) int { return len(ix.data[typ]) }

// Overrides returns how many ids of a type were replaced by a later pack.
func (ix *Index) Overrides(typ string) int { return ix.override[typ] }

// Languages lists the language codes found.
func (ix *Index) Languages() []string {
	out := make([]string, 0, len(ix.lang))
	for code := range ix.lang {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// Lang merges every lang file of a language; later packs win per key.
// Unreadable files are reported through onError and skipped.
func (ix *Index) Lang(code string, onError func(Entry, error)) map[string]string {
	out := map[string]string{}
	for _, e := range ix.lang[strings.ToLower(code)] {
		var m map[string]any
		if err := e.ReadJSON(&m); err != nil {
			if onError != nil {
				onError(e, err)
			}
			continue
		}
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}
