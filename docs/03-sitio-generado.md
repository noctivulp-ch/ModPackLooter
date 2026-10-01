# 03 · El sitio web generado

El sitio es el producto que ve el jugador. Objetivo: **encontrar un objeto en
menos de 5 segundos** y entender el resultado sin conocimientos técnicos.

## Principios de diseño

- **La búsqueda es la portada.** Barra grande, con autocompletado por nombre
  traducido, ID técnico y mod.
- **Iconos siempre.** Cada objeto, estructura y bioma con su icono o un
  marcador de color consistente.
- **Probabilidades legibles.** “≈ 23 % por cofre · 1–3 unidades” en lugar de
  pesos crudos. Barras visuales para comparar fuentes.
- **Enlaces en todas direcciones.** Desde un objeto se salta a la estructura,
  desde ella al bioma, y de vuelta.
- **Agrupar, no listar.** Las listas largas se agrupan (por mod, dimensión,
  tipo de fuente) y se pueden filtrar.
- **Responsive y accesible.** Usable en móvil (consultar mientras se juega),
  modo claro/oscuro, contraste suficiente, navegable con teclado.
- **Sin servidor.** Funciona abriendo `index.html` directamente; la búsqueda
  usa un índice precalculado incluido en el sitio.

## Mapa del sitio

```
/                       Inicio: búsqueda + accesos rápidos
/objetos/               Todos los objetos con loot (filtros: mod, rareza)
/objetos/<id>/          Ficha de objeto
/estructuras/           Estructuras agrupadas por dimensión y mod
/estructuras/<id>/      Ficha de estructura
/biomas/                Biomas agrupados por dimensión
/biomas/<id>/           Ficha de bioma
/fuentes/               Otras fuentes (mobs, pesca, arqueología, gameplay)
/mods/<id>/             Resumen de lo que aporta cada mod
/tablas/<id>/           Vista técnica de una tabla de loot (para creadores)
/acerca/                Modpack analizado, versión, fecha y diagnósticos
```

## Fichas principales

### Ficha de objeto — *“¿Dónde lo consigo?”*

1. Cabecera: icono, nombre, ID técnico, mod de origen.
2. **Mejores fuentes**: top 3 por probabilidad por cofre.
3. Tabla completa de fuentes con columnas: fuente · tipo · probabilidad ·
   cantidad · confianza. Ordenable y filtrable.
4. **Por bioma**: biomas donde puede aparecer agrupando sus estructuras.

### Ficha de estructura — *“¿Qué hay dentro?”*

1. Cabecera: nombre, mod, dimensión, frecuencia de generación si se conoce.
2. **Dónde aparece**: lista de biomas (chips enlazables).
3. **Loot**: una sección por tabla de loot (p. ej. “Cofre principal”,
   “Cofre de la biblioteca”) con sus objetos y probabilidades.
4. Indicador de confianza de la asociación estructura↔loot.

### Ficha de bioma — *“¿Qué puedo saquear aquí?”*

1. Cabecera: nombre, dimensión, tags.
2. Estructuras con loot que pueden generarse en él, agrupadas por mod.
3. Resumen de los objetos más destacados disponibles en el bioma.

## Búsqueda

- Índice JSON generado en la construcción del sitio (objetos, estructuras,
  biomas, mods).
- Búsqueda difusa (tolerante a errores) por nombre en el idioma elegido, en
  inglés y por ID.
- Resultados agrupados por tipo: *Objetos · Estructuras · Biomas · Mods*.
- Búsqueda inversa directa: escribir un objeto muestra al instante sus
  estructuras principales en el desplegable.

## Idioma

- Interfaz del sitio traducible (español e inglés al inicio).
- Nombres de objetos/estructuras/biomas tomados de los archivos `lang` de los
  mods en el idioma elegido, con *fallback* a inglés y luego al ID.

## Personalización (para creadores de modpacks)

- Título, logo y colores del sitio vía configuración.
- Ocultar mods, estructuras u objetos concretos (p. ej. contenido de debug).
- Texto libre en la portada.
