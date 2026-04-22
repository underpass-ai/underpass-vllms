# Orchestrator API

## Endpoints

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/healthz` | health check |
| `GET` | `/readyz` | readiness check |
| `GET` | `/v1/models` | lista OpenAI-compatible de modelos expuestos por el orquestador |
| `GET` | `/v1/models/{id}` | detalle OpenAI-compatible del modelo expuesto |
| `POST` | `/v1/chat/completions` | fachada OpenAI-compatible sobre el mismo caso de uso |
| `POST` | `/v1/responses` | fachada Responses API sobre el mismo caso de uso |
| `POST` | `/v1/two-pass/structured` | ejecucion estructurada completa en `two_pass` o `single_pass` |

La ruta mantiene el nombre `two-pass` por compatibilidad historica. El modo real de ejecucion se expone en `metadata.execution_mode`.

La compatibilidad OpenAI esta centrada en structured extraction:

- `messages`
- `model`
- `response_format`
- `POST /v1/chat/completions`

No intenta replicar toda la plataforma OpenAI. La parte util aqui es mapear clientes de `chat.completions` al mismo orquestador sin forzarles a consumir la API custom.

La misma filosofia aplica a `POST /v1/responses`: se soporta el subconjunto util para structured extraction, no toda la superficie completa de la Responses API.

## Request Body

### `POST /v1/two-pass/structured`

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `request_id` | `string` | no | si no se envía, el servicio genera uno |
| `input` | `string` | sí | payload de entrada |
| `schema_version` | `string` | no | etiqueta lógica devuelta en metadata |
| `schema` | `json` | sí | debe ser JSON válido |
| `include_intermediate` | `boolean` | no | si se omite, la respuesta incluye `intermediate_representation` |
| `pass1` | `object` | no | overrides de Pass 1 |
| `pass2` | `object` | no | overrides de Pass 2 en `two_pass`; alias legacy en `single_pass` |
| `single_pass` | `object` | no | overrides del unico pase en `single_pass` |

### `pass1`, `pass2` y `single_pass`

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `model` | `string` | no | override del modelo configurado |
| `system_prompt` | `string` | no | override del prompt del sistema |
| `temperature` | `number` | no | override de temperatura |
| `top_p` | `number` | no | override de `top_p` |
| `top_k` | `integer` | no | override de `top_k` |
| `presence_penalty` | `number` | no | override de `presence_penalty` |
| `repetition_penalty` | `number` | no | override de `repetition_penalty` |
| `max_tokens` | `integer` | no | override de max tokens |
| `thinking_token_budget` | `integer` | no | límite de tokens para thinking en modelos compatibles |
| `preserve_thinking` | `boolean` | no | preserva thinking histórico en modelos compatibles |

## Response Body

| Field | Type | Presence | Notes |
| --- | --- | --- | --- |
| `request_id` | `string` | siempre | id de trazabilidad |
| `intermediate_representation` | `string` | por defecto sí | omitido si `include_intermediate=false` |
| `output` | `json` | siempre | JSON validado contra el schema |
| `metadata` | `object` | siempre | métricas y versiones lógicas |

### `metadata`

| Field | Type | Notes |
| --- | --- | --- |
| `schema_version` | `string` | eco del request |
| `pass1_prompt_version` | `string` | solo en `two_pass` |
| `pass2_prompt_version` | `string` | solo en `two_pass` |
| `single_pass_prompt_version` | `string` | solo en `single_pass` |
| `ir_version` | `string` | versión lógica del IR |
| `pass1` | `object` | solo en `two_pass` |
| `pass2` | `object` | solo en `two_pass` |
| `single_pass` | `object` | solo en `single_pass` |

### Métricas por ejecucion

| Field | Type |
| --- | --- |
| `model` | `string` |
| `attempts` | `integer` |
| `latency_ms` | `integer` |
| `prompt_tokens` | `integer` |
| `completion_tokens` | `integer` |
| `finish_reason` | `string` |
| `content_present` | `boolean` |
| `reasoning_present` | `boolean` |
| `used_reasoning_fallback` | `boolean` |
| `truncated` | `boolean` |

## Error Model

### Validación de request

- `400 Bad Request` si `input` falta o está vacío
- `400 Bad Request` si `schema` no es JSON válido

### Método no soportado

- `405 Method Not Allowed` si `/v1/two-pass/structured` no se invoca con `POST`

### Fallos upstream o de validación final

- `502 Bad Gateway` si falla `pass1`
- `502 Bad Gateway` si falla `pass2`
- `502 Bad Gateway` si falla `single_pass`
- `502 Bad Gateway` si la salida final devuelve JSON inválido o que no valida contra el schema

## Comportamiento interno relevante

- Pass 1 llama a `POST <PASS1_BASE_URL>/chat/completions`
- Pass 2 llama a `POST <PASS2_BASE_URL>/chat/completions`
- `single_pass` llama una sola vez a `POST <PASS2_BASE_URL>/chat/completions`
- El cliente siempre envía header `Authorization: Bearer <apiKey>`
- Pass 1 puede usar `message.reasoning` como fallback si `message.content` viene vacío
- Pass 1 puede añadir `top_p`, `top_k`, `presence_penalty`, `repetition_penalty`, `thinking_token_budget` y `chat_template_kwargs.preserve_thinking`
- Pass 2 añade:
  - `structured_outputs.json`
  - el schema también se inyecta en el prompt de canonicalización
  - `chat_template_kwargs.enable_thinking=false`

## Prompts internos por defecto

Si no se sobreescriben por request, el servicio usa estos prompts base:

- Pass 1: extracción semántica e IR completo
- Pass 2: canonicalización JSON estricta sin inventar campos
- Single pass: extracción JSON estricta en una sola llamada

## Example Request

```bash
curl -s http://localhost:8080/v1/two-pass/structured \
  -H 'content-type: application/json' \
  -d '{
    "input": "Invoice INV-2026-0017 from ACME Logistics. Total 1540.25 EUR. Invoice date 2026-04-18. Due 2026-05-18.",
    "schema_version": "invoice-v1",
    "schema": {
      "type": "object",
      "properties": {
        "invoice_number": { "type": "string" },
        "supplier_name": { "type": "string" },
        "currency": { "type": "string" },
        "total_amount": { "type": "number" },
        "invoice_date": { "type": "string" },
        "due_date": { "type": "string" },
        "warnings": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": [
        "invoice_number",
        "supplier_name",
        "currency",
        "total_amount",
        "invoice_date",
        "due_date",
        "warnings"
      ],
      "additionalProperties": false
    }
  }'
