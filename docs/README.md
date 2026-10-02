# Documentación de ModPackLooter

Esta carpeta recoge la visión del proyecto y todas las decisiones que se tomen
durante su desarrollo. La regla es simple: **si una decisión cambia cómo se
construye o se usa la app, queda escrita aquí**.

## Índice

| Documento | Contenido |
|---|---|
| [01-vision.md](01-vision.md) | Qué es la app, qué problema resuelve y qué produce. |
| [02-dominio.md](02-dominio.md) | Conceptos de Minecraft que maneja la app y cómo se relacionan. |
| [03-sitio-generado.md](03-sitio-generado.md) | Cómo deben verse y comportarse las páginas web generadas. |
| [04-arquitectura.md](04-arquitectura.md) | Arquitectura desacoplada, capas, puertos y aplicación de SOLID. |
| [05-cli.md](05-cli.md) | Interfaz de línea de comandos (y cómo encaja la futura TUI). |
| [06-alcance-y-hoja-de-ruta.md](06-alcance-y-hoja-de-ruta.md) | MVP, fases siguientes y riesgos conocidos. |
| [07-tecnologias.md](07-tecnologias.md) | Comparativa de lenguajes y pila elegida (Go). |
| [08-investigacion-app-externa-vs-mod.md](08-investigacion-app-externa-vs-mod.md) | Investigación: app externa frente a mod multicargador/multiversión. |
| [09-versiones-y-cargadores.md](09-versiones-y-cargadores.md) | Diferencias de formato 1.20.1 → 26.3 y entre Forge/NeoForge/Fabric. |
| [10-proceso-de-pruebas-y-entrega.md](10-proceso-de-pruebas-y-entrega.md) | Ciclo de review con el modpack real, releases y GitHub/GitLab Pages. |
| [11-lootr.md](11-lootr.md) | Cómo funciona Lootr y cómo se soporta. |
| [12-arquitectura-de-descubrimiento.md](12-arquitectura-de-descubrimiento.md) | Puerta de enganche: descubridores encadenados, reclamaciones y genérico final. |
| [13-diseno-del-sitio.md](13-diseno-del-sitio.md) | Diseño visual del sitio con Wulfenite UI, PC primero. |
| [14-mundo-modelo-y-desactivadores.md](14-mundo-modelo-y-desactivadores.md) | `--world` y la puerta de desactivadores (seguro / posible). |
| [15-pestanas-por-mod-y-lost-cities.md](15-pestanas-por-mod-y-lost-cities.md) | Pestañas dedicadas por mod (solo si está) y Lost Cities por capas; nombres desde `lang`. |
| [16-pesca.md](16-pesca.md) | Pestaña de pesca: Starcatcher, Tide 2 y vanilla, por bioma y condiciones. |
| [decisiones/](decisiones/) | Registro de decisiones de arquitectura (ADR). |

## Cómo documentamos decisiones

Usamos ADR (*Architecture Decision Records*) ligeros. Cada decisión es un
archivo numerado en [`decisiones/`](decisiones/) creado a partir de la
[plantilla](decisiones/0000-plantilla.md). Una decisión nunca se borra: si
cambia, se crea una nueva que la **reemplaza** y se marca la antigua como
`Reemplazada por ADR-XXXX`.

## Decisiones registradas

| ADR | Decisión |
|---|---|
| [0001](decisiones/0001-documentar-decisiones-con-adr.md) | Documentar decisiones con ADR en `docs/`. |
| [0002](decisiones/0002-cli-primero-nucleo-independiente.md) | CLI primero, con un núcleo independiente de la interfaz. |
| [0003](decisiones/0003-salida-sitio-estatico.md) | La salida principal es un sitio web estático. |
| [0004](decisiones/0004-app-externa-con-volcador-opcional.md) | App externa; mod volcador opcional en el futuro. |
| [0005](decisiones/0005-lenguaje-go.md) | Implementar la app en Go. |
| [0006](decisiones/0006-rango-de-versiones-y-cargadores.md) | 1.20.1 → 26.3; Forge y NeoForge primero, Fabric después. |
| [0007](decisiones/0007-distribucion-y-publicacion.md) | GitHub Releases + sitio compatible con GitHub/GitLab Pages. |
| [0008](decisiones/0008-ciclo-de-iteracion-con-modpack-real.md) | Ciclo de review con DeseacedCraft BetaCerrada. |
| [0009](decisiones/0009-estructura-del-repositorio.md) | Monorepo: `app/` (Go) y `mod/` (futuro mod). |
| [0010](decisiones/0010-descubrimiento-por-ganchos-encadenados.md) | Descubrimiento con piezas enganchables encadenadas y genérico final. |
| [0011](decisiones/0011-guias-de-diseno-de-interfaces.md) | Guías de referencia: impeccable (sitio) y tui-architect (CLI/TUI). |
| [0012](decisiones/0012-sitio-con-wulfenite-ui.md) | El sitio usa Wulfenite UI; PC primero, móvil aceptable; español por defecto. |
| [0013](decisiones/0013-mundo-modelo.md) | Mundo modelo (`--world`) en lugar de un menú de creación de mundos. |
| [0014](decisiones/0014-desactivadores-con-dos-niveles.md) | Desactivadores con dos niveles: desactivado / posiblemente desactivado. |
| [0015](decisiones/0015-pestanas-por-mod.md) | Pestañas dedicadas por mod, solo si el mod está; nombres desde cualquier `lang`. |
| [0016](decisiones/0016-pesca-por-fuente.md) | Pesca agrupada por sistema de pesca, cada uno con sus reglas. |
