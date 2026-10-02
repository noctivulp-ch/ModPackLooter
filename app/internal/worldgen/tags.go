// Package worldgen reads structures, template pools, processor lists,
// structure templates and tags from the merged resource index.
package worldgen

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/resources"
)

// Tags resolves tags of one type, merging every pack's file like the game
// does (unless a later file sets "replace": true).
type Tags struct {
	ix    *resources.Index
	typ   string
	cache map[domain.ResourceID][]domain.ResourceID
	diags *domain.Diagnostics
}

// NewTags creates a resolver for a tag type such as resources.TypeBiomeTag.
func NewTags(ix *resources.Index, typ string, diags *domain.Diagnostics) *Tags {
	return &Tags{ix: ix, typ: typ, cache: map[domain.ResourceID][]domain.ResourceID{}, diags: diags}
}

type rawTag struct {
	Replace bool              `json:"replace"`
	Values  []json.RawMessage `json:"values"`
}

// Resolve returns every element of the tag, expanding nested tags, sorted.
func (t *Tags) Resolve(id domain.ResourceID) []domain.ResourceID {
	if v, ok := t.cache[id]; ok {
		return v
	}
	t.cache[id] = nil // cycle guard
	set := map[domain.ResourceID]bool{}
	for _, e := range t.ix.All(t.typ, id) {
		var raw rawTag
		if err := e.ReadJSON(&raw); err != nil {
			if t.diags != nil {
				t.diags.Add(domain.LevelWarning, "parser", e.String(), "tag ilegible: %v", err)
			}
			continue
		}
		if raw.Replace {
			set = map[domain.ResourceID]bool{}
		}
		for _, v := range raw.Values {
			ref := tagValue(v)
			if ref == "" {
				continue
			}
			if nested, ok := strings.CutPrefix(ref, "#"); ok {
				nid, err := domain.ParseResourceID(nested)
				if err != nil {
					continue
				}
				for _, x := range t.Resolve(nid) {
					set[x] = true
				}
				continue
			}
			if rid, err := domain.ParseResourceID(ref); err == nil {
				set[rid] = true
			}
		}
	}
	out := make([]domain.ResourceID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	t.cache[id] = out
	return out
}

// tagValue reads either "id" or {"id": "…", "required": false}.
func tagValue(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.ID
	}
	return ""
}

// HolderSet resolves a field that may be an id, a "#tag" or a list of ids.
func (t *Tags) HolderSet(raw json.RawMessage) []domain.ResourceID {
	if len(raw) == 0 {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		var single string
		if err := json.Unmarshal(raw, &single); err != nil {
			return nil
		}
		list = []string{single}
	}
	set := map[domain.ResourceID]bool{}
	for _, ref := range list {
		if tag, ok := strings.CutPrefix(ref, "#"); ok {
			if tid, err := domain.ParseResourceID(tag); err == nil {
				for _, x := range t.Resolve(tid) {
					set[x] = true
				}
			}
			continue
		}
		if rid, err := domain.ParseResourceID(ref); err == nil {
			set[rid] = true
		}
	}
	out := make([]domain.ResourceID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
