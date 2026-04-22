# Model Lab Notes 2026-04-21

Resumen de los despliegues y pruebas de modelos hechos en este repo durante la sesion de laboratorio del `2026-04-21`.

## Modelos probados

### Qwen baseline

Artefactos:

- [tmp/swe-matrix/20260421T182212Z/REPORT.md](../tmp/swe-matrix/20260421T182212Z/REPORT.md)
- [tmp/swe-matrix/20260421T182212Z/JUDGMENT.md](../tmp/swe-matrix/20260421T182212Z/JUDGMENT.md)

Lectura:

- ruta `two_pass`
- stack estable
- rendimiento flojo en casos ambiguos, PR review y log analysis

Resultado:

- `2 pass`
- `2 partial`
- `4 fail`

### GPT-OSS 120B

Documentacion de despliegue:

- [docs/gpt-oss-120b.md](gpt-oss-120b.md)

Artefactos:

- [tmp/swe-matrix/20260421T213623Z/REPORT.md](../tmp/swe-matrix/20260421T213623Z/REPORT.md)
- [tmp/swe-matrix/20260421T213623Z/JUDGMENT.md](../tmp/swe-matrix/20260421T213623Z/JUDGMENT.md)

Lectura:

- ruta `single_pass`
- mejor calidad global de los modelos comparados
- mas caro y mas lento que `Gemma`

Resultado:

- `7 pass`
- `1 partial`
- `0 fail`

### Gemma 4 26B A4B

Documentacion de despliegue:

- [docs/gemma-4-family.md](gemma-4-family.md)
- [docs/gemma-4-26b-a4b.md](gemma-4-26b-a4b.md)
- [docs/gemma-4-31b.md](gemma-4-31b.md)

Artefactos:

- [tmp/swe-matrix/20260421T221011Z/REPORT.md](../tmp/swe-matrix/20260421T221011Z/REPORT.md)
- [tmp/swe-matrix/20260421T221011Z/JUDGMENT.md](../tmp/swe-matrix/20260421T221011Z/JUDGMENT.md)

Lectura:

- ruta `single_pass`
- mejor relacion `speed/quality`
- muy rapido porque no expone reasoning ni hace segundo pase
- fue el `Gemma 4` que mejor sentido practico tuvo en esta sesion

Resultado:

- `6 pass`
- `2 partial`
- `0 fail`

### Qwopus 3.5 27B

Documentacion de despliegue:

- [docs/qwopus3.5-27b.md](qwopus3.5-27b.md)

Artefactos:

- [tmp/swe-matrix/20260421T223752Z/REPORT.md](../tmp/swe-matrix/20260421T223752Z/REPORT.md)

Lectura:

- ruta `two_pass`
- despliegue correcto
- smoke test correcto
- bateria SWE inutilizable con la configuracion actual por timeouts del orquestador

Resultado:

- `0 pass`
- `0 partial`
- `8` errores `502`

## Tabla rapida

| Modelo | Ruta | Calidad | Velocidad | Estado operativo |
| --- | --- | --- | --- | --- |
| `Qwen baseline` | `two_pass` | baja en esta bateria | baja | estable pero superado |
| `GPT-OSS 120B` | `single_pass` | mejor | media | listo |
| `Gemma 4 26B A4B` | `single_pass` | buena | muy alta | listo |
| `Qwopus 3.5 27B` | `two_pass` | sin medir bien por timeout | baja con config actual | no listo |

## Decision sugerida

Si hay que escoger ya:

- `default rapido`: `Gemma 4 26B A4B`
- `siguiente Gemma a probar`: `Gemma 4 31B`
- `default premium`: `GPT-OSS 120B`
- `Qwen baseline`: no recomendado como default con esta evidencia
- `Qwopus`: no volver a compararlo hasta ajustar timeouts o budgets
