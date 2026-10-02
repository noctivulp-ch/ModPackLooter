# ModPackLooter

Genera un **sitio web estático** con todas las fuentes de loot de un modpack de
Minecraft: en qué estructuras y biomas aparece cada objeto, qué hay en cada
estructura y qué se puede saquear en cada bioma.

| Carpeta | Contenido |
|---|---|
| [`app/`](app/) | Programa principal (Go, CLI; TUI en el futuro). |
| [`mod/`](mod/) | Futuro mod “volcador” que exporta datos que la app no puede leer de los archivos. |
| [`docs/`](docs/) | Visión, arquitectura, investigación y decisiones (ADR). |

Empieza por [`docs/README.md`](docs/README.md).
