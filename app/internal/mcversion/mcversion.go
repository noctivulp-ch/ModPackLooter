// Package mcversion parses and compares Minecraft Java Edition versions,
// covering both the classic scheme (1.20.1) and the yearly one (26.3).
package mcversion

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a Minecraft release version such as 1.20.1 or 26.3.
type Version struct {
	parts [3]int
}

// Parse reads a version like "1.20.1", "1.21" or "26.3".
func Parse(s string) (Version, error) {
	fields := strings.Split(strings.TrimSpace(s), ".")
	if len(fields) < 2 || len(fields) > 3 {
		return Version{}, fmt.Errorf("versión de Minecraft no válida: %q", s)
	}
	var v Version
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("versión de Minecraft no válida: %q", s)
		}
		v.parts[i] = n
	}
	return v, nil
}

// MustParse is Parse for constants known to be valid; it panics otherwise.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns -1, 0 or 1 when v is lower, equal or greater than o.
func (v Version) Compare(o Version) int {
	for i := range v.parts {
		switch {
		case v.parts[i] < o.parts[i]:
			return -1
		case v.parts[i] > o.parts[i]:
			return 1
		}
	}
	return 0
}

func (v Version) String() string {
	if v.parts[2] == 0 {
		return fmt.Sprintf("%d.%d", v.parts[0], v.parts[1])
	}
	return fmt.Sprintf("%d.%d.%d", v.parts[0], v.parts[1], v.parts[2])
}

// Range is a set of constraints such as ">=1.20.1 <1.21". The zero value
// matches every version.
type Range struct {
	constraints []constraint
	raw         string
}

type constraint struct {
	op      string
	version Version
}

// ParseRange reads space-separated constraints using >=, >, <=, < or =.
func ParseRange(s string) (Range, error) {
	r := Range{raw: strings.TrimSpace(s)}
	for _, field := range strings.Fields(s) {
		op := ""
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(field, candidate) {
				op = candidate
				break
			}
		}
		if op == "" {
			return Range{}, fmt.Errorf("restricción de versión sin operador: %q", field)
		}
		v, err := Parse(strings.TrimPrefix(field, op))
		if err != nil {
			return Range{}, err
		}
		r.constraints = append(r.constraints, constraint{op: op, version: v})
	}
	return r, nil
}

// MustParseRange is ParseRange for constants known to be valid.
func MustParseRange(s string) Range {
	r, err := ParseRange(s)
	if err != nil {
		panic(err)
	}
	return r
}

// Contains reports whether v satisfies every constraint of the range.
func (r Range) Contains(v Version) bool {
	for _, c := range r.constraints {
		cmp := v.Compare(c.version)
		ok := false
		switch c.op {
		case ">=":
			ok = cmp >= 0
		case ">":
			ok = cmp > 0
		case "<=":
			ok = cmp <= 0
		case "<":
			ok = cmp < 0
		case "=":
			ok = cmp == 0
		}
		if !ok {
			return false
		}
	}
	return true
}

func (r Range) String() string {
	if r.raw == "" {
		return "*"
	}
	return r.raw
}
