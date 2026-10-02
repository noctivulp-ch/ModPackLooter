package nbt

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func gzipBytes(data []byte) []byte {
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	zw.Write(data)
	zw.Close()
	return b.Bytes()
}

func TestDecodeStructureLikeDocument(t *testing.T) {
	doc := Compound{
		"DataVersion": int32(3465),
		"size":        List{int32(3), int32(2), int32(3)},
		"palette":     List{Compound{"Name": "minecraft:chest", "Properties": Compound{"facing": "north"}}},
		"blocks": List{Compound{
			"pos":   List{int32(1), int32(0), int32(1)},
			"state": int32(0),
			"nbt":   Compound{"id": "minecraft:chest", "LootTable": "minecraft:chests/simple_dungeon", "LootTableSeed": int64(42)},
		}},
		"entities": List{},
		"misc":     Compound{"b": int8(-1), "s": int16(7), "f": float32(1.5), "d": 2.25, "ba": []byte{1, 2}, "ia": []int32{1, 2}, "la": []int64{3}},
	}
	for name, data := range map[string][]byte{"plano": Encode(doc), "gzip": gzipBytes(Encode(doc))} {
		got, err := Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		block := got.List("blocks")[0].(Compound)
		if lt := block.Compound("nbt").String("LootTable"); lt != "minecraft:chests/simple_dungeon" {
			t.Errorf("%s: LootTable = %q", name, lt)
		}
		if v, ok := got.Int("DataVersion"); !ok || v != 3465 {
			t.Errorf("%s: DataVersion = %v", name, v)
		}
		if got.Compound("misc")["f"].(float32) != 1.5 || len(got.List("entities")) != 0 {
			t.Errorf("%s: valores mal decodificados: %v", name, got)
		}
	}
}

func TestDecodeRejectsCorruptData(t *testing.T) {
	good := Encode(Compound{"a": "hola", "l": List{int32(1), int32(2)}})
	for i := 1; i < len(good); i++ {
		if _, err := Decode(good[:i]); err == nil {
			t.Fatalf("datos truncados a %d bytes deberían fallar", i)
		}
	}
	huge := []byte{tagCompound, 0, 0, tagIntArray, 0, 1, 'x', 0x7f, 0xff, 0xff, 0xff}
	if _, err := Decode(huge); err == nil {
		t.Error("una longitud enorme debe rechazarse sin reservar memoria")
	}
}

func TestDecodeGzipWithoutTrailer(t *testing.T) {
	full := gzipBytes(Encode(Compound{"LootTable": "minecraft:chests/x"}))
	// Drop the CRC32 and size trailer, as some structure editors do.
	got, err := Decode(full[:len(full)-8])
	if err != nil || got.String("LootTable") != "minecraft:chests/x" {
		t.Fatalf("gzip sin cola: %v %v", got, err)
	}
	if _, err := Decode(full[:len(full)/2]); err == nil {
		t.Error("un documento realmente cortado debe seguir fallando")
	}
}
