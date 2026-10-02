# 07 · Tecnologías

> Estado: **lenguaje decidido** ([ADR-0005](decisiones/0005-lenguaje-go.md)).
> Las librerías concretas se confirmarán al crear el esqueleto del proyecto.

## Requisitos que guían la elección

1. Leer muchos `.jar`/`.zip` (incluido jar-in-jar) y JSON, y decodificar
   **NBT** comprimido.
2. **Binario autocontenido** para Windows, macOS y Linux (unas pocas
   dependencias externas son aceptables).
3. CLI ahora y **TUI** de calidad después.
4. Facilidad para una arquitectura hexagonal y SOLID (interfaces, inyección).
5. Generar un sitio estático e incrustar las plantillas y assets en el
   binario.
6. Release automatizado en GitHub y ejecución en CI de GitHub/GitLab.

## Comparativa de candidatos

| Criterio | **Go** | Kotlin/Java (JVM) | Rust | Python |
|---|---|---|---|---|
| Binario autocontenido | ★★★ nativo, *cross-compile* trivial | ★★ GraalVM native-image (lento, configuración de reflexión) o exigir JRE | ★★★ nativo; *cross-compile* más delicado | ★ PyInstaller, frágil |
| Zip/JSON/gzip | ★★★ librería estándar | ★★★ | ★★★ crates | ★★★ |
| NBT | ★★ varias librerías; lector propio de ~200 líneas también es viable | ★★★ (ecosistema MC) | ★★★ `fastnbt` | ★★★ `nbtlib` |
| CLI | ★★★ Cobra | ★★★ Clikt / picocli | ★★★ clap | ★★★ Typer |
| TUI | ★★★ **Bubble Tea** (referencia del sector) | ★★ Mosaic / Lanterna | ★★★ Ratatui | ★★★ Textual |
| Plantillas HTML incrustadas | ★★★ `html/template` + `embed` | ★★ | ★★ | ★★ |
| Interfaces / desacople | ★★★ interfaces implícitas, sencillas | ★★★ | ★★ (traits, más fricción) | ★★ (tipado opcional) |
| Curva de aprendizaje / velocidad | ★★★ | ★★ | ★ | ★★★ |
| Compartir código con un futuro mod | ✗ (se comparte un **contrato JSON**) | ★★★ | ✗ | ✗ |

## Decisión

**Go**. Es el que mejor cumple los requisitos prioritarios: binario único
multiplataforma, librería estándar que cubre zip/JSON/gzip/HTML, *embed* para
el sitio y Bubble Tea para la futura TUI.

La única ventaja fuerte del JVM era compartir código con un mod. No es
necesaria: el mod volcador futuro ([ADR-0004](decisiones/0004-app-externa-con-volcador-opcional.md))
se comunica con la app mediante **JSON con formato de datapack**, que es un
contrato independiente del lenguaje.

## Pila prevista (a confirmar en el esqueleto)

| Necesidad | Opción prevista |
|---|---|
| Lenguaje | Go (última versión estable) |
| CLI | `spf13/cobra` (+ `charmbracelet/fang` para ayuda/errores con estilo, a evaluar) |
| TUI (fase 3) | **Bubble Tea v2** + Lip Gloss v2 + Bubbles v2 (`charm.land/...`), Huh para formularios |
| NBT | Librería existente o lector propio mínimo detrás del puerto `StructureTemplateReader` |
| Zip / JSON / gzip | Librería estándar |
| Configuración | TOML (`BurntSushi/toml` o `pelletier/go-toml`) |
| Plantillas del sitio | `html/template` + `embed` |
| Frontend del sitio | HTML + CSS propio, JS mínimo sin framework |
| Búsqueda en el navegador | Índice JSON precalculado + búsqueda difusa ligera (p. ej. MiniSearch o una propia) incluida en el sitio |
| Tests | `testing` + *golden files* |
| Lint / formato | `gofmt`, `go vet`, `golangci-lint` |
| Releases | GoReleaser en GitHub Actions |

## Pendiente

- Elegir la librería NBT tras probarla con plantillas reales de 1.20.1.
- Elegir la librería de búsqueda en el cliente según el tamaño del índice de
  DeseacedCraft.
