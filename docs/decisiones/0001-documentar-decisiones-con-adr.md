# ADR-0001: Documentar las decisiones con ADR en `docs/`

- **Estado:** Aceptada
- **Fecha:** 2026-10-01

## Contexto

El proyecto empieza desde cero y se quiere conservar el razonamiento detrás de
cada decisión (arquitectura, tecnologías, alcance) para poder revisarlo más
adelante.

## Decisión

La documentación vive en el repositorio, en `docs/`, en Markdown. Las
decisiones se registran como ADR numerados en `docs/decisiones/`, usando
`0000-plantilla.md`.

## Alternativas consideradas

- **Wiki externa** — fácil de editar, pero se desincroniza del código.
- **Solo comentarios en issues/PR** — difíciles de encontrar después.

## Consecuencias

- Las decisiones se revisan junto al código en los mismos PR.
- Una decisión no se borra: se reemplaza con un ADR nuevo.
