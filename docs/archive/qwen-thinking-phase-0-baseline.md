# Fase 0 Baseline: Qwen Thinking Integration

Fecha de ejecucion: 2026-04-21

Documento relacionado:

- [docs/qwen-thinking-execution-checklist.md](/home/tirso/ai/developents/underpass-vllms/docs/qwen-thinking-execution-checklist.md)

## Objetivo

Congelar una linea base reproducible del estado actual de la integracion antes de tocarla.

Esta Fase 0 tenia cinco objetivos:

- capturar la revision Helm desplegada del orquestador,
- capturar imagenes y tags desplegados,
- guardar un ejemplo reproducible del fallo `pass1_empty_response`,
- guardar el mismo request ejecutado desde el workspace local contra los mismos upstreams,
- dejar escrita la conclusion de drift.

## Resumen Ejecutivo

Resultado de Fase 0:

- el orquestador desplegado falla con `pass1_empty_response`,
- el endpoint `reasoning` de vLLM devuelve `reasoning` poblado, `content=null` y `finish_reason=length`,
- el orquestador local del workspace, usando exactamente los mismos upstreams `reasoning` y `structured`, completa el flujo correctamente,
- por tanto, el problema observado no apunta al modelo en si, sino a drift o desalineacion entre el artefacto desplegado y el comportamiento esperado del workspace, ademas de un tuning fragil en `Pass 1`.

## Baseline del despliegue

### Helm history del orquestador

Comando:

```bash
helm history underpass-llm-orchestrator -n underpass-runtime
```

Salida relevante:

```text
REVISION  UPDATED                  STATUS     CHART               APP VERSION  DESCRIPTION
1         Mon Apr 20 12:12:13 2026 superseded underpass-llm-0.2.0 0.1.0        Install complete
2         Mon Apr 20 12:50:43 2026 superseded underpass-llm-0.2.0 0.1.0        Upgrade complete
3         Mon Apr 20 14:35:53 2026 superseded underpass-llm-0.2.0 0.1.0        Upgrade complete
4         Mon Apr 20 14:40:56 2026 deployed   underpass-llm-0.2.0 0.1.0        Upgrade complete
```

### Imagenes y parametros efectivos

Comando base:

```bash
kubectl get deploy -n underpass-runtime \
  underpass-llm-orchestrator-orchestrator \
  underpass-llm-reasoning-reasoning \
  underpass-llm-structured-structured \
  -o json
```

Hallazgos relevantes:

- `orchestrator`
  - image: `ghcr.io/tgarciai/underpass-vllms:20260420-promptfix`
  - `PASS1_BASE_URL=http://underpass-llm-reasoning-reasoning.underpass-runtime.svc.cluster.local:8000/v1`
  - `PASS2_BASE_URL=http://underpass-llm-structured-structured.underpass-runtime.svc.cluster.local:8000/v1`
  - `PASS1_MODEL=palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4`
  - `PASS2_MODEL=palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4`
  - `PASS1_MAX_TOKENS=1500`
  - `PASS2_MAX_TOKENS=800`
- `reasoning`
  - image: `docker.io/vllm/vllm-openai:v0.19.1-x86_64-cu130`
  - args incluyen `--reasoning-parser=qwen3`
  - modelo: `palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4`
- `structured`
  - image: `docker.io/vllm/vllm-openai:v0.19.1-x86_64-cu130`
  - args incluyen `--default-chat-template-kwargs={"enable_thinking": false}`
  - modelo: `palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4`

### Values efectivos del release del orquestador

Comando:

```bash
helm get values underpass-llm-orchestrator -n underpass-runtime
```

Hallazgos relevantes:

- `orchestrator.image.tag: 20260420-promptfix`
- `orchestrator.config.pass1MaxTokens: 1500`
- `orchestrator.config.pass2MaxTokens: 800`
- `orchestrator.config.pass1Temperature: "0.2"`
- `orchestrator.config.pass2Temperature: "0"`

## Evidencia Dinamica

Caso usado para baseline:

- `bug-regression-triage`
- fuente: [testdata/swe-matrix/cases.json](/home/tirso/ai/developents/underpass-vllms/testdata/swe-matrix/cases.json)

### 1. Respuesta del orquestador desplegado

Setup:

- `kubectl port-forward -n underpass-runtime svc/underpass-llm-orchestrator-orchestrator 18080:8080`

Request:

```bash
PAYLOAD=$(jq -c '.cases[] | select(.id=="bug-regression-triage") | .payload' testdata/swe-matrix/cases.json)
curl -s http://127.0.0.1:18080/v1/two-pass/structured \
  -H 'content-type: application/json' \
  -d "$PAYLOAD"
```

Respuesta:

```json
{"error":{"code":"pass1_empty_response","message":"Pass 1 returned an empty intermediate representation","retryable":false}}
```

