# Runbook: Streaming Debug

Usa este runbook cuando:

- `stream=true` no devuelve nada
- parece que los chunks llegan todos juntos
- quieres distinguir si falla el orquestador o el cliente/proxy

## Regla base

Prueba primero desde dentro del cluster.

Si dentro del cluster hay deltas y fuera no, el problema normalmente no esta en el orquestador.

## 1. Confirmar que el backend activo es `single_pass`

Streaming publico solo esta soportado para `single_pass`.

Si el release usa `two_pass`, el comportamiento esperado es:

- `400`
- `invalid_request_error`
- mensaje tipo `stream=true is only supported for single_pass backends`

## 2. Seguir logs del orquestador en tiempo real

```bash
kubectl logs -f -n underpass-runtime \
  deployment/underpass-llm-gemma-4-31b-orchestrator --tail=20
```

Esperado al cerrar una request:

```text
request_id=... adapter=single_pass model=google/gemma-4-31B-it latency_ms=... finish_reason=stop content_present=true reasoning_present=false
```

## 3. Lanzar la prueba desde dentro del cluster

Usa el pod `structured` del mismo release.

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

## 4. Que debes esperar

### En `chat.completions`

- un primer chunk con `delta.role=assistant`
- varios chunks con `delta.content`
- un ultimo chunk con `finish_reason=stop`
- `data: [DONE]`

### En `responses`

- `response.created`
- varios `response.output_text.delta`
- `response.output_text.done`
- `response.completed`

## 5. Importante: los deltas pueden ser muy pequeños

Esto es normal.

Con `Gemma 4 31B` ya se vio en vivo este patron:

- deltas cada `~50-70 ms`
- trozos muy pequeños
- a veces un token
- a veces un subtoken
- a veces solo `{\"`, `value`, `\":`, `"hello"`

No esperes bloques grandes o frases completas.

## 6. Si parece que no hay deltas

Haz esta prueba con timestamps:

```bash
kubectl exec -n underpass-runtime <structured-pod> -- \
  python3 -u -c 'import json,time,urllib.request; payload={"model":"google/gemma-4-31B-it","input":"Return hello in the value field","text":{"format":{"type":"json_schema","name":"hello_schema","schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":False}}},"stream":True}; req=urllib.request.Request("http://underpass-llm-gemma-4-31b-orchestrator:8080/v1/responses", data=json.dumps(payload).encode(), headers={"content-type":"application/json"}); start=time.time(); resp=urllib.request.urlopen(req, timeout=180); idx=0\nfor raw in resp:\n line=raw.decode().rstrip("\\n")\n if not line:\n  continue\n idx += 1\n elapsed=time.time()-start\n print(f"{elapsed:7.3f}s | {idx:03d} | {line[:220]}", flush=True)\n'
```

Si ahi ves tiempos progresivos, el stream esta bien.

## 7. Si dentro del cluster funciona y fuera no

Sospecha de:

- cliente que no consume SSE de verdad
- proxy que bufferiza
- ingress que no flush-ea por chunk
- herramienta de consola que espera el body completo

En ese caso el orquestador no es el primer sospechoso.

## 8. Si `usage` sale a cero al final del stream

Eso tambien puede ser normal.

Ahora mismo el orquestador solo rellena `usage` final si el upstream manda usage en los chunks.
Si el backend no lo manda durante el stream, `response.completed.usage` puede salir a cero.

## 9. Criterio de diagnostico rapido

- Si no hay log final en el orquestador: la request no llego o no termino.
- Si hay log final y dentro del cluster ves deltas: el problema esta fuera del orquestador.
- Si todo llega en trozos minusculos: eso es comportamiento normal del backend.
