# ADR-0004: App externa; mod “volcador” opcional en el futuro

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Hay que soportar Minecraft 1.20.1 → 26.3 con Forge y NeoForge (Fabric más
adelante). La alternativa era hacer un mod multicargador y multiversión.
Investigación completa: [08-investigacion-app-externa-vs-mod.md](../08-investigacion-app-externa-vs-mod.md).

## Decisión

ModPackLooter es una **aplicación externa** que analiza los datos de los
archivos del modpack sin ejecutar el juego.

Más adelante se podrá crear un **mod volcador mínimo** que exporte el estado
resuelto en tiempo de ejecución (loot tables, estructuras, biomas y tags) a
JSON con formato de datapack. La app lo consumirá como una fuente más
(`RuntimeDumpSource`).

## Alternativas consideradas

- **Mod multicargador/multiversión completo** (MultiLoader-Template o
  Architectury + Stonecutter): precisión máxima, pero matriz de compilación
  *versión × cargador*, Java 17/21/25, cambios de mappings y APIs internas,
  y hay que arrancar el juego para usarlo.
- **Solo app externa, sin plan de mod**: más simple, pero renuncia para
  siempre al loot definido por código.

## Consecuencias

- Soportar una versión nueva = ajustar un adaptador de formato de datos.
- La ofuscación y las versiones de Java no afectan.
- Se puede ejecutar en CI para publicar en GitHub/GitLab Pages.
- El loot definido por código o scripts no es visible: se cubre con
  heurísticas, tabla de conocimiento vanilla, overrides y, en el futuro, el
  volcador.
