# Pendientes

Tareas para continuar en la próxima sesión.

## 1. Cotejar la guía de edificios de los jugadores

Comparar la guía «Deceased Craft V6 beta buildings guide» con los datos
generados para encontrar **faltantes**: se asume que el scraper no miente, así
que el error esperado es que falten datos, no que sobren.

- [ ] Pedir las hojas de distritos exportadas como CSV, una por pestaña (por
      ahora solo se tiene la hoja «readme»). También la hoja de armas TaCZ.
- [ ] Volver a pedir datos de prueba si hace falta: extracto del pack y jars de
      Minecraft 1.20.1 y 1.21 (límite de subida 30 MB; zip multiparte si pasa).
- [ ] Cruzar edificio por edificio: NPCs y loot de la guía contra el sitio.

Pistas de la readme:

- Cada hoja de distrito trae fotos, NPCs y loot por edificio.
- Columna «Floor guide»: los edificios salen con distinto número y variante de
  pisos.
- «Mira dentro de contenedores de dos bloques como iron locker o hielo, pueden
  tener un cofre dentro».
- Se ignoran la cesta de pan y la caja de chocolate; las pociones no se
  mencionan.

Faltantes probables a investigar:

- NPCs de CustomNPCs colocados en edificios de Lost Cities.
- Contenedores escondidos dentro de bloques multibloque.
- Partes y variantes de pisos.

## 2. Planear el mod companion de datos extra

Un mod (carpeta `mod/`) que registre desde el juego lo que los archivos no
dicen y lo exporte para que el generador lo lea:

- spawns por código o por eventos de mods;
- aldeanos creados por scripts;
- loot generado por código;
- condiciones dinámicas.

- [ ] Escribir el plan y una ADR nueva en `docs/decisiones/`, **sin código
      todavía**. Partir de `mod/README.md`, [ADR-0004](docs/decisiones/0004-app-externa-con-volcador-opcional.md)
      y [ADR-0009](docs/decisiones/0009-estructura-del-repositorio.md).
