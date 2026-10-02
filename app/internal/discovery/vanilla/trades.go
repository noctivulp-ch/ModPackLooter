package vanilla

import (
	"context"
	"strconv"

	"github.com/EnierAragon/ModPackLooter/app/internal/discovery"
	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
	"github.com/EnierAragon/ModPackLooter/app/internal/mcversion"
	"github.com/EnierAragon/ModPackLooter/app/internal/trades"
)

// TradesID is the ID of the vanilla trades discoverer.
const TradesID = "vanilla-trades"

// Trades1_20 attaches the villager and wandering trader trades of Minecraft
// 1.20.x. They are assigned by code (VillagerTrades), not data, so they are
// built-in knowledge: confidence "known". Each level offers two trades
// picked at random from its list; the wandering trader offers five common
// trades and one rare. Mod changes to trades are reported by the change
// detectors and shown next to these lists.
type Trades1_20 struct{}

func (Trades1_20) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: TradesID, Phase: discovery.PhaseSpecific, Priority: 40,
		Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.20 <1.21")}}
}

func (Trades1_20) Discover(_ context.Context, _ discovery.Input, out *discovery.Claims) error {
	out.Attach(trades.ExtraPrefix+"minecraft", villagers(false))
	return nil
}

// Trades1_21 is the 1.21 variant, checked against the 1.21 client: the
// same lists plus the cartographer's trial chambers map, and turtle_scute
// instead of scute. The experimental "villager trade rebalance" lists are
// not used by default and are not included.
type Trades1_21 struct{}

func (Trades1_21) Descriptor() discovery.Descriptor {
	return discovery.Descriptor{ID: TradesID, Phase: discovery.PhaseSpecific, Priority: 40,
		Applies: discovery.Applicability{Versions: mcversion.MustParseRange(">=1.21 <1.22")}}
}

func (Trades1_21) Discover(_ context.Context, _ discovery.Input, out *discovery.Claims) error {
	out.Attach(trades.ExtraPrefix+"minecraft", villagers(true))
	return nil
}

var levelNames = []string{"", "Novato", "Aprendiz", "Oficial", "Experto", "Maestro"}

// Trade builders, named after VillagerTrades' classes.
func emeraldFor(item string, count, uses, xp int) trades.Offer { // EmeraldForItems
	return trades.Offer{Buy: []trades.Stack{trades.Of(item, count)}, Sell: trades.Stack{Item: trades.Emerald, Count: 1}, MaxUses: uses, XP: xp}
}

func itemsFor(item string, emeralds, count, uses, xp int) trades.Offer { // ItemsForEmeralds
	return trades.Offer{Buy: []trades.Stack{{Item: trades.Emerald, Count: emeralds}}, Sell: trades.Of(item, count), MaxUses: uses, XP: xp}
}

func enchanted(item string, emeralds, uses, xp int) trades.Offer { // EnchantedItemForEmeralds
	o := itemsFor(item, emeralds, 1, uses, xp)
	o.Sell.Enchanted = true
	o.Note = "encantado al azar (nivel 5–19); cuesta más esmeraldas según el encantamiento"
	return o
}

func cook(raw string, cooked string, uses, xp int) trades.Offer { // ItemsAndEmeraldsToItems
	return trades.Offer{Buy: []trades.Stack{trades.Of(raw, 6), {Item: trades.Emerald, Count: 1}}, Sell: trades.Of(cooked, 6), MaxUses: uses, XP: xp}
}

func book(xp int) trades.Offer { // EnchantBookForEmeralds
	return trades.Offer{Buy: []trades.Stack{{Item: trades.Emerald, Count: 5, Max: 64}, trades.Of("minecraft:book", 1)},
		Sell: trades.Stack{Item: domain.MustParseResourceID("minecraft:enchanted_book"), Count: 1, Enchanted: true}, MaxUses: 12, XP: xp,
		Note: "libro con un encantamiento comerciable al azar (incluida Reparación; no Velocidad del alma ni Sigilo rápido); el precio sube con el nivel del encantamiento"}
}

// stew is SuspiciousStewForEmerald: the effect lasts the given seconds.
func stew(effect string, seconds int) trades.Offer {
	o := itemsFor("minecraft:suspicious_stew", 1, 1, 12, 15)
	o.Note = "estofado sospechoso de " + effect
	if seconds > 0 {
		o.Note += " (" + strconv.Itoa(seconds) + " s)"
	}
	return o
}

