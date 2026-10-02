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

## 3. La loot table como pivote

Casi todas las fuentes de loot del juego (cofres, barriles, bóvedas, arena
sospechosa, vagonetas con cofre, mobs, pesca, regalos de gameplay) terminan
apuntando a una **loot table**. Por eso el modelo gira en torno a ella:

1. Se reúnen **todas** las loot tables (vanilla + mods + datapacks).
2. Se buscan **todos los sitios que las referencian**: plantillas NBT,
   processor lists, entidades, condiciones, tabla de conocimiento, overrides.
3. Cada referencia se convierte en una `LootSource`, que a su vez se asocia a
   estructuras y biomas.
4. Las loot tables **sin ninguna referencia** se muestran igualmente,
   clasificadas por su tipo/ruta (`chests/`, `entities/`, `gameplay/`…) y
   marcadas como “origen no determinado”.

## 4. Cómo se llega de un objeto a un bioma

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
2. **Pieza → tablas de loot.** Se lee cada `.nbt` y se buscan:
   - bloques con entidad (`chest`, `barrel`, `trapped_chest`, `vault`,
     `suspicious_sand`, `dispenser`…) con `LootTable`;
   - entidades guardadas en la plantilla (p. ej. `chest_minecart`) con
     `LootTable`, y mobs con `DeathLootTable`;
   - **processor lists** del pool (`worldgen/processor_list`) con reglas
     `block_entity_modifier` de tipo `minecraft:append_loot`, que asignan loot
     al generar;
   - **data markers** (bloques de estructura con `Metadata`), que se
     resuelven con la tabla de conocimiento o con overrides.
3. **Estructura → biomas.** El campo `biomes` (ID, lista o `#tag`) se expande
   resolviendo tags de forma recursiva, incluidos los añadidos por otros mods.
4. **Tabla → objetos.** Se recorren pools y entradas; las entradas de tipo
   `loot_table` (tablas anidadas), `tag` y `alternatives`/`group` se expanden.
5. **Modificadores.** Se aplican Global Loot Modifiers, datapacks y scripts
   reconocidos sobre las tablas resultantes.

6. **Entidades → biomas.** Los `spawners` del JSON de cada bioma y los
   *biome modifiers* `add_spawns` de Forge/NeoForge indican dónde aparece
   cada mob; con eso su loot table se asocia a biomas.
7. **Condiciones de la propia tabla.** Una condición `location_check` con
   bioma o estructura (p. ej. la pesca en la jungla) también crea una
   asociación con ese bioma o estructura.

### Casos que no se pueden resolver estáticamente

- Loot asignado por **código Java** (algunos mods rellenan cofres sin
  `LootTable` en el NBT, o usan eventos de Fabric/Forge).
- Estructuras generadas proceduralmente sin plantillas NBT.

Para estos casos la app usará, en este orden:

1. **Tabla de conocimiento vanilla** por versión: estructuras clásicas
   generadas por código (mineshaft, fortaleza, stronghold, templos,
   naufragios…).
2. **Heurísticas** marcadas como tales (p. ej. una tabla
   `examplemod:chests/ruined_tower` se asocia a la estructura
   `examplemod:ruined_tower` por nombre).
3. **Overrides manuales** en el archivo de configuración del proyecto.
4. Si nada aplica: la fuente aparece como **“origen no determinado”**.

Cada relación guarda su **nivel de confianza**: `exacta`, `conocida`,
`heurística` o `manual`, y el sitio lo muestra.

## 5. Tipos de fuente de loot

| Tipo | Ejemplo | ¿Asociable a bioma? |
|---|---|---|
| Cofre de estructura | Templo del desierto | Sí, vía estructura. |
| Bóveda / dispensador / barril | Trial Chambers | Sí, vía estructura. |
| Arqueología | Arena sospechosa en ruinas | Sí, vía estructura. |
| Vagoneta con cofre | Mina abandonada | Sí, vía estructura. |
| Entidad (drop de mob) | Wither Skeleton | Sí, vía spawns del bioma y biome modifiers. |
| Pesca | Tesoro de pesca | Global, salvo condiciones `location_check`. |
| Gameplay | Regalos de aldeanos, gato | No. |
| Bloque | Drops al romper | No (fuera del MVP). |

## 6. Probabilidades

Para cada objeto en una tabla se calculan, cuando sea posible:

- **Probabilidad por tirada**: `peso_entrada / suma_pesos_del_pool`.
- **Probabilidad por cofre**: probabilidad de que aparezca al menos una vez,
  teniendo en cuenta el número de tiradas (`rolls`, que puede ser un rango) y
  las condiciones simples (`random_chance`).
- **Cantidad**: rango `min–max` según la función `set_count`.

Condiciones o funciones no soportadas no se ignoran en silencio: la cifra se
marca como **aproximada**.

## 7. Modelo de dominio (borrador)

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