```

## Example Success Shape

```json
{
  "request_id": "4c5f7b0e2c9a4b11",
  "intermediate_representation": "Task type: invoice_extraction\n...",
  "output": {
    "invoice_number": "INV-2026-0017",
    "supplier_name": "ACME Logistics",
    "currency": "EUR",
    "total_amount": 1540.25,
    "invoice_date": "2026-04-18",
    "due_date": "2026-05-18",
    "warnings": [
      "VAT amount not found explicitly"
    ]
  },
  "metadata": {
    "schema_version": "invoice-v1",
    "pass1_prompt_version": "2026-04-21.2",
    "pass2_prompt_version": "2026-04-21.1",
    "ir_version": "1.0.0",
    "pass1": {
      "model": "Qwen/Qwen3.6-35B-A3B",
      "attempts": 1,
      "latency_ms": 1234,
      "prompt_tokens": 100,
      "completion_tokens": 200,
      "finish_reason": "stop",
      "content_present": true,
      "reasoning_present": false,
      "used_reasoning_fallback": false,
      "truncated": false
    },
    "pass2": {
      "model": "Qwen/Qwen3.6-35B-A3B",
      "attempts": 1,
      "latency_ms": 456,
      "prompt_tokens": 120,
      "completion_tokens": 90,
      "finish_reason": "stop",
      "content_present": true,
      "reasoning_present": false,
      "used_reasoning_fallback": false,
      "truncated": false
    }
  }
}
```

## Example Success Shape For `single_pass`

```json
{
  "request_id": "4c5f7b0e2c9a4b11",
  "output": {
    "invoice_number": "INV-2026-0017",
    "supplier_name": "ACME Logistics",
    "currency": "EUR",
    "total_amount": 1540.25,
    "invoice_date": "2026-04-18",
    "due_date": "2026-05-18",
    "warnings": []
  },
  "metadata": {
    "schema_version": "invoice-v1",
    "single_pass_prompt_version": "2026-04-21.1",
    "ir_version": "1.0.0",
    "single_pass": {
      "model": "google/gemma-4-31B-it",
      "attempts": 1,
      "latency_ms": 3345,
      "prompt_tokens": 170,
      "completion_tokens": 7,
      "finish_reason": "stop",
      "content_present": true,
      "reasoning_present": false,
      "used_reasoning_fallback": false,
      "truncated": false
    }
  }
}
```

## OpenAI-Compatible Chat Completions

### Request soportada

`POST /v1/chat/completions` acepta un subconjunto util del contrato OpenAI:

- `model`
- `messages`
- `response_format`
- `temperature`
- `top_p`
- `presence_penalty`
- `max_completion_tokens`
- `max_tokens`
- `reasoning_effort`
- `n` solo con valor `1`
- `stream`

### Restricciones intencionales

- no hay tools
- no hay tool calls
- no hay audio
- el caso de uso principal es `response_format.type=json_schema`
- `json_object` tambien se acepta con un schema permisivo
- `stream=true` solo se soporta cuando el backend activo es `single_pass`
- si el orquestador esta usando `two_pass`, `stream=true` devuelve `400 invalid_request_error`

### Mapeo interno

- el transcript de `messages` se aplana a `input`
- `response_format` aporta el `schema`
- `model` se propaga como override a `pass1`, `pass2` y `single_pass`
- la respuesta final se devuelve como `choices[0].message.content`, con el JSON serializado como string
- con `stream=true`, la respuesta se emite como SSE `chat.completion.chunk` y termina con `data: [DONE]`

### Example Request

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{
    "model": "google/gemma-4-31B-it",
    "messages": [
      {
        "role": "developer",
        "content": "Extract data strictly and return only valid JSON."
      },
      {
        "role": "user",
        "content": "Invoice INV-2026-0017 from ACME Logistics. Total 1540.25 EUR."
      }
    ],
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "name": "invoice_v1",
        "strict": true,
        "schema": {
          "type": "object",
          "properties": {
            "invoice_number": { "type": "string" },
            "currency": { "type": "string" }
          },
          "required": ["invoice_number", "currency"],
          "additionalProperties": false
        }
      }
    }
  }'
```

