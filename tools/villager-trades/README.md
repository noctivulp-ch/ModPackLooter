# Verificar los tradeos vanilla contra el jar del juego

Los tradeos de aldeanos están en código (`VillagerTrades`), no en datos. La app
los trae como conocimiento incluido (`app/internal/discovery/vanilla/trades.go`)
y se verificaron así con los jars de cliente 1.20.1 y 1.21:

1. Descomprimir las clases del jar: `unzip client.jar '*.class' -d clases`.
2. Encontrar las clases ofuscadas (los nombres cambian en cada versión):
   - `Items`: la que contiene la cadena `netherite_upgrade_smithing_template`.
   - `Blocks`: la clase más usada en `getstatic` dentro de `Items`.
   - `VillagerProfession`: el `Record` que contiene `cartographer`; sus campos
     siguen el orden none, armorer, butcher, … (ver `javap -c` de su `<clinit>`).
   - `VillagerTrades`: la clase que usa `Items` y `VillagerProfession` y tiene
     una docena de clases internas `X$a`, `X$b`…; sus constructores dicen qué
     tipo de tradeo es cada una (`javap -p 'X$a.class'`).
3. Mapas de campos → ids: en el `<clinit>` de `Items` y `Blocks`, cada
   `putstatic` va precedido de su cadena (`ldc "wheat"`) o del bloque del que
   sale el ítem. Guardarlos como JSON (`itemmap.json`, `blockmap.json`).
4. `javap -c -p -constants X.class > trades.txt` y ejecutar
   `interp.py trades.txt itemmap.json blockmap.json profmap.json Items Blocks VillagerProfession X 'private static void a(java.util.HashMap);'`
   para la lista por profesión (el comerciante errante sale del `static {}`).

Diferencias 1.20.1 → 1.21: el cartógrafo oficial suma el mapa de cámaras de
prueba (12 esmeraldas) y `scute` pasa a `turtle_scute`. Las listas del
«rebalanceo de tradeos» de 1.21 son experimentales y no se incluyen.
