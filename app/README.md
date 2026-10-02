# ModPackLooter · app

Programa de línea de comandos en Go. Analiza un modpack de Minecraft y genera un sitio
estático con todas sus fuentes de loot. Arquitectura y decisiones en
[`../docs`](../docs/README.md).

## Descargar o compilar

- **Releases**: binarios para Windows, macOS y Linux en las releases del repositorio
  (etiquetas `app/vX.Y.Z`). No necesitan instalar nada.
- **Compilar** (Go 1.24 o superior):

  ```bash
  cd app
  go build -o modpacklooter ./cmd/modpacklooter    # en Windows: -o modpacklooter.exe
  ```

## Uso

```bash
# Genera el sitio en ./site a partir de la carpeta de la instancia
modpacklooter build "C:\Users\yo\curseforge\minecraft\Instances\DeceasedCraft" --out site

# Solo analiza y muestra un resumen con los avisos
modpacklooter scan  "C:\...\Instances\DeceasedCraft"

# Qué descubridores y ajustes se aplicarían a una versión/cargador/mods
modpacklooter plan --mc-version 1.20.1 --loader forge --mod lootr --mod lostcities
```

Abre `site/index.html` en el navegador. El sitio funciona sin servidor y se puede subir tal
cual a GitHub Pages o GitLab Pages.

### Opciones

| Opción | Descripción |
|---|---|
| `--out, -o` | Carpeta de salida (`site` por defecto). |
| `--title` | Título del sitio (por defecto, el nombre de la carpeta). |
| `--metal` | Acento visual de Wulfenite UI: `oro`, `jade` o `peltre`. |
| `--lang` | Idioma de los nombres (`es_es`, `es_ar`, `en_us`…). Por defecto el del juego (`options.txt`) o `es_es`; si falta una traducción se usa otra variante del mismo idioma y luego inglés. |
| `--minecraft-jar` | Jar de Minecraft vanilla (o carpeta con `data/`). Se busca solo en CurseForge, Prism y el launcher oficial. |
| `--assets-dir` | Carpeta `assets` del launcher, para traducir los nombres vanilla. |
| `--mc-version`, `--loader` | Fuerzan la versión y el cargador si no se detectan. |
| `--datapack` | Datapack extra (carpeta o `.zip`); repetible. |
| `--json` | (`scan`, `plan`) salida para scripts. |

Los avisos y el progreso van a *stderr*; el resultado, a *stdout*.

## Qué detecta hoy (Minecraft 1.20.x, Forge/NeoForge)

- Cofres, barriles, vagonetas y arqueología de **plantillas NBT** y **processor lists**
  (`append_loot`) de cualquier mod con estructuras jigsaw.
- Estructuras vanilla cuyo loot asigna el código (pirámides, minas, fortalezas…).
- **Lost Cities**: paletas y condiciones de loot con su peso.
- **Lootr**: loot por jugador, *refresh*, *decay* y exclusiones según su configuración.
- Heurística por nombre y, al final, todas las tablas restantes clasificadas por su ruta.

## Desarrollo

```bash
go test ./...                                   # tests
MPL_MCMETA_DATA=/ruta/mcmeta-1.20.1-data go test ./...   # + tests con datos vanilla reales
```

`MPL_MCMETA_DATA` apunta a un clon de `misode/mcmeta` (rama `1.20.1-data`).

```
cmd/modpacklooter/       raíz de composición
internal/domain/         conceptos del dominio
internal/mcversion/      versiones de Minecraft (1.20.1 … 26.3) y rangos
internal/modpack/        abre la instancia: mods, datapacks, kubejs, vanilla, idioma
internal/resources/      índice fusionado y layouts de carpetas por versión
internal/nbt/            lector NBT propio
internal/loot/           loot tables y probabilidades
internal/worldgen/       estructuras, pools, processor lists, plantillas, tags
internal/discovery/      puertas de descubrimiento y enriquecimiento (+ un paquete por plugin)
internal/plugins/        registro de todos los plugins
internal/analysis/       caso de uso: analizar un modpack
internal/names/          nombres traducidos e idiomas
internal/site/           generador del sitio (plantillas y assets embebidos)
internal/cli/            adaptador de línea de comandos
```
