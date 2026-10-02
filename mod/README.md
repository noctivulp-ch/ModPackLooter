# ModPackLooter · mod volcador (futuro)

Mod mínimo que, ejecutado dentro del juego, exportará a JSON (con formato de
datapack) el estado **ya resuelto**: loot tables, estructuras, biomas y tags,
incluido lo que otros mods o scripts cambian por código. La app lo leerá como
una fuente más.

Todavía no hay código. Plan previsto:

- MultiLoader-Template (`common/` + `forge/` + `neoforge/`, `fabric/` más
  adelante) y Stonecutter para varias versiones.
- Superficie de API mínima: recorrer registros y serializar con los codecs
  del juego.

Contexto: [ADR-0004](../docs/decisiones/0004-app-externa-con-volcador-opcional.md)
y [ADR-0009](../docs/decisiones/0009-estructura-del-repositorio.md).
