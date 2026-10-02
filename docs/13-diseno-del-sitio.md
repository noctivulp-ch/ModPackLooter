# 13 · Diseño visual del sitio generado

> Decisión: [ADR-0012](decisiones/0012-sitio-con-wulfenite-ui.md). Complementa
> [03-sitio-generado.md](03-sitio-generado.md) (qué contiene) con el **cómo se ve**.

## Contexto de uso

- **PC primero**: el jugador consulta en una segunda pantalla o con alt-tab. Se prioriza la
  densidad: tablas anchas, navegación lateral siempre visible, búsqueda en la barra superior.
- **Móvil aceptable**: por debajo de 1024 px la navegación pasa arriba; por debajo de 600 px
  todo va en una columna y las tablas se desplazan en horizontal dentro de su panel.
- Modo de interfaz: *Operate* (el visitante completa una tarea: encontrar un objeto).

## Sistema de diseño: Wulfenite UI

El sitio usa los tokens del sistema **Wulfenite UI** (artefacto del usuario), traducidos a CSS:

| Concepto Wulfenite | En el sitio |
|---|---|
| `bandeja` (marco de laca) | Fondo de la página, barra superior y navegación lateral. |
| `pozo` (interior) | Paneles de contenido: fichas, tablas, resultados. |
| `canto-<metal>` 1 px | Borde de cada panel: la profundidad la marca el canto, **nunca sombras**. |
| `metal-<metal>` | Acento: enlace activo, selección, barras de probabilidad, foco. |
| `pared` 12 px | Separación entre paneles. |
| `radio-bandeja` 20 px / `radio-pozo` 4 px | Geometría invertida: exterior suave, paneles rectos. |
| `pozo-contacto` | Degradado duro en el 7 % superior de cada panel. |
| Bordes 1/2/3 px | Reposo, seleccionado y foco (el foco siempre se ve). |

- **Modos**: oscuro (laca negra y roja) y claro (laca blanca y verde). Por defecto sigue
  `prefers-color-scheme`; el visitante puede cambiarlo y se recuerda en su navegador.
- **Metales**: oro (por defecto), jade y peltre. El creador del modpack elige el metal con
  `--metal`; el visitante también puede cambiarlo.
- **Tipografía**: la del sistema operativo con la escala del sistema (40/24/20/16/14/12).
- **Objetivo táctil** mínimo de 48 px en controles (relevante en móvil).

## Patrones de página

- **Barra superior**: nombre del modpack, buscador global (atajo `/`), modo y metal.
- **Navegación lateral**: Objetos, Estructuras, Biomas, Otras fuentes, Mods, Acerca.
- **Ficha de objeto**: cabecera, "mejores fuentes" y tabla completa con probabilidad,
  cantidad, confianza y notas (Lootr, encantado…).
- **Barra de probabilidad**: el porcentaje siempre en texto; la barra es apoyo visual.
- **Confianza**: etiqueta textual + símbolo, nunca solo color.
- **Estados vacíos** que explican por qué no hay datos y qué hacer.
