package world

import "github.com/EnierAragon/ModPackLooter/app/internal/nbt"

// Sample returns the Data compound of a small synthetic level.dat, for tests.
func Sample() nbt.Compound {
	point := func(biome string) nbt.Compound {
		return nbt.Compound{"biome": biome, "parameters": nbt.Compound{"temperature": float32(0)}}
	}
	return nbt.Compound{
		"LevelName": "Prueba",
		"Version":   nbt.Compound{"Name": "1.20.1"},
		"DataPacks": nbt.Compound{"Enabled": nbt.List{"vanilla", "mod:lootr"}, "Disabled": nbt.List{"file/extra.zip"}},
		"WorldGenSettings": nbt.Compound{"dimensions": nbt.Compound{
			"minecraft:overworld": nbt.Compound{"type": "minecraft:overworld", "generator": nbt.Compound{
				"type": "minecraft:noise", "settings": "minecraft:overworld",
				"biome_source": nbt.Compound{"type": "minecraft:multi_noise", "biomes": nbt.List{point("minecraft:plains"), point("minecraft:desert"), point("minecraft:plains")}},
			}},
			"minecraft:the_end": nbt.Compound{"type": "minecraft:the_end", "generator": nbt.Compound{
				"type": "minecraft:noise", "settings": "minecraft:end", "biome_source": nbt.Compound{"type": "minecraft:the_end"},
			}},
			"minecraft:the_nether": nbt.Compound{"type": "minecraft:the_nether", "generator": nbt.Compound{
				"type": "minecraft:noise", "settings": "minecraft:nether", "biome_source": nbt.Compound{"type": "minecraft:multi_noise", "preset": "minecraft:nether"},
			}},
		}},
	}
}
