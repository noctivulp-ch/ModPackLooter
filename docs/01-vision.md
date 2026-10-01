# 01 · Visión de la aplicación

## En una frase

**ModPackLooter** es una herramienta de línea de comandos que analiza un
modpack de Minecraft y genera un **sitio web estático** que responde, de forma
clara y rápida, a la pregunta: *“¿dónde consigo este objeto?”*.

## El problema

En un modpack con decenas o cientos de mods, el loot está repartido entre:

- cofres de estructuras (vanilla y de mods),
- tablas de loot de mobs, pesca, arqueología, bóvedas, etc.,
- modificaciones hechas por datapacks, KubeJS, CraftTweaker o *Global Loot
  Modifiers*.

Esa información está enterrada en archivos JSON y NBT dentro de los `.jar` de
cada mod. Las wikis no cubren combinaciones concretas de mods ni versiones, y
JEI/REI/EMI muestran recetas, pero no responden bien a “¿en qué estructura y en
qué bioma aparece esto?”.

## La solución

1. El usuario apunta la CLI a la carpeta (o al archivo exportado) de su
   modpack.
2. La app **extrae** los datos de loot, estructuras y biomas de todos los mods,
   datapacks y scripts que pueda leer.
3. **Resuelve** las relaciones: qué estructura usa qué tablas de loot, en qué
   biomas puede generarse cada estructura y qué objetos contiene cada tabla
   (con probabilidades).
4. **Genera** un sitio web estático: archivos HTML/CSS/JS que se abren sin
   servidor, se pueden publicar en GitHub Pages o compartir en un `.zip`.

## Preguntas que el sitio debe responder

| Pregunta del jugador | Vista que la responde |
|---|---|
| ¿Dónde encuentro *X* objeto? | Ficha del **objeto**: estructuras, biomas y fuentes donde aparece, ordenadas por probabilidad. |
| ¿Qué puedo encontrar en *Y* estructura? | Ficha de la **estructura**: tablas de loot, objetos y biomas donde se genera. |
| ¿Qué estructuras con loot hay en *Z* bioma? | Ficha del **bioma**: estructuras y fuentes de loot agrupadas. |
| ¿Qué añade el mod *M*? | Ficha del **mod**: sus estructuras, tablas de loot y objetos. |
| ¿Cuál es la mejor fuente para *X*? | Ranking de fuentes por probabilidad por cofre / por tirada. |

## Principios del producto

1. **Intuitivo antes que completo.** El sitio debe poder usarse sin leer
   instrucciones: una barra de búsqueda grande y resultados legibles.
2. **Nombres humanos.** Se muestra “Templo del desierto” / “Desert Pyramid” y
   el icono del objeto, no `minecraft:chests/desert_pyramid`. El identificador
   técnico queda disponible como dato secundario.
3. **Honestidad con los datos.** Si una relación se dedujo por heurística o
   no se pudo resolver (p. ej. loot añadido por código), el sitio lo indica en
   lugar de inventarlo.
4. **Estático y portable.** Cero backend. El sitio funciona offline y desde
   `file://`.
5. **Reproducible.** Mismo modpack + misma configuración = mismo sitio.

## Principios de ingeniería

1. **CLI primero, TUI después.** Toda la lógica vive en un núcleo
   independiente de la interfaz; la CLI es solo una “piel” sobre él, y la TUI
   futura será otra (ver [ADR-0002](decisiones/0002-cli-primero-nucleo-independiente.md)).
2. **SOLID y bajo acoplamiento.** El dominio no conoce ni archivos `.jar`, ni
   HTML, ni la terminal (ver [04-arquitectura.md](04-arquitectura.md)).
3. **Extensible por adaptadores.** Nuevos loaders (Forge, NeoForge, Fabric,
   Quilt), nuevas versiones de Minecraft o nuevos formatos de salida se añaden
   implementando una interfaz, sin tocar el núcleo.
4. **Decisiones documentadas.** Toda decisión relevante queda en
   [`decisiones/`](decisiones/).

## Usuarios objetivo

- **Jugadores** de un modpack que buscan un objeto concreto.
- **Creadores de modpacks** que quieren publicar una guía de loot junto a su
  pack o revisar el balance del loot.
- **Servidores** que quieren ofrecer una wiki de loot a su comunidad.

## Fuera de alcance (por ahora)

- Ejecutar Minecraft o cargar los mods en tiempo de ejecución.
- Recetas de crafteo (eso ya lo resuelven JEI/REI/EMI).
- Edición de loot tables (la app es de solo lectura).
- Un servidor web con base de datos.
