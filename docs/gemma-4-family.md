# Gemma 4 Family On 96GB

Resumen operativo de las variantes oficiales de `Gemma 4` que tienen sentido en una maquina con `96 GB` de VRAM agregada, por ejemplo `4x RTX 3090`.

## Variantes oficiales relevantes

### `google/gemma-4-31B-it`

- enlace: <https://hf.co/google/gemma-4-31B-it>
- tamano publicado: `32682.4M` parametros
- tipo: `image-text-to-text`
- papel practico: opcion Gemma 4 mas grande para usar como tier de mas calidad dentro de la misma familia

### `google/gemma-4-26B-A4B-it`

- enlace: <https://hf.co/google/gemma-4-26B-A4B-it>
- tamano publicado: `26544.1M` parametros
- tipo: `image-text-to-text`
- papel practico: mejor equilibrio `speed/quality`
- estado en este repo: probado y recomendado

### `google/gemma-4-E4B-it`

- enlace: <https://hf.co/google/gemma-4-E4B-it>
- tamano publicado: `7996.2M` parametros
- tipo: `any-to-any`
- papel practico: variante ligera con mucho margen de memoria

### `google/gemma-4-E2B-it`

- enlace: <https://hf.co/google/gemma-4-E2B-it>
- tamano publicado: `5123.2M` parametros
- tipo: `any-to-any`
- papel practico: variante muy ligera; no es la forma de exprimir `96 GB`

## Lo que de verdad importa en 96GB

Si el objetivo es aprovechar `4x3090`, las variantes que de verdad interesan son estas:

- `gemma-4-26B-A4B-it`
- `gemma-4-31B-it`

`E4B` y `E2B` caben de sobra, pero ya juegan en otro rango de capacidad.

## Recomendacion operativa

- `default rapido`: `google/gemma-4-26B-A4B-it`
- `siguiente escalon a probar`: `google/gemma-4-31B-it`
- `E4B` y `E2B`: solo si quieres una ruta mucho mas barata o mas ligera

## Estado en este repo

### `26B-A4B-it`

Documentacion y values:

- [docs/gemma-4-26b-a4b.md](gemma-4-26b-a4b.md)
- [env/prod/gemma-4-26b-a4b.yaml](../env/prod/gemma-4-26b-a4b.yaml)

Estado:

- desplegada
- smoke tests correctos
- bateria SWE ejecutada
- recomendada como `default rapido`

### `31B-it`

Documentacion y values:

- [docs/gemma-4-31b.md](gemma-4-31b.md)
- [env/prod/gemma-4-31b.yaml](../env/prod/gemma-4-31b.yaml)

Estado:

- configuracion preparada
- no documentada como bateria completa en este repo todavia
- candidata natural para comparar contra `26B-A4B-it`

## Versiones base

Tambien existen versiones base sin `-it`:

- <https://hf.co/google/gemma-4-31B>
- <https://hf.co/google/gemma-4-26B-A4B>
- <https://hf.co/google/gemma-4-E4B>
- <https://hf.co/google/gemma-4-E2B>

Para este repo tienen menos interes que las `-it`, porque aqui estamos evaluando comportamiento instruction-tuned para structured extraction.
