# ADR-0011: Guías de referencia para las interfaces

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

El proyecto tiene dos interfaces: el **sitio web generado** (lo que usan los
jugadores) y la **CLI/TUI** (lo que usa quien genera el sitio). El usuario
aportó dos guías (skills) para usarlas como referencia.

## Decisión

- **impeccable**: referencia para el diseño, la UX, la accesibilidad, el
  copy, los estados vacíos o de error y la crítica del **sitio generado**
  (y del código frontend).
- **tui-architect**: referencia para la **CLI y la futura TUI**: núcleo sin
  UI, degradación sin TTY, `--json`, logging aparte, color semántico, Bubble
  Tea v2 y pruebas con `teatest`.
- Lo que impeccable dice de tipografía, color o animación **no se aplica
  literalmente** a la terminal; se traduce la intención.

## Consecuencias

- Las reglas concretas quedan reflejadas en [03-sitio-generado.md](../03-sitio-generado.md)
  y [05-cli.md](../05-cli.md).
- Las reviews del sitio se pueden contrastar con los criterios de impeccable.
