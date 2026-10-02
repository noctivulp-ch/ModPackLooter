# 14 · Mundo modelo y desactivadores

> Decisiones: [ADR-0013](decisiones/0013-mundo-modelo.md) y
> [ADR-0014](decisiones/0014-desactivadores-con-dos-niveles.md).

## 1. Mundo modelo (`--world`)

Muchos modpacks fijan cómo se genera el mundo: DeceasedCraft, por ejemplo, asigna por
defecto un perfil de Lost Cities (`deceasedcraft_onlycities`) y elimina biomas. En vez de
replicar el menú de creación de mundos con todas las opciones que añaden los mods, la app
lee **un mundo ya creado con el modpack**:

```bash
modpacklooter build <instancia> --world "Mi mundo"          # nombre dentro de saves/
modpacklooter build <instancia> --world /ruta/al/mundo     # carpeta con level.dat
```

Qué se toma del mundo (verificado con el mundo real *Resty* de DeceasedCraft 1.20.1):

| Dato | De dónde | Para qué |
|---|---|---|
| Dimensiones y su generador | `level.dat` → `Data.WorldGenSettings.dimensions` | Saber qué dimensiones existen (`lostcities:lostcity`, `deceasedcraft:abyss`…). |
| **Lista exacta de biomas** por dimensión | `biome_source.biomes` (multi_noise con lista, fixed, checkerboard; The End es fijo) | Estructuras cuyos biomas no existen → desactivadas con seguridad. En *Resty* el Overworld tiene 48 biomas: no hay `desert`, `deep_dark` ni `mushroom_fields`. |
| Datapacks activos y desactivados | `Data.DataPacks.Enabled/Disabled` + `<mundo>/datapacks/` | Cargar los datapacks del mundo respetando los desactivados. |
| Configs de servidor | `<mundo>/serverconfig/*.toml` | Son **por mundo** en Forge. Ej.: `lostcities-server.toml` → `selectedProfile`. |

Sin mundo, la app usa lo que el pack asigna a los **mundos nuevos**: `defaultconfigs/`
(Forge copia esos archivos a cada mundo nuevo) y, en último lugar, `config/`.

### Lost Cities

Verificado en `setup/Config.java` de Lost Cities 1.20: el Overworld usa el
`selectedProfile` de la config de servidor (`<CHECK>` = se elige al crear el mundo, vacío =
sin ciudades) y las demás dimensiones salen de `dimensionsWithProfiles` en la config común.
La ficha de Lost Cities muestra el perfil activo y, con un mundo, los biomas exactos de las
dimensiones con ciudades.

## 2. Desactivadores

Tercera puerta de enganche, con el mismo mecanismo que el descubrimiento (fases,
variantes por versión, mods requeridos). Cada desactivador informa **qué** (estructura,
bioma o criatura), **por qué** y **con qué seguridad**:

| Nivel | Significado | En el sitio |
|---|---|---|
| **Desactivado** (`Certainly`) | Lo confirman los datos o una configuración que la app entiende por completo. | Marca roja; la fuente baja al final y **no cuenta** para "mejores fuentes" ni para el resumen de biomas. Si todas las fuentes de un objeto lo están, se avisa de que no se puede conseguir. |
| **Posiblemente desactivado** (`Possibly`) | Hay indicios que la app no puede interpretar con certeza. | Marca con borde discontinuo; solo aviso. |

### Implementados

| Desactivador | Qué detecta | Nivel |
|---|---|---|
| `structure-sets` | Estructura que ningún `structure_set` coloca (un datapack la quitó o vació el set). | Seguro |
| `structure-biomes` | Estructura sin biomas; con mundo, estructuras y biomas que no existen en él. | Seguro (posible si alguna dimensión no guarda su lista de biomas) |
| `biome-replacer` | `config/biome_replacer.properties`: `bioma > null` o `bioma > otro`, también `#tags`. | Seguro |
| `structurify` | `config/structurify.json` (2.0.x): `disable_all_structures`, `is_disabled` en estructuras, sets y namespaces. | Seguro |
| `incontrol-spawns` | `config/incontrol/spawn.json`: reglas `"result": "deny"` por mob. | Seguro sin condiciones; posible con condiciones (dimensión, fase…) |
| `lostcities-profile` | Ninguna dimensión con perfil de Lost Cities. | Seguro con mundo; posible sin mundo |
| `config-mentions` | **Genérico**: un ID de estructura en `config/`, `defaultconfigs/` o `serverconfig/` del mundo cuya **clave** contiene *disable, blacklist, exclude, remove, deny, prevent, cancel, ban…* (también dentro de claves como `disabledStructures`). | Posible |
| `kubejs-scripts` | **Genérico**: un ID de estructura en scripts de KubeJS junto a esas palabras. | Posible |

Una estructura cuyos biomas están **todos** desactivados con seguridad queda también
desactivada.

Los genéricos ignoran los archivos que ya tienen un desactivador propio y las listas que
**no** desactivan la generación: las *blacklists* de Lootr solo evitan la conversión por
jugador, y `avoidStructures` de Lost Cities solo aleja las ciudades.

### Pendiente

- Mobs → biomas (spawns por bioma) para aprovechar del todo las reglas de InControl.
- Datapacks integrados en jars de mods (`mod/<id>:built_in_datapacks/...`) que el mundo
  activa o desactiva.
- Criterios de Structurify por bioma (`biome_check`, listas blancas y negras).
