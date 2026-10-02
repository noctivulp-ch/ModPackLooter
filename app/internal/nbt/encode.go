package nbt

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Encode writes an uncompressed NBT document. It is used to build test
// fixtures; map iteration order makes the byte output non-deterministic.
func Encode(root Compound) []byte {
	var b bytes.Buffer
	b.WriteByte(tagCompound)
	writeString(&b, "")
	writePayload(&b, root)
	return b.Bytes()
}

func writeString(b *bytes.Buffer, s string) {
	_ = binary.Write(b, binary.BigEndian, uint16(len(s)))
	b.WriteString(s)
}

func tagOf(v any) byte {
	switch v.(type) {
	case int8:
		return tagByte
	case int16:
		return tagShort
	case int32:
		return tagInt
	case int64:
		return tagLong
	case float32:
		return tagFloat
	case float64:
		return tagDouble
	case []byte:
		return tagByteArray
	case string:
		return tagString
	case List:
		return tagList
	case Compound:
		return tagCompound
	case []int32:
		return tagIntArray
	case []int64:
		return tagLongArray
	}
	panic("tipo no soportado")
}

func writePayload(b *bytes.Buffer, v any) {
	switch x := v.(type) {
	case int8:
		b.WriteByte(byte(x))
	case int16, int32, int64:
		_ = binary.Write(b, binary.BigEndian, x)
	case float32:
		_ = binary.Write(b, binary.BigEndian, math.Float32bits(x))
	case float64:
		_ = binary.Write(b, binary.BigEndian, math.Float64bits(x))
	case []byte:
		_ = binary.Write(b, binary.BigEndian, int32(len(x)))
		b.Write(x)
	case string:
		writeString(b, x)
	case List:
		elem := tagEnd
		if len(x) > 0 {
			elem = tagOf(x[0])
		}
		b.WriteByte(elem)
		_ = binary.Write(b, binary.BigEndian, int32(len(x)))
		for _, e := range x {
			writePayload(b, e)
		}
	case Compound:
		for k, e := range x {
			b.WriteByte(tagOf(e))
			writeString(b, k)
			writePayload(b, e)
		}
		b.WriteByte(tagEnd)
	case []int32:
		_ = binary.Write(b, binary.BigEndian, int32(len(x)))
		_ = binary.Write(b, binary.BigEndian, x)
	case []int64:
		_ = binary.Write(b, binary.BigEndian, int32(len(x)))
		_ = binary.Write(b, binary.BigEndian, x)
	}
}
