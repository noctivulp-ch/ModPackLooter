# 16 · Pestaña de pesca

Capas: **Pesca › Fuente › Bioma › Condiciones › Captura**. Cada fuente es un
sistema de pesca distinto (cada uno con su caña) y calcula las probabilidades
con sus propias reglas. La pestaña aparece si hay al menos una fuente; la
vanilla se incluye siempre que exista su tabla, porque también es útil.

Arquitectura (igual que Lost Cities, [doc 15](15-pestanas-por-mod-y-lost-cities.md)):
cada descubridor arma un `fishing.Source` y lo adjunta con la clave
`fishing/<mod>`. El paquete `internal/fishing` es el modelo común: entradas,
reglas de bioma, grupos y probabilidades por bioma. El sitio solo pinta.

## Biomas

`fishing.Biomes` reúne los biomas del modpack: su dimensión (del mundo modelo,
o de los tags `is_overworld` / `is_nether` / `is_end`), sus tags y su
temperatura (`worldgen/biome/*.json`). Con mundo modelo se usan solo sus biomas.
Los datos que no encajan con ningún bioma conocido se listan aparte.

## Starcatcher (2.3, 1.20.1)

Código: `FishingBobEntity.reel`, `FishProperties.calculateChance`, restricciones
en `registry/fishrestrictions`.

- Archivos: `data/<ns>/starcatcher/fish/*.json`. Starcatcher trae datos para
  muchos mods con `forge:conditions` (`mod_loaded`): se evalúan y se descartan
  los de mods ausentes.
- Probabilidad: `base_chance` + lo que sumen las restricciones; las que fallan
  restan 9999. Se elige al azar en proporción al resultado entre los peces
  posibles.
- Cebo: suma su bonificación. Un pez con base 0 **solo** sale con su cebo; se
  muestra en el grupo «Con cebo» con la probabilidad usando ese cebo.
- Se modelan bioma (lista, tags y listas negras), dimensión, líquido y sesgo
  por bioma. Altura, hora, clima, estaciones, límite de capturas, rareza
  previa y `percentage_chance` se muestran como condiciones. Los textos usan
  `translation_override` del lang cuando existe.
- Las estaciones solo cuentan con Serene Seasons, Ecliptic Seasons o TFC; si
  no, se marcan «(sin efecto)».

Ejemplo: la **Estrella del Nether** (`data/minecraft/starcatcher/fish/nether_star.json`)
tiene base 0 y solo sale con cebo de **calavera de esqueleto Wither** (+200);
al pescarla aparece un **Wither**.

## Tide 2 (2.1.1)

Código: `TideFishingManager`, `FishingRandomSelector`, `FishSelector`,
`CrateSelector`, `data/fishing/*`. Repositorio: github.com/Lightning-64/Tide-2.

- Archivos: `data/<ns>/fishing/fish/**`, `fishing/loot/*`, `fishing/crates/*`.
  Se respetan `associated_mods` y `forge:conditions`.
- Cada captura elige por peso entre: los botines cuyas condiciones se cumplen,
  los **peces** (peso 85) y las **cajas** (`crateWeight` de la config, 4 por
  defecto, si alguna cabe). La suerte suma `quality × suerte`.
- Pez: `selection_weight` × modificador de temperatura
  `max(0, 1 − ((t − preferida) / tolerancia)²)` con la temperatura del bioma.
- `found_in`, `freshwater`/`saltwater` (tag `tide:is_saltwater`), `dimension`
  y `fluid` deciden dónde aplica; el resto (altura, hora, luna, clima,
  estaciones, estructuras cercanas, suerte…) se muestra como condición.
- Config: `config/tide/tide_server.json` (`crateWeight`, `overrideVanillaRod`).
  Si Tide reemplaza la caña vanilla, la fuente vanilla lo avisa.
- Las tablas de botín y de cajas se publican como fuentes de pesca exactas.

## Vanilla

`minecraft:gameplay/fishing`: peces / basura / tesoro por peso; el tesoro
exige aguas abiertas. Se muestran dos grupos: en aguas abiertas y fuera.

## Limitaciones

- La altura no divide los grupos: un pez «por debajo de Y=40» aparece junto a
  los de superficie, con su condición a la vista.
- Los botines de Tide se muestran con su peso frente al de los peces, no como
  porcentaje, porque cuáles aplican depende sobre todo de la altura.
- Los bloqueos hechos por código de otros mods (eventos, LootJS, Global Loot
  Modifiers) aún no se detectan (pendiente). Los hechos con datos sí: ver abajo.

## Encantamientos en el loot (caso Reparación en DeceasedCraft)

En DeceasedCraft «no se puede pescar Reparación». La causa está en los datos de
DCTweaks 5.11.18: reemplaza `minecraft:gameplay/fishing/treasure` y cambia
`enchant_with_levels` de nivel 30 con `treasure: true` a **nivel 15 con
`treasure: false`**, así que no salen encantamientos de tesoro (Reparación,
Paso helado, Velocidad del alma, Sigilo rápido, maldiciones).

Como la app ya aplica la prioridad de packs (vanilla < mods < datapacks), usa
esa tabla. Ahora las notas de los objetos lo explican:

- `enchant_with_levels` → «Encantado (nivel 15, sin encantamientos de tesoro:
  no da Reparación, Paso helado ni maldiciones)» o «…puede dar encantamientos
  de tesoro como Reparación».
- `enchant_randomly` con lista → «Encantado al azar con: …».

Los nombres de encantamientos salen del lang (`enchantment.<ns>.<id>`).

Aparte, Quark tiene `Nerf Mending = true` en ese pack: Reparación ya no repara
con experiencia (en el yunque repara entero y se consume), pero no la quita del
loot.
