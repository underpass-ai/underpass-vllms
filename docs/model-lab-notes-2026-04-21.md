# Model Lab Notes 2026-04-21 to 2026-04-22

Resumen consolidado de los despliegues y pruebas de modelos hechos en este repo durante la sesion de laboratorio del `2026-04-21` y su continuacion del `2026-04-22`.

## Regla sobre artefactos

Los artefactos crudos de las baterias se generan bajo `tmp/swe-matrix/<run_id>/`, pero `tmp/` esta ignorado por Git. Por eso este documento conserva:

- `run_id`
- resumen numerico
- veredicto operativo

Si el directorio local existe, puedes volver a inspeccionarlo manualmente. No se enlaza desde la documentacion como si formara parte del repo versionado.

## Modelos probados

### Qwen baseline

Run principal:

- `20260421T182212Z`

Lectura:

- ruta `two_pass`
- stack estable
- rendimiento flojo en casos ambiguos, `PR review` y `log analysis`

Resultado:

- `2 pass`
- `2 partial`
- `4 fail`

Estado operativo:

- valido como referencia de `two_pass`
- no recomendado como `default`

### GPT-OSS 120B

Documentacion de despliegue:

- [docs/gpt-oss-120b.md](gpt-oss-120b.md)

Run principal:

- `20260421T213623Z`

Lectura:

- ruta `single_pass`
- mejor calidad global de los modelos comparados en esta tanda
- mas caro y mas lento que `Gemma`
- sigue flojo en `warnings` y en casos con conflicto de evidencia

Resultado:

- `7 pass`
- `1 partial`
- `0 fail`

Estado operativo:

- `default premium`

### Gemma 4 26B A4B

Documentacion de despliegue:

- [docs/gemma-4-family.md](gemma-4-family.md)
- [docs/gemma-4-26b-a4b.md](gemma-4-26b-a4b.md)

Run principal:

- `20260421T221011Z`

Lectura:

- ruta `single_pass`
- mejor relacion `speed/quality`
- muy rapido porque no expone reasoning, no hace segundo pase y no necesitaba retries
- fue el `Gemma 4` con mejor sentido practico en esta sesion

Resultado:

- `6 pass`
- `2 partial`
- `0 fail`

Estado operativo:

- `default rapido`

### Qwopus 3.5 27B

Documentacion de despliegue:

- [docs/qwopus3.5-27b.md](qwopus3.5-27b.md)

Run principal:

- `20260421T223752Z`

Lectura:

- ruta `two_pass`
- despliegue correcto
- smoke test correcto
- bateria SWE inutilizable con la configuracion actual por timeouts del orquestador

Resultado:

- `0 pass`
- `0 partial`
- `8` errores `502`

Estado operativo:

- no listo
- referencia de tuning, no de rendimiento

### Gemma 4 31B

Documentacion de despliegue:

- [docs/gemma-4-31b.md](gemma-4-31b.md)

Run principal:

- `20260422T112354Z`

Lectura:

- ruta `single_pass`
- `8/8` en disponibilidad y sin roturas del contrato JSON
- no mostro una mejora clara frente a `Gemma 4 26B-A4B`
- fue unas `6.5x` mas lento con una longitud de salida casi igual

Resultado:

- `4 pass`
- `4 partial`
- `0 fail`

Estado operativo:

- usable
- no justifica sustituir a `Gemma 4 26B-A4B` como `default rapido`

## Tabla rapida

| Modelo | Ruta | Calidad | Velocidad | Estado operativo |
| --- | --- | --- | --- | --- |
| `Qwen baseline` | `two_pass` | baja en esta bateria | baja | estable pero superado |
| `GPT-OSS 120B` | `single_pass` | mejor calidad pura | media | listo |
| `Gemma 4 26B A4B` | `single_pass` | buena | muy alta | listo |
| `Gemma 4 31B` | `single_pass` | buena, no mejor que `26B-A4B` | media-baja | listo |
| `Qwopus 3.5 27B` | `two_pass` | sin medir bien por timeout | baja con config actual | no listo |

## Decision sugerida

Si hay que escoger ya:

- `default rapido`: `Gemma 4 26B A4B`
- `default premium`: `GPT-OSS 120B`
- `Gemma 4 31B`: mantenerlo como comparador o variante de laboratorio
- `Qwen baseline`: no recomendado como `default` con esta evidencia
- `Qwopus`: no volver a compararlo hasta ajustar timeouts o budgets

## Estado actual del repo

- el contrato HTTP ya distingue bien `two_pass` y `single_pass`
- el perfil `gemma4` ya usa metadata `single_pass` limpia
- la fachada OpenAI ya soporta streaming real en `single_pass`
- `chat.completions` emite `chat.completion.chunk`
- `responses` emite `response.created`, `response.output_text.delta`, `response.output_text.done` y `response.completed`
- `tmp/` no se versiona
- la release activa de laboratorio documentada es `Gemma 4 31B`
