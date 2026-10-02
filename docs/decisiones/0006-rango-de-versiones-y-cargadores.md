# ADR-0006: Rango de versiones y cargadores soportados

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Los modpacks objetivo van desde 1.20.1 hasta la versión más reciente (26.3).
El modpack de referencia usa Forge 1.20.1.

## Decisión

- **Meta de versiones:** 1.20.1 → 26.3 (y siguientes).
- **Orden de soporte:** 1.20.1 (MVP) → 1.21.1 → 26.x.
- **Cargadores:** Forge y NeoForge primero; Fabric en el futuro.
- Las diferencias se aíslan en **adaptadores de formato por rango de
  versiones** y **adaptadores por cargador**. El dominio es único.

Detalle en [09-versiones-y-cargadores.md](../09-versiones-y-cargadores.md).

## Consecuencias

- Cada versión nueva requiere fixtures y, si cambia el formato, un adaptador.
- Fabric aportará poco loot visible en datos (sus mods suelen modificar loot
  por código); el mod volcador será más valioso allí.
