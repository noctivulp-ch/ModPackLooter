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
| [07-tecnologias.md](07-tecnologias.md) | **Pendiente de discutir.** Criterios y preguntas abiertas. |
| [decisiones/](decisiones/) | Registro de decisiones de arquitectura (ADR). |

## Cómo documentamos decisiones

Usamos ADR (*Architecture Decision Records*) ligeros. Cada decisión es un
archivo numerado en [`decisiones/`](decisiones/) creado a partir de la
[plantilla](decisiones/0000-plantilla.md). Una decisión nunca se borra: si
cambia, se crea una nueva que la **reemplaza** y se marca la antigua como
`Reemplazada por ADR-XXXX`.