func dyed(item string, emeralds, uses, xp int) trades.Offer { // DyedArmorForEmeralds
	o := itemsFor(item, emeralds, 1, uses, xp)
	o.Note = "teñido de colores al azar"
	return o
}

func colors(f func(color string) trades.Offer) []trades.Offer {
	var out []trades.Offer
	for _, c := range []string{"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray", "light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black"} {
		out = append(out, f(c))
	}
	return out
}

func lv(n int, offers ...[]trades.Offer) trades.Level {
	l := trades.Level{Level: n, Name: levelNames[n], Picks: 2}
	for _, o := range offers {
		l.Offers = append(l.Offers, o...)
	}
	return l
}

func one(o ...trades.Offer) []trades.Offer { return o }

func profession(id, station string, levels ...trades.Level) *trades.Merchant {
	return &trades.Merchant{ID: domain.MustParseResourceID(id), Station: domain.MustParseResourceID(station),
		Entity: domain.MustParseResourceID("minecraft:villager"), Levels: levels}
}

func villagers(v121 bool) *trades.Catalog {
	scute := "minecraft:scute"
	if v121 {
		scute = "minecraft:turtle_scute"
	}
	cartographer3 := one(emeraldFor("minecraft:compass", 1, 12, 20), trades.Offer{Buy: []trades.Stack{{Item: trades.Emerald, Count: 14}, trades.Of("minecraft:compass", 1)}, Sell: trades.Of("minecraft:filled_map", 1), MaxUses: 12, XP: 10, Note: "mapa del explorador del bosque (mansión)"})
	if v121 {
		cartographer3 = append(cartographer3, trades.Offer{Buy: []trades.Stack{{Item: trades.Emerald, Count: 12}, trades.Of("minecraft:compass", 1)}, Sell: trades.Of("minecraft:filled_map", 1), MaxUses: 12, XP: 10, Note: "mapa del explorador de cámaras de prueba"})
	}
	c := &trades.Catalog{
		Key: "aldeanos", Name: "Aldeanos y comerciante errante", Order: 10, Confidence: domain.ConfidenceKnown,
		Intro: []string{
			"Los tradeos vanilla están en el código del juego, no en datos: la app los trae incluidos, verificados contra el código del juego (1.20.1 y 1.21).",
			"Cada vez que un aldeano sube de nivel aprende 2 tradeos elegidos al azar de la lista de ese nivel. La probabilidad indica cuántos aldeanos de esa profesión ofrecen cada uno.",
			"Los precios bajan con la reputación (curar aldeanos zombi, Héroe de la aldea) y suben con la demanda.",
		},
	}
	c.Merchants = append(c.Merchants,
		profession("minecraft:farmer", "minecraft:composter",
			lv(1, one(emeraldFor("minecraft:wheat", 20, 16, 2), emeraldFor("minecraft:potato", 26, 16, 2), emeraldFor("minecraft:carrot", 22, 16, 2), emeraldFor("minecraft:beetroot", 15, 16, 2), itemsFor("minecraft:bread", 1, 6, 16, 1))),
			lv(2, one(emeraldFor("minecraft:pumpkin", 6, 12, 10), itemsFor("minecraft:pumpkin_pie", 1, 4, 12, 5), itemsFor("minecraft:apple", 1, 4, 16, 5))),
			lv(3, one(itemsFor("minecraft:cookie", 3, 18, 12, 10), emeraldFor("minecraft:melon", 4, 12, 20))),
			lv(4, one(itemsFor("minecraft:cake", 1, 1, 12, 15), stew("visión nocturna", 5), stew("salto", 8), stew("debilidad", 7), stew("ceguera", 6), stew("veneno", 14), stew("saturación", 0))),
			lv(5, one(itemsFor("minecraft:golden_carrot", 3, 3, 12, 30), itemsFor("minecraft:glistering_melon_slice", 4, 3, 12, 30))),
		),
		profession("minecraft:fisherman", "minecraft:barrel",
			lv(1, one(emeraldFor("minecraft:string", 20, 16, 2), emeraldFor("minecraft:coal", 10, 16, 2), cook("minecraft:cod", "minecraft:cooked_cod", 16, 1), itemsFor("minecraft:cod_bucket", 3, 1, 16, 1))),
			lv(2, one(emeraldFor("minecraft:cod", 15, 16, 10), cook("minecraft:salmon", "minecraft:cooked_salmon", 16, 5), itemsFor("minecraft:campfire", 2, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:salmon", 13, 16, 20), enchanted("minecraft:fishing_rod", 3, 3, 10))),
			lv(4, one(emeraldFor("minecraft:tropical_fish", 6, 12, 30))),
			lv(5, one(emeraldFor("minecraft:pufferfish", 4, 12, 30), trades.Offer{Buy: []trades.Stack{trades.Of("minecraft:oak_boat", 1)}, Sell: trades.Stack{Item: trades.Emerald, Count: 1}, MaxUses: 12, XP: 30, Note: "el tipo de barca depende del bioma de la aldea"})),
		),
		profession("minecraft:shepherd", "minecraft:loom",
			lv(1, one(emeraldFor("minecraft:white_wool", 18, 16, 2), emeraldFor("minecraft:brown_wool", 18, 16, 2), emeraldFor("minecraft:black_wool", 18, 16, 2), emeraldFor("minecraft:gray_wool", 18, 16, 2), itemsFor("minecraft:shears", 2, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:white_dye", 12, 16, 10), emeraldFor("minecraft:gray_dye", 12, 16, 10), emeraldFor("minecraft:black_dye", 12, 16, 10), emeraldFor("minecraft:light_blue_dye", 12, 16, 10), emeraldFor("minecraft:lime_dye", 12, 16, 10)),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_wool", 1, 1, 16, 5) }),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_carpet", 1, 4, 16, 5) })),
			lv(3, one(emeraldFor("minecraft:yellow_dye", 12, 16, 20), emeraldFor("minecraft:light_gray_dye", 12, 16, 20), emeraldFor("minecraft:orange_dye", 12, 16, 20), emeraldFor("minecraft:red_dye", 12, 16, 20), emeraldFor("minecraft:pink_dye", 12, 16, 20)),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_bed", 3, 1, 12, 10) })),
			lv(4, one(emeraldFor("minecraft:brown_dye", 12, 16, 30), emeraldFor("minecraft:purple_dye", 12, 16, 30), emeraldFor("minecraft:blue_dye", 12, 16, 30), emeraldFor("minecraft:green_dye", 12, 16, 30), emeraldFor("minecraft:magenta_dye", 12, 16, 30), emeraldFor("minecraft:cyan_dye", 12, 16, 30)),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_banner", 3, 1, 12, 15) })),
			lv(5, one(itemsFor("minecraft:painting", 2, 3, 12, 30))),
		),
		profession("minecraft:fletcher", "minecraft:fletching_table",
			lv(1, one(emeraldFor("minecraft:stick", 32, 16, 2), itemsFor("minecraft:arrow", 1, 16, 12, 1), trades.Offer{Buy: []trades.Stack{trades.Of("minecraft:gravel", 10), {Item: trades.Emerald, Count: 1}}, Sell: trades.Of("minecraft:flint", 10), MaxUses: 12, XP: 1})),
			lv(2, one(emeraldFor("minecraft:flint", 26, 12, 10), itemsFor("minecraft:bow", 2, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:string", 14, 16, 20), itemsFor("minecraft:crossbow", 3, 1, 12, 10))),
			lv(4, one(emeraldFor("minecraft:feather", 24, 16, 30), enchanted("minecraft:bow", 2, 3, 15))),
			lv(5, one(emeraldFor("minecraft:tripwire_hook", 8, 12, 30), enchanted("minecraft:crossbow", 3, 3, 15), trades.Offer{Buy: []trades.Stack{trades.Of("minecraft:arrow", 5), {Item: trades.Emerald, Count: 2}}, Sell: trades.Of("minecraft:tipped_arrow", 5), MaxUses: 12, XP: 30, Note: "con un efecto al azar"})),
		),
		profession("minecraft:librarian", "minecraft:lectern",
			lv(1, one(emeraldFor("minecraft:paper", 24, 16, 2), book(1), itemsFor("minecraft:bookshelf", 9, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:book", 4, 12, 10), book(5), itemsFor("minecraft:lantern", 1, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:ink_sac", 5, 12, 20), book(10), itemsFor("minecraft:glass", 1, 4, 12, 10))),
			lv(4, one(emeraldFor("minecraft:writable_book", 2, 12, 30), book(15), itemsFor("minecraft:clock", 5, 1, 12, 15), itemsFor("minecraft:compass", 4, 1, 12, 15))),
			lv(5, one(itemsFor("minecraft:name_tag", 20, 1, 12, 30))),
		),
		profession("minecraft:cartographer", "minecraft:cartography_table",
			lv(1, one(emeraldFor("minecraft:paper", 24, 16, 2), itemsFor("minecraft:map", 7, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:glass_pane", 11, 16, 10), trades.Offer{Buy: []trades.Stack{{Item: trades.Emerald, Count: 13}, trades.Of("minecraft:compass", 1)}, Sell: trades.Of("minecraft:filled_map", 1), MaxUses: 12, XP: 5, Note: "mapa del explorador oceánico (monumento)"})),
			lv(3, cartographer3),
			lv(4, one(itemsFor("minecraft:item_frame", 7, 1, 12, 15)), colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_banner", 3, 1, 12, 15) })),
			lv(5, one(itemsFor("minecraft:globe_banner_pattern", 8, 1, 12, 30))),
		),
		profession("minecraft:cleric", "minecraft:brewing_stand",
			lv(1, one(emeraldFor("minecraft:rotten_flesh", 32, 16, 2), itemsFor("minecraft:redstone", 1, 2, 12, 1))),
			lv(2, one(emeraldFor("minecraft:gold_ingot", 3, 12, 10), itemsFor("minecraft:lapis_lazuli", 1, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:rabbit_foot", 2, 12, 20), itemsFor("minecraft:glowstone", 4, 1, 12, 10))),
			lv(4, one(emeraldFor(scute, 4, 12, 30), emeraldFor("minecraft:glass_bottle", 9, 12, 30), itemsFor("minecraft:ender_pearl", 5, 1, 12, 15))),
			lv(5, one(emeraldFor("minecraft:nether_wart", 22, 12, 30), itemsFor("minecraft:experience_bottle", 3, 1, 12, 30))),
		),
		profession("minecraft:armorer", "minecraft:blast_furnace",
			lv(1, one(emeraldFor("minecraft:coal", 15, 16, 2), itemsFor("minecraft:iron_leggings", 7, 1, 12, 1), itemsFor("minecraft:iron_boots", 4, 1, 12, 1), itemsFor("minecraft:iron_helmet", 5, 1, 12, 1), itemsFor("minecraft:iron_chestplate", 9, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:iron_ingot", 4, 12, 10), itemsFor("minecraft:bell", 36, 1, 12, 5), itemsFor("minecraft:chainmail_boots", 1, 1, 12, 5), itemsFor("minecraft:chainmail_leggings", 3, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:lava_bucket", 1, 12, 20), emeraldFor("minecraft:diamond", 1, 12, 20), itemsFor("minecraft:chainmail_helmet", 1, 1, 12, 10), itemsFor("minecraft:chainmail_chestplate", 4, 1, 12, 10), itemsFor("minecraft:shield", 5, 1, 12, 10))),
			lv(4, one(enchanted("minecraft:diamond_leggings", 14, 3, 15), enchanted("minecraft:diamond_boots", 8, 3, 15))),
			lv(5, one(enchanted("minecraft:diamond_helmet", 8, 3, 30), enchanted("minecraft:diamond_chestplate", 16, 3, 30))),
		),
		profession("minecraft:weaponsmith", "minecraft:grindstone",
			lv(1, one(emeraldFor("minecraft:coal", 15, 16, 2), itemsFor("minecraft:iron_axe", 3, 1, 12, 1), enchanted("minecraft:iron_sword", 2, 3, 1))),
			lv(2, one(emeraldFor("minecraft:iron_ingot", 4, 12, 10), itemsFor("minecraft:bell", 36, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:flint", 24, 12, 20))),
			lv(4, one(emeraldFor("minecraft:diamond", 1, 12, 30), enchanted("minecraft:diamond_axe", 12, 3, 15))),
			lv(5, one(enchanted("minecraft:diamond_sword", 8, 3, 30))),
		),
		profession("minecraft:toolsmith", "minecraft:smithing_table",
			lv(1, one(emeraldFor("minecraft:coal", 15, 16, 2), itemsFor("minecraft:stone_axe", 1, 1, 12, 1), itemsFor("minecraft:stone_shovel", 1, 1, 12, 1), itemsFor("minecraft:stone_pickaxe", 1, 1, 12, 1), itemsFor("minecraft:stone_hoe", 1, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:iron_ingot", 4, 12, 10), itemsFor("minecraft:bell", 36, 1, 12, 5))),
			lv(3, one(emeraldFor("minecraft:flint", 30, 12, 20), enchanted("minecraft:iron_axe", 1, 3, 10), enchanted("minecraft:iron_shovel", 2, 3, 10), enchanted("minecraft:iron_pickaxe", 3, 3, 10), itemsFor("minecraft:diamond_hoe", 4, 1, 3, 10))),
			lv(4, one(emeraldFor("minecraft:diamond", 1, 12, 30), enchanted("minecraft:diamond_axe", 12, 3, 15), enchanted("minecraft:diamond_shovel", 5, 3, 15))),
			lv(5, one(enchanted("minecraft:diamond_pickaxe", 13, 3, 30))),
		),
		profession("minecraft:butcher", "minecraft:smoker",
			lv(1, one(emeraldFor("minecraft:chicken", 14, 16, 2), emeraldFor("minecraft:porkchop", 7, 16, 2), emeraldFor("minecraft:rabbit", 4, 16, 2), itemsFor("minecraft:rabbit_stew", 1, 1, 12, 1))),
			lv(2, one(emeraldFor("minecraft:coal", 15, 16, 2), itemsFor("minecraft:cooked_porkchop", 1, 5, 16, 5), itemsFor("minecraft:cooked_chicken", 1, 8, 16, 5))),
			lv(3, one(emeraldFor("minecraft:mutton", 7, 16, 20), emeraldFor("minecraft:beef", 10, 16, 20))),
			lv(4, one(emeraldFor("minecraft:dried_kelp_block", 10, 12, 30))),
			lv(5, one(emeraldFor("minecraft:sweet_berries", 10, 12, 30))),
		),
		profession("minecraft:leatherworker", "minecraft:cauldron",
			lv(1, one(emeraldFor("minecraft:leather", 6, 16, 2), dyed("minecraft:leather_leggings", 3, 12, 1), dyed("minecraft:leather_chestplate", 7, 12, 1))),
			lv(2, one(emeraldFor("minecraft:flint", 26, 12, 10), dyed("minecraft:leather_helmet", 5, 12, 5), dyed("minecraft:leather_boots", 4, 12, 5))),
			lv(3, one(emeraldFor("minecraft:rabbit_hide", 9, 12, 20), dyed("minecraft:leather_chestplate", 7, 12, 10))),
			lv(4, one(emeraldFor(scute, 4, 12, 30), dyed("minecraft:leather_horse_armor", 6, 12, 15))),
			lv(5, one(itemsFor("minecraft:saddle", 6, 1, 12, 30), dyed("minecraft:leather_helmet", 5, 12, 30))),
		),
		profession("minecraft:mason", "minecraft:stonecutter",
			lv(1, one(emeraldFor("minecraft:clay_ball", 10, 16, 2), itemsFor("minecraft:brick", 1, 10, 16, 1))),
			lv(2, one(emeraldFor("minecraft:stone", 20, 16, 10), itemsFor("minecraft:chiseled_stone_bricks", 1, 4, 16, 5))),
			lv(3, one(emeraldFor("minecraft:granite", 16, 16, 20), emeraldFor("minecraft:andesite", 16, 16, 20), emeraldFor("minecraft:diorite", 16, 16, 20), itemsFor("minecraft:dripstone_block", 1, 4, 16, 10), itemsFor("minecraft:polished_andesite", 1, 4, 16, 10), itemsFor("minecraft:polished_diorite", 1, 4, 16, 10), itemsFor("minecraft:polished_granite", 1, 4, 16, 10))),
			lv(4, one(emeraldFor("minecraft:quartz", 12, 12, 30)),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_terracotta", 1, 1, 12, 15) }),
				colors(func(c string) trades.Offer { return itemsFor("minecraft:"+c+"_glazed_terracotta", 1, 1, 12, 15) })),
			lv(5, one(itemsFor("minecraft:quartz_pillar", 1, 1, 12, 30), itemsFor("minecraft:quartz_block", 1, 1, 12, 30))),
		),
	)
	c.Merchants = append(c.Merchants, wanderer1_20())
	return c
}

// wanderer1_20 lists what the wandering trader sells (Minecraft 1.20.1,
// checked against the game's code): every time it spawns it picks 5 offers
// from the common list and 1 from the rare list.
func wanderer1_20() *trades.Merchant {
	type w struct {
		item            string
		emeralds, count int
		uses            int
	}
	common := []w{
		{"sea_pickle", 2, 1, 5},
		{"slime_ball", 4, 1, 5},
		{"glowstone", 2, 1, 5},
		{"nautilus_shell", 5, 1, 5},
		{"fern", 1, 1, 12},
		{"sugar_cane", 1, 1, 8},
		{"pumpkin", 1, 1, 4},
		{"kelp", 3, 1, 12},
		{"cactus", 3, 1, 8},
		{"dandelion", 1, 1, 12},
		{"poppy", 1, 1, 12},
		{"blue_orchid", 1, 1, 8},
		{"allium", 1, 1, 12},
		{"azure_bluet", 1, 1, 12},
		{"red_tulip", 1, 1, 12},
		{"orange_tulip", 1, 1, 12},
		{"white_tulip", 1, 1, 12},
		{"pink_tulip", 1, 1, 12},
		{"oxeye_daisy", 1, 1, 12},
		{"cornflower", 1, 1, 12},
		{"lily_of_the_valley", 1, 1, 7},
		{"wheat_seeds", 1, 1, 12},
		{"beetroot_seeds", 1, 1, 12},
		{"pumpkin_seeds", 1, 1, 12},
		{"melon_seeds", 1, 1, 12},
		{"acacia_sapling", 5, 1, 8},
		{"birch_sapling", 5, 1, 8},
		{"dark_oak_sapling", 5, 1, 8},
		{"jungle_sapling", 5, 1, 8},
		{"oak_sapling", 5, 1, 8},
		{"spruce_sapling", 5, 1, 8},
		{"cherry_sapling", 5, 1, 8},
		{"mangrove_propagule", 5, 1, 8},
		{"red_dye", 1, 3, 12},
		{"white_dye", 1, 3, 12},
		{"blue_dye", 1, 3, 12},
		{"pink_dye", 1, 3, 12},
		{"black_dye", 1, 3, 12},
		{"green_dye", 1, 3, 12},
		{"light_gray_dye", 1, 3, 12},
		{"magenta_dye", 1, 3, 12},
		{"yellow_dye", 1, 3, 12},
		{"gray_dye", 1, 3, 12},
		{"purple_dye", 1, 3, 12},
		{"light_blue_dye", 1, 3, 12},
		{"lime_dye", 1, 3, 12},
		{"orange_dye", 1, 3, 12},
		{"brown_dye", 1, 3, 12},
		{"cyan_dye", 1, 3, 12},
		{"brain_coral_block", 3, 1, 8},
		{"bubble_coral_block", 3, 1, 8},
		{"fire_coral_block", 3, 1, 8},
		{"horn_coral_block", 3, 1, 8},
		{"tube_coral_block", 3, 1, 8},
		{"vine", 1, 1, 12},
		{"brown_mushroom", 1, 1, 12},
		{"red_mushroom", 1, 1, 12},
		{"lily_pad", 1, 2, 5},
		{"small_dripleaf", 1, 2, 5},
		{"sand", 1, 8, 8},
		{"red_sand", 1, 4, 6},
		{"pointed_dripstone", 1, 2, 5},
		{"rooted_dirt", 1, 2, 5},
		{"moss_block", 1, 2, 5},
	}
	rare := []w{
		{"tropical_fish_bucket", 5, 1, 4},
		{"pufferfish_bucket", 5, 1, 4},
		{"packed_ice", 3, 1, 6},
		{"blue_ice", 6, 1, 6},
		{"gunpowder", 1, 1, 8},
		{"podzol", 3, 3, 6},
	}
	m := &trades.Merchant{ID: domain.MustParseResourceID("minecraft:wandering_trader"), Entity: domain.MustParseResourceID("minecraft:wandering_trader")}
	mk := func(name string, picks int, list []w) trades.Level {
		l := trades.Level{Level: len(m.Levels) + 1, Name: name, Picks: picks}
		for _, x := range list {
			l.Offers = append(l.Offers, itemsFor("minecraft:"+x.item, x.emeralds, x.count, x.uses, 1))
		}
		return l
	}
	m.Levels = append(m.Levels, mk("Ofertas comunes", 5, common))
	m.Levels = append(m.Levels, mk("Oferta rara", 1, rare))
	return m
}
