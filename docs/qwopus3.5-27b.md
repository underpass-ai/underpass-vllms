# Qwopus 3.5 27B

Configuracion y notas operativas para desplegar `Jackrong/Qwopus3.5-27B-v3` en `4x RTX 3090`.

Archivo:

- [env/prod/qwen3.5-27b.yaml](../env/prod/qwen3.5-27b.yaml)

## Perfil elegido

- hardware objetivo: `4x RTX 3090`
- despliegue: `4 GPU`
- modo del orquestador: `two_pass`
- backend HTTP: `vllm_chat_completions`
- release usada en laboratorio: `underpass-llm-qwen3-5-27b`

## Correcciones necesarias para meter el modelo real

La primera version de este values file no apuntaba a `Qwopus`, sino a:

- `Qwen/Qwen3.5-27B-GPTQ-Int4`

Se corrigio para usar:

- `Jackrong/Qwopus3.5-27B-v3`

Y ademas:

- se elimino la flag vieja de cuantizacion
- se bajo `maxModelLen` de `32768` a `16384`
- se bajo `pass1MaxTokens` a `8192`
- se bajo `pass1ThinkingTokenBudget` a `4096`
- se actualizo la imagen del orquestador al tag actual

## Gotchas encontrados

### 1. `thinking_token_budget` sin `reasoning-config` falla

Sin `--reasoning-config`, `vLLM` devolvia:

```text
thinking_token_budget is set but reasoning_config is not configured
```

Por eso fue necesario dejar:

```yaml
reasoning:
  extraArgs:
    - --language-model-only
    - --reasoning-parser=qwen3
    - '--reasoning-config={"reasoning_start_str":"<think>","reasoning_end_str":"I have to give the solution based on the reasoning directly now.</think>"}'
```

### 2. `pass1MaxTokens` no puede igualar `maxModelLen`

Con `maxModelLen: 16384` y `pass1MaxTokens: 16384`, el backend devolvia `400` porque no quedaba margen para el prompt.

La configuracion estable de laboratorio quedo en:

```yaml
orchestrator:
  config:
    pass1MaxTokens: 8192
    pass1ThinkingTokenBudget: 4096
```

## Validacion real

Smoke test minimo correcto:

- request: `Return hello in the value field`
- salida:

```json
{
  "intermediate_representation": "value: hello",
  "output": {
    "value": "hello"
  }
}
```

Metricas de esa prueba:

- `execution_mode = two_pass`
- `pass1.latency_ms = 33312`
- `pass2.latency_ms = 11213`
- `pass1.reasoning_present = true`
- `pass2.reasoning_present = true`

## Bateria SWE

Run:

- `20260421T223752Z`

Nota:

- los artefactos crudos se generaron en `tmp/swe-matrix/20260421T223752Z/`, pero `tmp/` no forma parte del repo versionado

Resultado:

- `8/8` casos con `502`
- `0` errores de transporte

Motivo principal:

- timeouts del orquestador contra `vLLM`
- mezcla de:
  - `pass1_transport_failure`
  - `pass2_transport_failure`
  - siempre por `context deadline exceeded`

Ejemplos vistos en logs del orquestador:

- `pass1` en `56s`
- `pass1` en `70s`
- `pass1` en `53s`

## Lectura operativa

`Qwopus` ya esta correctamente desplegado y responde a requests simples, pero con la configuracion actual no aguanta la bateria SWE completa dentro de los timeouts del orquestador.

Conclusion practica:

- `instalacion`: correcta
- `smoke test`: correcto
- `benchmark SWE`: no apto todavia
- `siguiente paso`: mas tuning de timeouts o de presupuestos de thinking antes de volver a compararlo con `Gemma` y `gpt-oss-120b`

Estado recomendado en este repo:

- conservar el values file como referencia de tuning
- no usarlo como perfil operativo por defecto