### Example Response

```json
{
  "id": "chatcmpl-75e81d175a16f2a1",
  "object": "chat.completion",
  "created": 1760000000,
  "model": "google/gemma-4-31B-it",
  "request_id": "75e81d175a16f2a1",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 170,
    "completion_tokens": 7,
    "total_tokens": 177
  },
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "invoice_v1",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": {
          "invoice_number": { "type": "string" },
          "currency": { "type": "string" }
        },
        "required": ["invoice_number", "currency"],
        "additionalProperties": false
      }
    }
  }
}
```

### Example Streaming Response

`POST /v1/chat/completions` con `stream=true` devuelve SSE de tipo OpenAI:

```text
data: {"id":"chatcmpl-75e81d175a16f2a1","object":"chat.completion.chunk","created":1760000000,"model":"google/gemma-4-31B-it","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-75e81d175a16f2a1","object":"chat.completion.chunk","created":1760000000,"model":"google/gemma-4-31B-it","choices":[{"index":0,"delta":{"content":"{\"invoice_number\":"},"finish_reason":null}]}

data: {"id":"chatcmpl-75e81d175a16f2a1","object":"chat.completion.chunk","created":1760000000,"model":"google/gemma-4-31B-it","choices":[{"index":0,"delta":{"content":"\"INV-2026-0017\",\"currency\":\"EUR\"}"},"finish_reason":null}]}

data: {"id":"chatcmpl-75e81d175a16f2a1","object":"chat.completion.chunk","created":1760000000,"model":"google/gemma-4-31B-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

## OpenAI-Compatible Responses API

### Request soportada

`POST /v1/responses` acepta un subconjunto util del contrato oficial:

- `model`
- `input`
- `instructions`
- `text.format`
- `temperature`
- `top_p`
- `max_output_tokens`
- `reasoning.effort`
- `stream`

### Restricciones intencionales

- no hay tools
- no hay conversation state
- no hay `previous_response_id`
- el caso de uso principal es `text.format.type=json_schema`
- `json_object` tambien se acepta
- `text.format.type=text` no se soporta en este orquestador
- `stream=true` solo se soporta cuando el backend activo es `single_pass`
- si el orquestador esta usando `two_pass`, `stream=true` devuelve `400 invalid_request_error`

### Mapeo interno

- `instructions` se inserta como bloque de instrucciones al principio del `input` interno
- `input` string se trata como mensaje `user`
- `input` array se aplana a transcript por roles
- `text.format` aporta el schema
- la respuesta se devuelve como objeto `response` con:
  - `output`
  - `output_text`
  - `usage`
  - `text.format`
- con `stream=true`, la respuesta se emite como SSE con eventos semanticos:
  - `response.created`
  - `response.output_text.delta`
  - `response.output_text.done`
  - `response.completed`

### Example Request

```bash
curl -s http://localhost:8080/v1/responses \
  -H 'content-type: application/json' \
  -d '{
    "model": "google/gemma-4-31B-it",
    "instructions": "Extract strictly and return valid JSON.",
    "input": "Invoice INV-2026-0017 from ACME Logistics. Total 1540.25 EUR.",
    "text": {
      "format": {
        "type": "json_schema",
        "name": "invoice_v1",
        "strict": true,
        "schema": {
          "type": "object",
          "properties": {
            "invoice_number": { "type": "string" },
            "currency": { "type": "string" }
          },
          "required": ["invoice_number", "currency"],
          "additionalProperties": false
        }
      }
    }
  }'
