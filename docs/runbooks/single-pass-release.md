# Runbook: Single-Pass Release

Usa este runbook para releases como:

- `underpass-llm-gemma-4-26b-a4b`
- `underpass-llm-gemma-4-31b`
- `underpass-llm-gpt-oss-120b`

## 1. Validar el values file

```bash
helm template underpass-llm-gemma-4-31b charts/vllm \
  -n underpass-runtime \
  -f env/prod/gemma-4-31b.yaml
```

Checklist:

- `reasoning.enabled=false`
- `structured.enabled=true`
- `orchestrator.enabled=true`
- `orchestrator.modelType` correcto
- imagen del orquestador correcta

## 2. Desplegar

```bash
helm upgrade --install underpass-llm-gemma-4-31b charts/vllm \
  -n underpass-runtime \
  -f env/prod/gemma-4-31b.yaml
```

## 3. Esperar rollout

```bash
kubectl rollout status deployment/underpass-llm-gemma-4-31b-orchestrator \
  -n underpass-runtime --timeout=180s

kubectl rollout status deployment/underpass-llm-gemma-4-31b-structured \
  -n underpass-runtime --timeout=180s
```

## 4. Ver estado rapido

```bash
kubectl get pods -n underpass-runtime \
  -l app.kubernetes.io/instance=underpass-llm-gemma-4-31b
```

Debe quedar algo asi:

```text
underpass-llm-gemma-4-31b-orchestrator-...   1/1 Running
underpass-llm-gemma-4-31b-structured-...     1/1 Running
```

## 5. Smoke test minimo

### `readyz`

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  curl -s http://underpass-llm-gemma-4-31b-orchestrator:8080/readyz
```

Esperado:

```json
{"status":"ok"}
```

### `chat.completions`

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  curl -s http://underpass-llm-gemma-4-31b-orchestrator:8080/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{
    "model":"google/gemma-4-31B-it",
    "messages":[{"role":"user","content":"Return hello in the value field"}],
    "response_format":{
      "type":"json_schema",
      "json_schema":{
        "name":"hello_schema",
        "schema":{
          "type":"object",
          "properties":{"value":{"type":"string"}},
          "required":["value"],
          "additionalProperties":false
        }
      }
    }
  }'
```

### `responses`

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  curl -s http://underpass-llm-gemma-4-31b-orchestrator:8080/v1/responses \
  -H 'content-type: application/json' \
  -d '{
    "model":"google/gemma-4-31B-it",
    "input":"Return hello in the value field",
    "text":{
      "format":{
        "type":"json_schema",
        "name":"hello_schema",
        "schema":{
          "type":"object",
          "properties":{"value":{"type":"string"}},
          "required":["value"],
          "additionalProperties":false
        }
      }
    }
  }'
```

## 6. Smoke test de streaming

### `chat.completions`

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  curl -sN http://underpass-llm-gemma-4-31b-orchestrator:8080/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{
    "model":"google/gemma-4-31B-it",
    "messages":[{"role":"user","content":"Return hello in the value field"}],
    "response_format":{
      "type":"json_schema",
      "json_schema":{
        "name":"hello_schema",
        "schema":{
          "type":"object",
          "properties":{"value":{"type":"string"}},
          "required":["value"],
          "additionalProperties":false
        }
      }
    },
    "stream":true
  }'
```

Esperado:

- varios `data: {..."object":"chat.completion.chunk"...}`
- cierre con `data: [DONE]`

### `responses`

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  curl -sN http://underpass-llm-gemma-4-31b-orchestrator:8080/v1/responses \
  -H 'content-type: application/json' \
  -d '{
    "model":"google/gemma-4-31B-it",
    "input":"Return hello in the value field",
    "text":{
      "format":{
        "type":"json_schema",
        "name":"hello_schema",
        "schema":{
          "type":"object",
          "properties":{"value":{"type":"string"}},
          "required":["value"],
          "additionalProperties":false
        }
      }
    },
    "stream":true
  }'
```

Esperado:

- `response.created`
- uno o varios `response.output_text.delta`
- `response.output_text.done`
- `response.completed`

## 7. Seguir logs del orquestador

```bash
kubectl logs -f -n underpass-runtime \
  deployment/underpass-llm-gemma-4-31b-orchestrator --tail=20
```

Esperado por request:

```text
request_id=... adapter=single_pass model=... latency_ms=... finish_reason=stop content_present=true reasoning_present=false
```

## 8. Rollback rapido

Ver historial:

```bash
helm history underpass-llm-gemma-4-31b -n underpass-runtime
```

Volver a la revision anterior:

```bash
helm rollback underpass-llm-gemma-4-31b <revision> -n underpass-runtime
```

## 9. Criterio de exito

Considera la release sana si cumple todo esto:

- pods `1/1 Running`
- `readyz` correcto
- `chat.completions` correcto
- `responses` correcto
- streaming correcto en ambos endpoints
- logs del orquestador cierran con `finish_reason=stop`
