# Orchestrator API

## Endpoints

| Método | Ruta | Uso |
| --- | --- | --- |
| `GET` | `/healthz` | health check |
| `GET` | `/readyz` | readiness check |
| `GET` | `/v1/models` | lista OpenAI-compatible de modelos expuestos por el orquestador |
| `GET` | `/v1/models/{id}` | detalle OpenAI-compatible del modelo expuesto |
| `POST` | `/v1/chat/completions` | fachada OpenAI-compatible sobre el mismo caso de uso |
| `POST` | `/v1/two-pass/structured` | ejecucion estructurada completa en `two_pass` o `single_pass` |

La ruta mantiene el nombre `two-pass` por compatibilidad historica. El modo real de ejecucion se expone en `metadata.execution_mode`.

La compatibilidad OpenAI esta centrada en structured extraction:

- `messages`
- `model`
- `response_format`
- `POST /v1/chat/completions`

No intenta replicar toda la plataforma OpenAI. La parte util aqui es mapear clientes de `chat.completions` al mismo orquestador sin forzarles a consumir la API custom.

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
- `stream=false`

### Restricciones intencionales

- no hay streaming
- no hay tools
- no hay tool calls
- no hay audio
- el caso de uso principal es `response_format.type=json_schema`
- `json_object` tambien se acepta con un schema permisivo

### Mapeo interno

- el transcript de `messages` se aplana a `input`
- `response_format` aporta el `schema`
- `model` se propaga como override a `pass1`, `pass2` y `single_pass`
- la respuesta final se devuelve como `choices[0].message.content`, con el JSON serializado como string

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
