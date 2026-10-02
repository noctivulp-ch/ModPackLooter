package names

import (
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

func TestAssetUsesAnyLangKey(t *testing.T) {
	es := map[string]string{
		"deceasedcraft.advancement.title.retail_district":       "Distrito: Comercial",
		"deceasedcraft.advancement.description.retail_district": "Visita el distrito comercial",
		"deceasedcraft.advancement.description.suburb":          "Solo descripción",
		"deceasedcraft.advancement.title.park":                  "Parque",
		"deceasedcraft.advancement.title.empty":                 "",
	}
	en := map[string]string{
		"citystyle.deceasedcraft.highrise": "Highrise",
		"citystyle.deceasedcraft.park":     "Park",
	}
	n := New(es, en)
	cases := map[string]string{
		"deceasedcraft:retail_district": "Distrito: Comercial",
		"deceasedcraft:highrise":        "Highrise",
		"deceasedcraft:suburb":          "Suburb", // only a description: readable id
		"deceasedcraft:park":            "Parque", // es found by search beats en explicit key
		"other:retail_district":         "Retail District",
		"deceasedcraft:empty":           "Empty", // empty value counts as missing
	}
	for id, want := range cases {
		if got := n.Asset("citystyle", domain.MustParseResourceID(id)); got != want {
			t.Errorf("Asset(%s) = %q, quiero %q", id, got, want)
		}
	}
}
