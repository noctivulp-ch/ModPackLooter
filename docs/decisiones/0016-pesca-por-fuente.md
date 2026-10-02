# ADR-0016: Pesca agrupada por sistema de pesca

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Starcatcher, Tide 2 y la caña vanilla son sistemas independientes, cada uno con
su caña y con reglas de probabilidad distintas (suma de probabilidades con
cebos; pesos anidados con temperatura; tabla de loot). El usuario pidió capas
pesca › fuente › bioma › condiciones › loot, e incluir la vanilla.

## Decisión

Un modelo común (`internal/fishing`) con entradas, reglas de bioma y grupos de
capturas por bioma. Cada mod tiene su descubridor, que aplica **sus propias
reglas** (leídas de su código fuente) y adjunta un `fishing.Source`. Solo se
modelan como «dónde» el bioma, la dimensión y el líquido; el resto de
condiciones se muestran como texto.

## Alternativas consideradas

- **Una tabla única de pesca mezclando mods** — engañosa: cada caña usa un
  sistema distinto.
- **Simular todas las condiciones (altura, hora, clima)** — demasiadas
  combinaciones para leer; se prefiere mostrarlas junto a cada captura.

## Consecuencias

- Añadir otro mod de pesca = otro descubridor que produce un `fishing.Source`.
- Las probabilidades son «en ese bioma y líquido»; las condiciones visibles
  explican cuándo cambian.
