package generic

import (
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

func TestKindFromPath(t *testing.T) {
	cases := map[string]domain.SourceKind{
		"chests/desert_pyramid":              domain.KindContainer,
		"chests/village/village_weaponsmith": domain.KindContainer,
		"entities/zombie":                    domain.KindEntity,
		"entity/item_frame_empty":            domain.KindEntity,
		"gameplay/fishing/treasure":          domain.KindFishing,
		"gameplay/hero_of_the_village/baby":  domain.KindGameplay,
		"archaeology/desert_pyramid":         domain.KindArchaeology,
		"blocks/oak_log":                     domain.KindBlock,
		"structures/tower/top_chest":         domain.KindUnknown,
		"custom_fishing_rewards":             domain.KindFishing,
	}
	for path, want := range cases {
		if got := KindFromPath(path); got != want {
			t.Errorf("KindFromPath(%q) = %s, se esperaba %s", path, got, want)
		}
	}
}
