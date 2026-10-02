# 05 · Interfaz de línea de comandos

> Borrador de la experiencia de uso. Nombres y flags pueden cambiar cuando se
> elijan las tecnologías.

## Comandos

```bash
# Genera el sitio estático (comando principal)
modpacklooter build <ruta-modpack> --out ./site

# Analiza y muestra un resumen + diagnósticos, sin generar el sitio
modpacklooter scan <ruta-modpack>

# Consultas rápidas desde la terminal (reutilizan el mismo índice)
modpacklooter where <objeto>          # ¿dónde aparece este objeto?
modpacklooter structure <estructura>  # ¿qué loot tiene esta estructura?
modpacklooter biome <bioma>           # ¿qué estructuras con loot hay aquí?

# Exporta el índice a JSON para otras herramientas
modpacklooter export <ruta-modpack> --format json --out loot.json

# Sirve el sitio generado en local para previsualizarlo
modpacklooter serve ./site

# Crea un archivo de configuración comentado en el directorio actual
modpacklooter init

# Además genera el workflow para publicar en GitHub Pages o GitLab Pages
modpacklooter init --ci github
modpacklooter init --ci gitlab
```

## Opciones comunes

| Opción | Descripción |
|---|---|
| `--mc-version <v>` | Fuerza la versión de Minecraft si no se detecta. |
| `--minecraft-jar <ruta>` | Jar de vanilla para incluir loot, nombres e iconos base. |
| `--download-vanilla` | Descarga el jar vanilla del manifiesto oficial de Mojang y lo guarda en caché. |
| `--loader forge\|neoforge\|fabric` | Fuerza el cargador si no se detecta. |
| `--lang <código>` | Idioma de los nombres (`es_es`, `en_us`…). |
| `--config <archivo>` | Archivo de configuración del proyecto. |
| `--include-mod / --exclude-mod` | Filtrar mods. |
| `--format text\|json` | Salida legible o para máquinas (en comandos de consulta). |
| `-q / -v` | Menos / más detalle en la salida. |

## Archivo de configuración (borrador)

Permite reproducir la misma generación y guardar los *overrides*:

```
[site]
title = "Guía de loot de MiModpack"
language = "es_es"

[sources]
minecraft_jar = "~/.minecraft/versions/1.20.1/1.20.1.jar"
exclude_mods = ["debugmod"]

[[overrides.structure_loot]]
structure = "examplemod:ruined_tower"
loot_tables = ["examplemod:chests/ruined_tower_top"]
```

## Comportamiento esperado

- Detecta automáticamente el tipo de entrada (carpeta de instancia, `.zip`,
  `.mrpack`).
- Muestra progreso por etapas y, al final, un resumen: nº de mods,
  estructuras, tablas, objetos y advertencias.
- Códigos de salida: `0` éxito, `1` error fatal, `2` éxito con advertencias
  (opcional vía `--strict`).

## Reglas de terminal (guía tui-architect)

- **Sin TTY = salida plana**: sin colores, spinners ni cursor si `stdout` no es
  una terminal (CI, tuberías). Se respetan `NO_COLOR`, `FORCE_COLOR` y
  `TERM=dumb`.
- **`--json`** en todos los comandos de consulta y en `scan`, para scripts.
- **stdout para resultados, stderr para progreso y logs**; `--debug` escribe
  un log detallado a archivo.
- **Color semántico y escaso** (acento, ok, aviso, error, atenuado), siempre
  acompañado de un símbolo (`✓ ✗ !`), con opción `--ascii`.
- **Mensajes de error** en una línea: qué pasó + cómo arreglarlo.
- `--version` y `--help` útiles desde el primer día.
- La TUI será otra capa sobre el mismo núcleo; el modo CLI no es un
  *fallback*, es de primera clase.

## Futura TUI

`modpacklooter tui <ruta-modpack>` abrirá una interfaz interactiva para
explorar el índice (buscar objetos, navegar estructuras/biomas) y lanzar la
generación del sitio. Reutilizará exactamente los mismos casos de uso que la
CLI; solo cambia la capa de presentación.
