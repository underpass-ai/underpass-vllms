# Gemma 4 26B A4B

Configuracion y notas operativas para desplegar `google/gemma-4-26B-A4B-it` en `4x RTX 3090`.

Archivo:

- [env/prod/gemma-4-26b-a4b.yaml](../env/prod/gemma-4-26b-a4b.yaml)

## Perfil elegido

- hardware objetivo: `4x RTX 3090`
- despliegue: `4 GPU`
- modo del orquestador: `single_pass`
- backend HTTP: `vllm_chat_completions`
- release usada en laboratorio: `underpass-llm-gemma-4-26b-a4b`

## Posicion dentro de la familia

Dentro de `Gemma 4`, esta es la variante que mejor encajo en este repo para `96 GB`.

Referencia de familia:

- [docs/gemma-4-family.md](gemma-4-family.md)

Alternativa mas grande ya preparada en el repo:

- [docs/gemma-4-31b.md](gemma-4-31b.md)

## Decisiones de configuracion

- `structured.model: google/gemma-4-26B-A4B-it`
- `tensorParallelSize: 4`
- `maxModelLen: 16384`
- `orchestrator.modelType: gemma4`
- `pass2.provider: vllm_chat_completions`
- `pass2MaxTokens: 4096`
- `pass2Timeout: 90s`

## Gotchas encontrados

### 1. `gemma4` necesitaba soporte real de `single_pass` en el chart

El binario ya soportaba `MODEL_TYPE=gemma4`, pero el chart solo inyectaba:

- `SINGLE_PASS_PROMPT_VERSION`
- `SINGLE_PASS_SYSTEM_PROMPT`
- `SINGLE_PASS_USER_PROMPT_TEMPLATE`

para `gpt_oss`.

Se corrigio en:

- [charts/vllm/templates/orchestrator-deployment.yaml](../charts/vllm/templates/orchestrator-deployment.yaml)

## 2. `--limit-mm-per-prompt` tenia que ir en JSON

Con `vLLM v0.19.1`, esta forma fallaba:

```text
--limit-mm-per-prompt=image=0,audio=0
```

La forma valida fue:

```text
--limit-mm-per-prompt={"image":0,"audio":0}
```

Se dejo asi en:

- [env/prod/gemma-4-26b-a4b.yaml](../env/prod/gemma-4-26b-a4b.yaml)
- [env/prod/gemma-4-31b.yaml](../env/prod/gemma-4-31b.yaml)

## Validacion real

Smoke tests correctos:

- request minima: devuelve `{"value":"hello"}`
- extraccion de factura: devuelve JSON correcto con campos estructurados y `warnings=[]`

La release quedo sana en cluster:

- `underpass-llm-gemma-4-26b-a4b-orchestrator` -> `1/1`
- `underpass-llm-gemma-4-26b-a4b-structured` -> `1/1`

## Bateria SWE

Run principal:

- `20260421T221011Z`

Nota:

- los artefactos crudos de esta bateria se generaron en `tmp/swe-matrix/20260421T221011Z/`, pero `tmp/` no forma parte del repo versionado

Resultado:

- `8/8` casos con `200`
- `0` errores HTTP
- `0` errores de transporte
- tiempo total medido de la tanda repetida: `15s`

Veredicto practico de esa tanda:

- `pass`: `6`
- `partial`: `2`
- `fail`: `0`

## Lectura operativa

`Gemma 4 26B A4B` no fue el mejor en calidad pura, pero si el mejor equilibrio `speed/quality` de los modelos probados en este repo hasta ahora.

Decision sugerida:

- `default rapido`: si
- `default premium`: no
- `mejor Gemma practica en este repo`: si
- `casos ambiguos/conflictivos`: mejor revisarlos con mas cautela o compararlos contra `gpt-oss-120b`

Comparativa directa dentro de la familia:

- `Gemma 4 31B-it` fue mas lento y no mejoro de forma clara la calidad
- por eso `26B-A4B-it` sigue siendo el punto dulce practico del repo
