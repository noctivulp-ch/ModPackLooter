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

## Fichas maestras y puntos de navegación

Toda página es de uno de dos tipos. Para decidirlo se hacen dos preguntas:

1. **¿Aquí se quedaría el jugador buscando información?**
2. **¿Es acotado?** ¿Se puede leer completo?

Si las dos respuestas son sí, es una **ficha maestra**: lo dice *todo* sin obligar
a ir a otra página. Los enlaces a otras fichas siguen siendo correctos, pero nunca
esconden datos detrás de un «ver detalles». Si no, es un **punto de navegación**:
lista, filtra y lleva a las fichas.

| Página | Tipo | Por qué |
|---|---|---|
| Objeto (y cada variante) | Ficha maestra | La pregunta más común: «¿dónde consigo esto?». Acotada. |
| Estructura | Ficha maestra | «¿Qué hay aquí?»: de qué mod es, dónde aparece, sus contenedores y qué tiene cada uno. |
| Edificio de Lost Cities | Ficha maestra | Igual que una estructura, piso por piso. |
| Criatura (mob, aldeano, comerciante) | Ficha maestra | Dónde aparece, qué suelta y con qué probabilidad, qué comercia y a qué nivel. |
| Comerciante (profesión, NPC) | Ficha maestra | Sus tradeos por nivel, precios y dónde aparece. |
| Pescar en un bioma | Ficha maestra | Junta todas las cañas: dimensión › bioma › caña › condición › captura. |
| Bioma | Punto de navegación | Un bioma entero no se puede leer completo: lleva a sus estructuras, criaturas y pesca. |
| Listas (objetos, estructuras, criaturas…), mods, tablas | Punto de navegación | Buscan y filtran; el detalle vive en las fichas. |

Si aparece una pregunta nueva del jugador que ninguna ficha contesta completa, se
crea una ficha maestra nueva con este mismo criterio.

### La jerarquía: dimensión › bioma › estructura › contenedor › objeto

Un objeto viene en un contenedor (un cofre, una criatura que lo suelta o un
comerciante que lo vende), el contenedor está en una estructura y la estructura en
un bioma de una dimensión. El **objeto** es la ficha principal; los
**contenedores** importan por lo que tienen, y las **estructuras** porque es adonde
el jugador va a buscar algo. Los biomas se agrupan por dimensión y llevan a sus
estructuras y a sus criaturas.

En la pesca, el papel de la estructura lo hace la **caña** (cada sistema de pesca:
Starcatcher, Tide, caña vanilla…). El del contenedor lo hace cada **condición**
(agua, lava o vacío; con o sin cebo; aguas abiertas…), y dentro están las
capturas. La ficha «Pescar en <bioma>» ordena así: dimensión › bioma › caña ›
condición › captura, con un índice arriba para saltar a cada caña y condición.

### La línea completa

Cuando cada eslabón tiene probabilidad, la ficha multiplica la cadena y muestra la
**probabilidad real** con su unidad. Por ejemplo: «Mundo normal › Taiga de pinos ›
Aldeano zombi 4,6 % de las apariciones de tipo hostil › Aldeano (al convertirlo) ›
Bibliotecario lo ofrece 1,8 % = 0,083 % · 1 de cada 1.200, por cada criatura
hostil que aparece en la Taiga de pinos». Se usa el bioma donde la criatura es más
probable. Las probabilidades pequeñas se muestran con dos cifras significativas y
como «1 de cada N».

### La ficha de objeto

1. **Título** destacado y, debajo, la **carta héroe**: la forma más probable de
   conseguirlo, con el número grande y su unidad («por contenedor», «al matarla»,
   «por captura», «de que te lo ofrezcan»).
2. **Cartas iguales**, una por forma de conseguirlo, con *todos* los datos y
   ordenadas de la que tiene más datos a la que menos:
   - **Estructuras**: separadas por vanilla y por mod; cada estructura con sus
     biomas y cada contenedor con su probabilidad, cantidad y condiciones.
   - **Lost Cities**: agrupadas por estilo de ciudad; edificio → parte →
     contenedor → probabilidad y pisos.
   - **Criaturas**: probabilidad al matarla, condiciones (que la mate un jugador,
     Botín…) y dónde aparece la criatura.
   - **Pesca**: agrupada por bioma; sistema, probabilidad por captura y condiciones.
   - **Tradeos y NPCs**: comerciante, nivel, precio, usos, probabilidad de que lo
     ofrezca y dónde aparece la entidad (o «no se sabe»).
   - Tablas sin origen conocido, siempre al final.
3. Variantes, biomas, cambios de mods y fuentes desactivadas.

### Dónde aparece una criatura

Se lee de los datos: los `spawners` de cada bioma, los modificadores de bioma
(`add_spawns` y `remove_spawns` de Forge y NeoForge), las criaturas y generadores de
monstruos guardados en las plantillas `.nbt` y los `spawn_overrides` de las
estructuras. A eso se suma lo que el juego hace por código y la app conoce (curar
un aldeano zombi, golems de las aldeas, asaltos…). Si no hay nada, la ficha lo
dice: «no se sabe dónde aparece». El mod companion podrá registrarlo desde el juego.

## Patrones de página

- **Barra superior**: nombre del modpack, botón de búsqueda (atajo `/`) y
  apariencia (modo Auto/Claro/Oscuro × metal Oro/Jade/Peltre). La búsqueda se abre
  en un panel translúcido que ocupa casi toda la pantalla (margen del 5 % al 10 %);
  al escribir, la barra sube y muestra los resultados al momento; elegir uno lleva
  a su ficha y cierra el panel. La portada tiene su propia barra fija.
- **Buscador inteligente**: coincidencias parciales, sin tildes y con erratas;
  `@mod` filtra por mod y `ns:id` busca por ID; también encuentra por el nombre en
  inglés.
- **Navegación lateral agrupada**: Buscar (Inicio, Objetos) · Lugares (Estructuras,
  Biomas, Lost Cities) · Otras formas (Criaturas, Pesca, Tradeos, Otras fuentes) ·
  El modpack (Cambios de mods, Mods, Acerca). Las pestañas de mods solo aparecen
  si hay datos.
- **Barra de probabilidad**: el porcentaje siempre en texto; la barra es apoyo visual.
- **Confianza**: etiqueta textual + símbolo, nunca solo color.
- **Estados vacíos** que explican por qué no hay datos y qué hacer.
