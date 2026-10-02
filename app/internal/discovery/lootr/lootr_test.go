package lootr

import "testing"

func TestParseConfigFlatAndSectioned(t *testing.T) {
	flat := "refresh_all = true\nrefresh_value = 6000\nloot_modid_blacklist = [\"a\"]\nconvert_mineshafts = false\n"
	sectioned := "[refresh]\nrefresh_all = true\nrefresh_value = 6000\n[whitelist]\nloot_table_modid_blacklist = [\"a\"]\n[conversion]\nconvert_mineshafts = false\n"
	for name, data := range map[string]string{"1.20.1 (raíz)": flat, "1.21 (secciones)": sectioned} {
		cfg, err := parseConfig(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !cfg.RefreshAll || cfg.RefreshValue != 6000 || len(cfg.LootModidBlacklist) != 1 || cfg.ConvertMineshafts {
			t.Errorf("%s: %+v", name, cfg)
		}
		if cfg.DecayValue != 6000 && cfg.DecayValue != 5*60*20 {
			t.Errorf("%s: decay por defecto = %d", name, cfg.DecayValue)
		}
	}
	if _, err := parseConfig("not = [valid"); err == nil {
		t.Error("TOML inválido debe dar error")
	}
}

func TestMinutes(t *testing.T) {
	for ticks, want := range map[int]string{1200: "1 minuto", 24000: "20 minutos", 1800: "1.5 minutos"} {
		if got := minutes(ticks); got != want {
			t.Errorf("minutes(%d) = %q, se esperaba %q", ticks, got, want)
		}
	}
}