```

### Example Response

```json
{
  "id": "resp_053abd05e4c7decb",
  "object": "response",
  "created_at": 1760000000,
  "completed_at": 1760000000,
  "status": "completed",
  "error": null,
  "incomplete_details": null,
  "instructions": "Extract strictly and return valid JSON.",
  "max_output_tokens": null,
  "model": "google/gemma-4-31B-it",
  "output": [
    {
      "id": "msg_053abd05e4c7decb",
      "type": "message",
      "status": "completed",
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}",
          "annotations": []
        }
      ]
    }
  ],
  "output_text": "{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}",
  "reasoning": {
    "effort": null,
    "summary": null
  },
  "text": {
    "format": {
      "type": "json_schema",
      "name": "invoice_v1",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": {
          "invoice_number": { "type": "string" },
          "currency": { "type": "string" }
        },
        "required": ["invoice_number", "currency"],
        "additionalProperties": false
      }
    }
  },
  "usage": {
    "input_tokens": 191,
    "output_tokens": 6,
    "output_tokens_details": {
      "reasoning_tokens": 0
    },
    "total_tokens": 197
  },
  "metadata": {}
}
```

### Example Streaming Response

`POST /v1/responses` con `stream=true` devuelve SSE semantico sin `data: [DONE]`:

```text
data: {"type":"response.created","sequence_number":0,"response":{"id":"resp_053abd05e4c7decb","object":"response","created_at":1760000000,"completed_at":0,"status":"in_progress","error":null,"incomplete_details":null,"instructions":"Extract strictly and return valid JSON.","max_output_tokens":null,"model":"google/gemma-4-31B-it","output":[],"output_text":"","reasoning":{"effort":null,"summary":null},"text":{"format":{"type":"json_schema","name":"invoice_v1","schema":{"type":"object"}}},"usage":{"input_tokens":0,"output_tokens":0,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":0},"metadata":{}}}

data: {"type":"response.output_text.delta","sequence_number":1,"response_id":"resp_053abd05e4c7decb","item_id":"msg_053abd05e4c7decb","output_index":0,"content_index":0,"delta":"{\"invoice_number\":"}

data: {"type":"response.output_text.done","sequence_number":2,"response_id":"resp_053abd05e4c7decb","item_id":"msg_053abd05e4c7decb","output_index":0,"content_index":0,"text":"{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}"}

data: {"type":"response.completed","sequence_number":3,"response":{"id":"resp_053abd05e4c7decb","object":"response","created_at":1760000000,"completed_at":1760000000,"status":"completed","error":null,"incomplete_details":null,"instructions":"Extract strictly and return valid JSON.","max_output_tokens":null,"model":"google/gemma-4-31B-it","output":[{"id":"msg_053abd05e4c7decb","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}","annotations":[]}]}],"output_text":"{\"invoice_number\":\"INV-2026-0017\",\"currency\":\"EUR\"}","reasoning":{"effort":null,"summary":null},"text":{"format":{"type":"json_schema","name":"invoice_v1","schema":{"type":"object"}}},"usage":{"input_tokens":170,"output_tokens":7,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":177},"metadata":{}}}
```