### 2. Respuesta raw del endpoint reasoning

Setup:

- `kubectl port-forward -n underpass-runtime svc/underpass-llm-reasoning-reasoning 18081:8000`

Request:

```bash
curl -s http://127.0.0.1:18081/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{...same prompt and max_tokens=1500...}'
```

Hallazgos del payload de respuesta:

- `choices[0].message.reasoning` estaba poblado
- `choices[0].message.content` era `null`
- `choices[0].finish_reason` era `"length"`
- `usage.completion_tokens` era `1500`

Extracto relevante:

```json
{
  "choices": [
    {
      "message": {
        "content": null,
        "reasoning": "Here's a thinking process: ..."
      },
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 225,
    "completion_tokens": 1500
  }
}
```

Lectura operativa:

- el modelo si respondio,
- gasto todo el presupuesto de salida pensando,
- no emitio respuesta final en `content`.

### 3. Mismo request desde el orquestador local del workspace

Setup:

- `kubectl port-forward -n underpass-runtime svc/underpass-llm-reasoning-reasoning 18081:8000`
- `kubectl port-forward -n underpass-runtime svc/underpass-llm-structured-structured 18082:8000`
- orquestador local:

```bash
SERVER_ADDR=127.0.0.1:18090 \
PASS1_BASE_URL=http://127.0.0.1:18081/v1 \
PASS1_MODEL=palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4 \
PASS2_BASE_URL=http://127.0.0.1:18082/v1 \
PASS2_MODEL=palmfuture/Qwen3.6-35B-A3B-GPTQ-Int4 \
PASS1_API_KEY=EMPTY \
PASS2_API_KEY=EMPTY \
PASS1_MAX_TOKENS=1500 \
PASS2_MAX_TOKENS=800 \
PASS1_TIMEOUT=45s \
PASS2_TIMEOUT=45s \
go run ./cmd/two-pass-server
```

Request:

```bash
PAYLOAD=$(jq -c '.cases[] | select(.id=="bug-regression-triage") | .payload' testdata/swe-matrix/cases.json)
curl -s http://127.0.0.1:18090/v1/two-pass/structured \
  -H 'content-type: application/json' \
  -d "$PAYLOAD"
```

Respuesta relevante:

```json
{
  "request_id": "c78331e5401e40bf",
  "intermediate_representation": "Here's a thinking process: ...",
  "output": {
    "kind": "bug",
    "summary": "Go API crashes on login when tenant_id is missing",
    "language": "Go",
    "environment": "production",
    "severity": "high",
    "files": [
      "internal/auth/session.go",
      "cmd/api/main.go"
    ],
    "warnings": [
      "Root cause is suspected, not confirmed"
    ]
  },
  "metadata": {
    "pass1": {
      "attempts": 1,
      "latency_ms": 13958,
      "completion_tokens": 1500
    },
    "pass2": {
      "attempts": 1,
      "latency_ms": 1808
    }
  }
}
```

Lectura operativa:

- el workspace local consigue completar el flujo,
- lo hace usando los mismos upstreams `reasoning` y `structured`,
- y con el mismo `PASS1_MAX_TOKENS=1500`.

## Estado del workspace

Comando:

```bash
git status --short
```

Hallazgo:

- el worktree contiene una gran cantidad de codigo sin trackear en Git, incluyendo `cmd/`, `internal/`, `docs/`, `env/` y `scripts/`

Extracto:

```text
?? Dockerfile
?? Makefile
?? README.md
?? cmd/
?? deploy/
?? docs/
?? env/
?? go.mod
?? go.sum
?? internal/
?? mk/
?? scripts/
?? testdata/
```

Lectura operativa:

- hoy no existe trazabilidad limpia entre `HEAD`, el workspace real y la imagen desplegada
- esto agrava el riesgo de drift y justifica la Fase 6 del plan

## Conclusion de Drift

Conclusion de Fase 0:

- el problema observado no queda explicado por el modelo ni por los endpoints `reasoning/structured` por separado
- el mismo request, contra los mismos upstreams y con los mismos limites de tokens, falla en el orquestador desplegado y funciona en el orquestador local del workspace
- por tanto, hay evidencia fuerte de drift o desalineacion funcional entre:
  - el artefacto desplegado `ghcr.io/tgarciai/underpass-vllms:20260420-promptfix`
  - y el comportamiento del codigo que vive hoy en este workspace

Ademas:

- incluso en el caso exitoso, `Pass 1` sigue siendo fragil
- `completion_tokens=1500` exactos y `intermediate_representation` inflado indican que el tuning de `thinking` sigue siendo pobre para operacion normal

Conclusion operativa final:

- Fase 0 queda cerrada
- el siguiente trabajo correcto es Fase 1: fijar el contrato interno `content/reasoning/finish_reason`
- en paralelo, hay que preparar Fase 3: tuning explicito del presupuesto de `Pass 1`
