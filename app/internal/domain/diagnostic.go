package domain

import (
	"fmt"
	"sort"
	"sync"
)

// Level is the severity of a diagnostic.
type Level int

const (
	LevelInfo Level = iota
	LevelWarning
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelWarning:
		return "aviso"
	default:
		return "info"
	}
}

// MarshalText makes levels readable in JSON reports.
func (l Level) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

// Diagnostic is a non-fatal problem or note found while analysing a modpack.
type Diagnostic struct {
	Level   Level  `json:"level"`
	Stage   string `json:"stage"`            // loader, parser, discovery…
	Source  string `json:"source,omitempty"` // file or pack that caused it
	Message string `json:"message"`
}

// Diagnostics collects diagnostics safely from several goroutines.
type Diagnostics struct {
	mu    sync.Mutex
	items []Diagnostic
}

// Add records a diagnostic. It is a no-op on a nil receiver.
func (d *Diagnostics) Add(level Level, stage, source, format string, args ...any) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items = append(d.items, Diagnostic{Level: level, Stage: stage, Source: source, Message: fmt.Sprintf(format, args...)})
}

// Items returns the diagnostics sorted by severity (worst first), then stage and source.
func (d *Diagnostics) Items() []Diagnostic {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := append([]Diagnostic(nil), d.items...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level > out[j].Level
		}
		if out[i].Stage != out[j].Stage {
			return out[i].Stage < out[j].Stage
		}
		return out[i].Source < out[j].Source
	})
	return out
}

// Count returns how many diagnostics have the given level.
func (d *Diagnostics) Count(level Level) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, it := range d.items {
		if it.Level == level {
			n++
		}
	}
	return n
}
