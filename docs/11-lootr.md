# 11 · Soporte de Lootr

> Investigado el 2026-10-02 leyendo el código fuente de Lootr (ramas `1.20.1`
> y `mdg-26.1.2` de [LootrMinecraft/Lootr](https://github.com/LootrMinecraft/Lootr)).

## Qué hace Lootr

Lootr reemplaza los contenedores generados por el mundo que tienen una
`LootTable` por versiones **instanciadas por jugador**: cada jugador obtiene su
propio loot del mismo cofre.

- **No modifica las loot tables.** El loot se genera con la misma tabla que
  tendría el contenedor vanilla, así que las probabilidades no cambian.
- **Convierte en tiempo de ejecución.** En 1.20.1 la conversión ocurre al
  cargar chunks (evento `HandleChunk` + mixin en `LevelChunk`) y al aparecer
  vagonetas (`HandleCart`). No queda nada escrito en los archivos de datos, por
  lo que **no hay nada que “descubrir”**; hay que **anotar** las fuentes ya
  descubiertas.
- **Los cofres ya abiertos no se convierten.** Si un mod rellena el cofre por
  código, sin `LootTable`, Lootr tampoco puede convertirlo.

## Qué hay que mostrar en el sitio

| Dato | Para qué le sirve al jugador |
|---|---|
| “Loot por jugador (Lootr)” | Saber que aunque otro haya saqueado el cofre, él también puede. |
| **Refresh**: el cofre se rellena cada *N* minutos | Saber que se puede volver a farmear. |
| **Decay**: el cofre desaparece *N* minutos después de abrirse | Saber que es de un solo uso. |
| Excluido de Lootr (blacklist) | Saber que ese cofre es compartido. |
| Élitros de la ciudad del End convertidos a cofre | Con la opción activada, aparece la tabla `lootr:chests/elytra`. |

## Identificadores por versión

Es un buen ejemplo de por qué el descubrimiento necesita **variantes por
versión**:

| Concepto | 1.20.1 (Forge) | 26.x (NeoForge/Fabric, multicargador) |
|---|---|---|
| Bloques | `lootr:lootr_chest`, `lootr_trapped_chest`, `lootr_barrel`, `lootr_shulker`, `lootr_inventory` | `lootr:chest`, `trapped_chest`, `barrel`, `shulker_box`, `inventory`, `copper_chest` (+ variantes oxidadas), `suspicious_sand`, `suspicious_gravel`, `decorated_pot` |
| Entidades | `lootr:lootr_minecart` | `lootr:chest_minecart`, `lootr:item_frame` |
| Qué convierte | Cofres, cofres trampa, barriles, shulkers y vagonetas de mina | Lo anterior + cofres de cobre, arena/grava sospechosa, vasijas decoradas y marcos de item (élitros) |
| Configuración | `config/lootr-common.toml` (ForgeConfigSpec) | Config propia del mod (formato *por verificar*) **+ tags de datos** |
| Tags de datos | `lootr:tags/blocks/*` | `lootr:tags/block/convert/*`, `lootr:tags/worldgen/structure/{blacklist,whitelist,decay,refresh}` |
| Loot tables propias | Solo drops de sus bloques | `lootr:chests/elytra`, drops de bloques, `entity/item_frame_empty` |

> `lootr_inventory` / `inventory` es un contenedor con **items fijos** (no
> usa loot table). Si aparece en una plantilla NBT se registra como fuente de
> items fijos.

## Opciones de configuración relevantes (1.20.1)

| Clave | Efecto en el sitio |
|---|---|
| `dimension_whitelist` / `dimension_blacklist` | Qué dimensiones se convierten. |
| `loot_table_blacklist`, `loot_modid_blacklist`, `loot_structure_blacklist` | Fuentes que **no** son por jugador. |
| `decay_value`, `decay_all`, `decay_loot_tables`, `decay_modids`, `decay_dimensions`, `decay_structures` | Marcar los cofres con *decay* y su tiempo. |
| `refresh_value`, `refresh_all`, `refresh_loot_tables`, `refresh_modids`, `refresh_dimensions`, `refresh_structures` | Marcar los cofres con *refresh* y su tiempo. |
| `convert_mineshafts` | Las vagonetas de mina pasan a ser por jugador. |
| `convert_wooden_chests`, `convert_trapped_chests`, `additional_chests` | Qué bloques de mods se consideran contenedores convertibles. |

En 26.x aparecen además `loot_table_force_whitelist`,
`modid_dimension_whitelist/blacklist`, `convert_elytras_to_chests`,
`convert_elytras_to_item_frames` y `convert_structure_item_frames`, entre
otras.

## Cómo encaja en la arquitectura

Lootr se soporta con **tres piezas enganchables** independientes (ver
[12-arquitectura-de-descubrimiento.md](12-arquitectura-de-descubrimiento.md)):

1. **Catálogo de contenedores** (`ContainerCatalog`): registra los IDs de
   bloques y entidades de Lootr de cada versión como contenedores con loot,
   para que el lector NBT los reconozca si un mod los coloca directamente en
   sus plantillas.
2. **Descubridor `lootr-extras`**: aporta la fuente
   `lootr:chests/elytra` → ciudad del End cuando la opción está activa.
3. **Enriquecedor `lootr`** (fase posterior al descubrimiento): lee la
   configuración y los tags de Lootr y **anota** cada fuente de tipo
   contenedor con `por jugador`, `refresh` (tiempo) o `decay` (tiempo).

Las tres piezas solo se activan si el modpack incluye Lootr, y cada una tiene
variantes para 1.20.1 y 26.x.

## Entrada que necesita la app

- Detectar Lootr por su `modId` en los metadatos de los jars.
- Leer `config/lootr-common.toml` (y `defaultconfigs/` si existe) en 1.20.1.
- Leer los tags `lootr:*` de los datos fusionados en 26.x.
- Si no hay configuración, se usan los valores por defecto del mod de esa
  versión.
