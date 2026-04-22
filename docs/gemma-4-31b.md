# Gemma 4 31B

Configuracion y resultados operativos para `google/gemma-4-31B-it` en `4x RTX 3090`.

Archivo:

- [env/prod/gemma-4-31b.yaml](../env/prod/gemma-4-31b.yaml)

Modelo oficial:

- <https://hf.co/google/gemma-4-31B-it>

## Perfil elegido

- hardware objetivo: `4x RTX 3090`
- despliegue: `4 GPU`
- modo del orquestador: `single_pass`
- backend HTTP: `vllm_chat_completions`
- release usada en laboratorio: `underpass-llm-gemma-4-31b`
- imagen actual del orquestador: `registry.underpassai.com/underpass-vllms:20260422-openai-responses-compat`

## Decisiones de configuracion

- `structured.model: google/gemma-4-31B-it`
- `tensorParallelSize: 4`
- `maxModelLen: 32768`
- `structured.extraArgs` incluye:
  - `--reasoning-parser=gemma4`
  - `--limit-mm-per-prompt={"image":0,"audio":0}`
- `orchestrator.modelType: gemma4`
- `orchestrator.pass2.provider: vllm_chat_completions`
- `orchestrator.config.singlePassPromptVersion: 2026-04-21.1`
- `orchestrator.config.pass2MaxTokens: 4096`
- `orchestrator.config.pass2Timeout: 90s`

## Validacion real

Estado observado en cluster:

- `underpass-llm-gemma-4-31b-orchestrator` -> `1/1`
- `underpass-llm-gemma-4-31b-structured` -> `1/1`

Smoke test minimo:

- `GET /readyz` correcto
- request minima `{"value":"hello"}` correcta

Shape de respuesta validada:

- `execution_mode = single_pass`
- `metadata.single_pass` presente
- `metadata.pass1` y `metadata.pass2` ausentes

Eso confirma que el refactor del contrato `single_pass` ya esta activo en el despliegue real.

## Bateria SWE

Run principal:

- `20260422T112354Z`

Resultado de disponibilidad:

- `8/8` casos con `200`
- `0` errores HTTP
- `0` errores de transporte

Veredicto semantico:

- `4 pass`
- `4 partial`
- `0 fail`

Lectura:

- no rompio el contrato JSON
- no mostro una mejora clara frente a `Gemma 4 26B-A4B-it`
- salio unas `6.5x` mas lento con una longitud media de salida casi igual

Metricas comparativas clave:

| Modelo | Latencia media | Completion tokens medios |
| --- | --- | --- |
| `Gemma 4 31B-it` | `10743.75 ms` | `175.125` |
| `Gemma 4 26B-A4B-it` | `1643.875 ms` | `179.25` |

## Donde se comporto bien

- disponibilidad `8/8`
- contrato JSON limpio
- `bug-regression-triage`
- `incident-prod-checkout`
- `refactor-plan-selector`
- `java-log-connection-timeout`

## Donde no compenso

- `feature-request-acceptance`: peor que `26B-A4B` en detalle util
- `pr-review-risk`: pierde warnings operativos que `26B-A4B` si conserva
- casos ambiguos: no mejora la disciplina de incertidumbre

## Lectura operativa

`Gemma 4 31B-it` queda operativo y usable, pero con la evidencia de este repo no justifica sustituir a `Gemma 4 26B-A4B-it` como `default rapido`.

Decision sugerida:

- `default rapido`: no
- `comparador de laboratorio`: si
- `misma familia con mas coste`: si

Referencia recomendada dentro de Gemma:

- [docs/gemma-4-26b-a4b.md](gemma-4-26b-a4b.md)
