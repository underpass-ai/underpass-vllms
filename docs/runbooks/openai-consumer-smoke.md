# Runbook: OpenAI Consumer Smoke Test

Usa este runbook cuando un consumidor quiere tratar el orquestador como si fuera una API OpenAI.

Sirve para validar:

- descubrimiento de modelo
- `chat.completions`
- `responses`
- `stream=true`
- shape de errores OpenAI

No sirve para validar toda la plataforma OpenAI. Este orquestador solo expone el subconjunto util para structured extraction.

## 1. Descubrir el modelo publico

Primero averigua el `model` exacto que expone el release:

```bash
curl -s http://<host>:8080/v1/models | jq
```

Debes ver un unico modelo en `data[0].id`.

Si no conoces el release exacto dentro del cluster:

```bash
kubectl get svc -n <namespace> | rg orchestrator
```

## 2. Smoke de `chat.completions`

Prueba minima:

```bash
curl -s http://<host>:8080/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{
    "model": "<public-model>",
    "messages": [
      { "role": "user", "content": "Return hello in the value field" }
    ],
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "name": "hello_schema",
        "schema": {
          "type": "object",
          "properties": {
            "value": { "type": "string" }
          },
          "required": ["value"],
          "additionalProperties": false
        }
      }
    }
  }' | jq
```

Esperado:

- `object = "chat.completion"`
- `choices[0].message.role = "assistant"`
- `choices[0].message.content` contiene un JSON serializado
- `usage` presente

## 3. Smoke de `responses`

Prueba minima:

```bash
curl -s http://<host>:8080/v1/responses \
  -H 'content-type: application/json' \
  -d '{
    "model": "<public-model>",
    "input": "Return hello in the value field",
    "text": {
      "format": {
        "type": "json_schema",
        "name": "hello_schema",
        "schema": {
          "type": "object",
          "properties": {
            "value": { "type": "string" }
          },
          "required": ["value"],
          "additionalProperties": false
        }
      }
    }
  }' | jq
```

Esperado:

- `object = "response"`
- `status = "completed"`
- `output_text` contiene el JSON final
- `output[0].content[0].type = "output_text"`

## 4. Smoke de streaming

Solo aplica cuando el backend activo es `single_pass`.

Si el release es `two_pass`, el comportamiento correcto es:

- `400`
- `invalid_request_error`
- mensaje explicando que `stream=true` solo esta soportado en `single_pass`

### `chat.completions` streaming

```bash
curl -sN http://<host>:8080/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{
    "model": "<public-model>",
    "messages": [
      { "role": "user", "content": "Return hello in the value field" }
    ],
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "name": "hello_schema",
        "schema": {
          "type": "object",
          "properties": {
            "value": { "type": "string" }
          },
          "required": ["value"],
          "additionalProperties": false
        }
      }
    },
    "stream": true
  }'
```

Esperado:

- eventos `data: {...chat.completion.chunk...}`
- chunk inicial con `delta.role = "assistant"`
- chunks con `delta.content`
- cierre con `data: [DONE]`

### `responses` streaming

```bash
curl -sN http://<host>:8080/v1/responses \
  -H 'content-type: application/json' \
  -d '{
    "model": "<public-model>",
    "input": "Return hello in the value field",
    "text": {
      "format": {
        "type": "json_schema",
        "name": "hello_schema",
        "schema": {
          "type": "object",
          "properties": {
            "value": { "type": "string" }
          },
          "required": ["value"],
          "additionalProperties": false
        }
      }
    },
    "stream": true
  }'
```

Esperado:

- `response.created`
- varios `response.output_text.delta`
- `response.output_text.done`
- `response.completed`

## 5. Errores esperados y lectura rapida

### Modelo incorrecto

Esperado:

```json
{
  "error": {
    "message": "unsupported model \"wrong-model\"",
    "type": "invalid_request_error",
    "param": "model",
    "code": "model_not_found"
  }
}
```

### Payload invalido

Esperado:

- `400`
- `error.type = "invalid_request_error"`
- `error.param` apuntando al campo conflictivo cuando aplica

### Fallo upstream

Esperado:

- `502`
- `error.type = "server_error"`
- `error.code` con el codigo interno util, por ejemplo `pass2_transport_failure`

## 6. Lo que este facade no soporta

No intentes validar estas features porque no forman parte del contrato actual:

- tools / function calling
- audio
- conversation state persistente
- `previous_response_id`
- `store`
- `background`
- `n > 1`
- `text.format.type = "text"`
- streaming publico en `two_pass`

## 7. Si algo no cuadra

Sigue este orden:

1. [streaming-debug.md](streaming-debug.md) si el problema es streaming
2. [single-pass-release.md](single-pass-release.md) si sospechas del despliegue
3. [../api.md](../api.md) para confirmar el contrato exacto
