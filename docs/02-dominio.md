# 02 · Dominio: de dónde sale el loot

Este documento describe los conceptos de Minecraft con los que trabaja la app
y cómo se conectan entre sí. Es la base del modelo de dominio.

## 1. Fuentes de datos dentro de un modpack

| Origen | Ubicación típica | Qué aporta |
|---|---|---|
| Minecraft vanilla | `client.jar` / `server.jar` de la versión | Tablas de loot, estructuras, biomas, nombres y texturas base. |
| Mods | `mods/*.jar` → carpetas `data/` y `assets/` | Tablas de loot, estructuras, biomas, tags, idiomas, texturas. |
| Datapacks | `datapacks/`, `world/datapacks/`, `global_packs/` (carpeta o `.zip`) | Añaden o **sobrescriben** tablas, estructuras y tags. |
| Scripts | `kubejs/`, `scripts/` (CraftTweaker) | Modificaciones de loot por script. |
| Configuración | `config/`, `defaultconfigs/` | Algunos mods controlan su loot por config. |
| Exportaciones | `.zip` de CurseForge / `.mrpack` de Modrinth | Manifiesto con la lista de mods (y `overrides/`). |

> **Orden de prioridad:** como en el juego, un datapack puede sobrescribir el
> archivo de un mod, y un mod el de vanilla. La app debe aplicar la misma
> regla de “el último gana” y registrar qué archivo terminó siendo el efectivo.

## 2. Archivos relevantes dentro de `data/<namespace>/`

> Las rutas cambian según la versión: desde la 1.21 varias carpetas pasaron a
> singular (`loot_tables` → `loot_table`, `structures` → `structure`). Por eso
> la lectura de formatos se aísla en adaptadores por versión.

| Ruta | Concepto |
|---|---|
| `loot_table(s)/**.json` | **Tablas de loot**: cofres, entidades, bloques, pesca, arqueología, gameplay… |
| `worldgen/structure/*.json` | **Estructuras**: tipo, biomas permitidos (normalmente un *tag*), pools de jigsaw. |
| `worldgen/structure_set/*.json` | Reglas de colocación (espaciado, frecuencia). |
| `worldgen/template_pool/*.json` | Piezas que componen una estructura jigsaw. |
| `structure(s)/**.nbt` | **Plantillas NBT**: bloques de la pieza; los cofres llevan la etiqueta `LootTable`. |
| `worldgen/biome/*.json` | **Biomas** definidos por mods. |
| `tags/worldgen/biome/*.json` | **Tags de bioma** (p. ej. `#minecraft:is_ocean`). |
| `tags/items/*.json` | Tags de objetos (una entrada de loot puede apuntar a un tag). |
| `loot_modifiers/` (Forge/NeoForge) | **Global Loot Modifiers**: añaden objetos a tablas existentes. |

Y en `assets/<namespace>/`:

| Ruta | Concepto |
|---|---|
| `lang/*.json` | Nombres traducidos de objetos, estructuras y biomas. |
| `textures/item/`, `models/item/` | Iconos de los objetos. |

## 3. Cómo se llega de un objeto a un bioma

```
Objeto ──aparece en──▶ Entrada de loot ──pertenece a──▶ Tabla de loot
                                                          │
                     (referenciada por un cofre/barril/bóveda dentro de)
                                                          ▼
Bioma ◀──puede generarse en── Estructura ◀──compuesta por── Pieza (plantilla NBT)
```

Pasos de resolución:

1. **Estructura → piezas.** Se recorren los `template_pool` desde el
   `start_pool` de la estructura (de forma recursiva por los conectores
   jigsaw) o, para estructuras clásicas, se usan las piezas conocidas.
2. **Pieza → tablas de loot.** Se lee cada `.nbt` y se buscan bloques con
   entidad (`chest`, `barrel`, `trapped_chest`, `vault`, `suspicious_sand`,
   `dispenser`…) que tengan `LootTable`. También spawners y entidades con
   `DeathLootTable`.
3. **Estructura → biomas.** El campo `biomes` (ID, lista o `#tag`) se expande
   resolviendo tags de forma recursiva, incluidos los añadidos por otros mods.
4. **Tabla → objetos.** Se recorren pools y entradas; las entradas de tipo
   `loot_table` (tablas anidadas), `tag` y `alternatives`/`group` se expanden.
5. **Modificadores.** Se aplican Global Loot Modifiers, datapacks y scripts
   reconocidos sobre las tablas resultantes.

### Casos que no se pueden resolver estáticamente

- Loot asignado por **código Java** (algunos mods rellenan cofres sin
  `LootTable` en el NBT, o usan eventos de Fabric/Forge).
- Estructuras generadas proceduralmente sin plantillas NBT.

Para estos casos la app usará, en este orden:

1. **Heurísticas** marcadas como tales (p. ej. una tabla
   `examplemod:chests/ruined_tower` se asocia a la estructura
   `examplemod:ruined_tower` por nombre).
2. **Overrides manuales** en el archivo de configuración del proyecto.
3. Si nada aplica: la fuente aparece como **“origen no determinado”**.

Cada relación guarda su **nivel de confianza**: `exacta`, `heurística` o
`manual`, y el sitio lo muestra.

## 4. Tipos de fuente de loot

| Tipo | Ejemplo | ¿Asociable a bioma? |
|---|---|---|
| Cofre de estructura | Templo del desierto | Sí, vía estructura. |
| Bóveda / dispensador / barril | Trial Chambers | Sí, vía estructura. |
| Arqueología | Arena sospechosa en ruinas | Sí, vía estructura. |
| Entidad (drop de mob) | Wither Skeleton | Parcialmente (spawns por bioma, fase futura). |
| Pesca | Tesoro de pesca | No (global, salvo condiciones). |
| Gameplay | Regalos de aldeanos, gato | No. |
| Bloque | Drops al romper | No (fuera del MVP). |

## 5. Probabilidades

Para cada objeto en una tabla se calculan, cuando sea posible:

- **Probabilidad por tirada**: `peso_entrada / suma_pesos_del_pool`.
- **Probabilidad por cofre**: probabilidad de que aparezca al menos una vez,
  teniendo en cuenta el número de tiradas (`rolls`, que puede ser un rango) y
  las condiciones simples (`random_chance`).
- **Cantidad**: rango `min–max` según la función `set_count`.

Condiciones o funciones no soportadas no se ignoran en silencio: la cifra se
marca como **aproximada**.

## 6. Modelo de dominio (borrador)

| Entidad | Atributos principales |
|---|---|
| `Modpack` | nombre, versión de Minecraft, loader, lista de `Mod`. |
| `Mod` | id/namespace, nombre, versión. |
| `ResourceId` | `namespace:path` (objeto de valor, base de todos los IDs). |
| `Item` | id, nombre traducido, icono, mod de origen. |
| `LootTable` | id, tipo, pools, archivo de origen, modificadores aplicados. |
| `LootPool` / `LootEntry` | tiradas, pesos, condiciones, funciones. |
| `ItemDrop` | objeto, tabla, prob. por tirada, prob. por cofre, cantidad, flag *aproximada*. |
| `Structure` | id, nombre, mod, dimensión, biomas, tablas de loot. |
| `Biome` | id, nombre, dimensión, tags. |
| `LootSource` | tipo (estructura, entidad, pesca…), tablas, confianza. |
| `LootIndex` | El grafo resuelto: consultas objeto→fuentes, bioma→estructuras, etc. |
| `Diagnostic` | Advertencias del análisis (archivo ilegible, referencia rota, heurística aplicada). |
