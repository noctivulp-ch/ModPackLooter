package names

import (
	"reflect"
	"testing"

	"github.com/EnierAragon/ModPackLooter/app/internal/domain"
)

func TestLangChain(t *testing.T) {
	cases := []struct {
		req       string
		available []string
		want      []string
	}{
		{"es-AR", []string{"es_mx", "es_es", "en_us", "fr_fr"}, []string{"es_ar", "es_es", "es_mx", "en_us"}},
		{"", nil, []string{"es_es", "en_us"}},
		{"en_gb", []string{"en_us"}, []string{"en_gb", "en_us"}},
		{"pt_pt", []string{"pt_br"}, []string{"pt_pt", "pt_br", "en_us"}},
		{"tok", nil, []string{"tok", "en_us"}},
	}
	for _, c := range cases {
		if got := LangChain(c.req, c.available); !reflect.DeepEqual(got, c.want) {
			t.Errorf("LangChain(%q) = %v, se esperaba %v", c.req, got, c.want)
		}
	}
}

func TestNamerUsesChainOrder(t *testing.T) {
	n := New(map[string]string{"item.minecraft.potato": "Papa"}, map[string]string{"item.minecraft.potato": "Patata", "item.minecraft.carrot": "Zanahoria"})
	if n.Item(mustID("minecraft:potato")) != "Papa" || n.Item(mustID("minecraft:carrot")) != "Zanahoria" || n.Item(mustID("x:golden_thing")) != "Golden Thing" {
		t.Error("el primer idioma debe ganar y el siguiente rellenar huecos")
	}
}

func mustID(s string) domain.ResourceID { return domain.MustParseResourceID(s) }
