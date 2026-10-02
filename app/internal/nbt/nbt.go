// Package nbt decodes Minecraft's Named Binary Tag format (Java Edition,
// big-endian), as used by structure templates. Gzip and zlib compressed
// input is detected automatically.
package nbt

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Compound is an NBT compound tag.
type Compound map[string]any

// List is an NBT list tag.
type List []any

const (
	tagEnd byte = iota
	tagByte
	tagShort
	tagInt
	tagLong
	tagFloat
	tagDouble
	tagByteArray
	tagString
	tagList
	tagCompound
	tagIntArray
	tagLongArray
)

const maxDepth = 512

// Decode reads a (possibly compressed) NBT document and returns its root compound.
func Decode(data []byte) (Compound, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	d := &decoder{buf: raw}
	typ, err := d.byte()
	if err != nil {
		return nil, err
	}
	if typ != tagCompound {
		return nil, fmt.Errorf("nbt: la raíz debe ser un compound, es el tipo %d", typ)
	}
	if _, err := d.string(); err != nil { // root name, usually empty
		return nil, err
	}
	v, err := d.payload(tagCompound, 0)
	if err != nil {
		return nil, err
	}
	return v.(Compound), nil
}

func decompress(data []byte) ([]byte, error) {
	switch {
	case len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b:
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("nbt: gzip: %w", err)
		}
		defer zr.Close()
		return io.ReadAll(zr)
	case len(data) >= 2 && data[0] == 0x78:
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("nbt: zlib: %w", err)
		}
		defer zr.Close()
		return io.ReadAll(zr)
	default:
		return data, nil
	}
}

var errShort = errors.New("nbt: datos truncados")

type decoder struct {
	buf []byte
	pos int
}

func (d *decoder) take(n int) ([]byte, error) {
	if n < 0 || d.pos+n > len(d.buf) {
		return nil, errShort
	}
	b := d.buf[d.pos : d.pos+n]
	d.pos += n
	return b, nil
}

func (d *decoder) byte() (byte, error) {
	b, err := d.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *decoder) u16() (uint16, error) {
	b, err := d.take(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func (d *decoder) i32() (int32, error) {
	b, err := d.take(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b)), nil
}

func (d *decoder) i64() (int64, error) {
	b, err := d.take(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b)), nil
}

func (d *decoder) string() (string, error) {
	n, err := d.u16()
	if err != nil {
		return "", err
	}
	b, err := d.take(int(n))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// length reads an array/list length and checks it fits in the remaining data.
func (d *decoder) length(elemSize int) (int, error) {
	n, err := d.i32()
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("nbt: longitud negativa %d", n)
	}
	if elemSize > 0 && int(n) > (len(d.buf)-d.pos)/elemSize {
		return 0, errShort
	}
	return int(n), nil
}

func (d *decoder) payload(typ byte, depth int) (any, error) {
	if depth > maxDepth {
		return nil, errors.New("nbt: anidamiento demasiado profundo")
	}
	switch typ {
	case tagByte:
		b, err := d.byte()
		return int8(b), err
	case tagShort:
		v, err := d.u16()
		return int16(v), err
	case tagInt:
		return d.i32()
	case tagLong:
		return d.i64()
	case tagFloat:
		v, err := d.i32()
		return math.Float32frombits(uint32(v)), err
	case tagDouble:
		v, err := d.i64()
		return math.Float64frombits(uint64(v)), err
	case tagByteArray:
		n, err := d.length(1)
		if err != nil {
			return nil, err
		}
		b, err := d.take(n)
		return append([]byte(nil), b...), err
	case tagString:
		return d.string()
	case tagList:
		elem, err := d.byte()
		if err != nil {
			return nil, err
		}
		n, err := d.length(1)
		if err != nil {
			return nil, err
		}
		if elem == tagEnd {
			// Empty lists are stored with element type End.
			n = 0
		}
		list := make(List, 0, n)
		for i := 0; i < n; i++ {
			v, err := d.payload(elem, depth+1)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		return list, nil
	case tagCompound:
		c := Compound{}
		for {
			t, err := d.byte()
			if err != nil {
				return nil, err
			}
			if t == tagEnd {
				return c, nil
			}
			name, err := d.string()
			if err != nil {
				return nil, err
			}
			v, err := d.payload(t, depth+1)
			if err != nil {
				return nil, err
			}
			c[name] = v
		}
	case tagIntArray:
		n, err := d.length(4)
		if err != nil {
			return nil, err
		}
		out := make([]int32, n)
		for i := range out {
			if out[i], err = d.i32(); err != nil {
				return nil, err
			}
		}
		return out, nil
	case tagLongArray:
		n, err := d.length(8)
		if err != nil {
			return nil, err
		}
		out := make([]int64, n)
		for i := range out {
			if out[i], err = d.i64(); err != nil {
				return nil, err
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("nbt: tipo de tag desconocido %d", typ)
	}
}

// String returns a string field, or "" if missing or of another type.
func (c Compound) String(key string) string {
	s, _ := c[key].(string)
	return s
}

// Compound returns a nested compound, or nil.
func (c Compound) Compound(key string) Compound {
	v, _ := c[key].(Compound)
	return v
}

// List returns a list field, or nil.
func (c Compound) List(key string) List {
	v, _ := c[key].(List)
	return v
}

// Int returns any integer field as int.
func (c Compound) Int(key string) (int, bool) {
	switch v := c[key].(type) {
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	}
	return 0, false
}
