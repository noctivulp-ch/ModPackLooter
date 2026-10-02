# 15 · Pestañas por mod y Lost Cities por capas

## Pestañas que solo existen si el mod está

Algunos mods tienen una lógica propia que no cabe en «estructura → loot»: Lost
Cities arma ciudades por capas; la pesca de Starcatcher o Tide depende de cebo,
bioma y clima. Para ellos el sitio tiene **pestañas dedicadas**.

Regla ([ADR-0015](decisiones/0015-pestanas-por-mod.md)): **una pestaña de mod no
aparece si el mod no está presente** (ni en la navegación, ni en la búsqueda, ni
como carpeta del sitio). No hay pestañas vacías ni mensajes de «este mod no está».

Cómo funciona:

1. El descubridor del mod (que ya solo se ejecuta si el mod está, por su
   `Applicability.RequiresMods`) arma su modelo y lo **adjunta** a las
   reclamaciones con `Claims.Attach(<id del descubridor>, modelo)`, solo si hay
   contenido.
2. El análisis lo expone en `Result.Extras`.
3. El sitio busca su clave en `Extras`; si está, añade una `Section` a la
   navegación y genera sus páginas. Si no, no hace nada.

Así el núcleo no conoce a ningún mod concreto y cada pestaña nueva es una pieza
más (descubridor + vista), sin tocar el resto.

## Lost Cities: LostCities › Estilos › Edificios › Contenedores › Loot

Modelo leído de los datos de Lost Cities 1.20 (`cityassets/*.java`) y del port
de DeceasedCraft:

| Capa | Archivo | Qué se toma |
|---|---|---|
| Perfil | `config/lostcities/profiles/<perfil>.json` | `worldStyle` (la sección cambió entre versiones; se busca en cualquiera). Perfil por dimensión: `lostcities-server.toml` (`selectedProfile`, mundo → `defaultconfigs/` → `config/`) y `dimensionsWithProfiles` de la configuración común. |
| Estilo de mundo | `lostcities/worldstyles/*.json` | `citystyles` (factor + regla de biomas `if_any` / `if_all` / `excluding`) y `scattered.list` (edificios sueltos). |
| Estilo de ciudad | `lostcities/citystyles/*.json` | `selectors.buildings` / `multibuildings` con su factor; `inherit` suma los del padre. |
| Edificio | `lostcities/buildings/*.json`, `multibuildings/*.json` | Partes (`parts`, `parts2`) y paleta (nombre, lista en línea u objeto `{"palette": [...]}`). Los múltiples son una rejilla de edificios. |
| Parte | `lostcities/parts/*.json` | `slices`: se cuenta cuántas veces aparece cada carácter. |
| Paleta | parte (`palette`) → `refpalette` → paleta del edificio → paletas de los estilos (`styles/*`) | Carácter → bloque + `loot` (una condición). |
| Condición | `lostcities/conditions/*.json` | `values`: tabla de loot con su `factor` y restricciones (`range` de pisos, `inpart`, `inbuilding`, `inbiome`). |

Páginas:

- `lostcities/`: dónde hay ciudades (perfil por dimensión), cada estilo de
  mundo con sus estilos de ciudad por bioma (los grupos de biomas se despliegan
  en sus biomas) y los edificios sueltos.
- `lostcities/estilos/`: estilos de ciudad; cada uno con sus edificios, la
  probabilidad de cada uno por solar y el loot posible.
- `lostcities/edificios/`: todos los edificios; cada uno con sus partes, los
  contenedores de cada parte (cantidad) y, para cada contenedor, las tablas de
  loot posibles con su probabilidad y restricciones («pisos 2 a 5»).

Probabilidades: la del edificio es su factor sobre el total del estilo de ciudad;
la de la tabla, el factor del valor sobre los valores de la condición que
aplican a esa parte y ese edificio. El peso de un estilo de ciudad se muestra
tal cual porque la elección depende de qué estilos admiten el bioma.

Las fuentes de loot del resto del sitio (ficha de objeto, «Lost Cities
(edificios de la ciudad)» en Estructuras) se mantienen: la pestaña es una vista
más detallada, no un reemplazo.

## Nombres para personas, no IDs

Quien lee el sitio no tiene por qué ser técnico, así que se usa **cualquier
clave de idioma que nombre la cosa** antes de mostrar el ID
(`names.Namer.Asset`):

1. Claves explícitas: `<tipo>.<ns>.<ruta>`, `<ns>.<tipo>.<ruta>`,
   `<ns>.<tipo>s.<ruta>`, `lostcities.<tipo>.<ruta>`.
2. Cualquier clave del mismo espacio de nombres que termine en `.<ruta>` y
   parezca un nombre (`title`, `name`). Por ejemplo, DeceasedCraft da a cada
   distrito un logro: `deceasedcraft.advancement.title.retail_district` →
   «Distrito: Comercial».
3. Cualquier otra clave que termine igual, salvo descripciones y *tooltips*.
4. Si nada sirve, el ID legible sin el prefijo del tipo
   (`citystyle_desert` → «Desert»).

Los idiomas se prueban en el orden de la cadena de respaldo (pedido → variante
principal → otras variantes → `en_us`), **clave por clave**: si falta un texto
en `es_ar` se busca en `es_es`, luego en `es_mx`…, y por último en inglés. Un
valor vacío cuenta como ausente. Una clave encontrada en el idioma preferido
gana a una clave «mejor» de un idioma de respaldo. Los IDs técnicos siguen
visibles, en pequeño, bajo el título.

Los grupos de biomas (`#minecraft:is_ocean`) se nombran con
`tag.worldgen.biome.<ns>.<ruta>` si existe, o con su ruta sin `is_` («Ocean»).
