# Plan de Integracion Perfecta de Qwen Thinking

Fecha de referencia: 2026-04-21

Documento operativo relacionado:

- [docs/qwen-thinking-execution-checklist.md](/home/tirso/ai/developents/underpass-vllms/docs/qwen-thinking-execution-checklist.md)

## Objetivo

Este documento define el plan para una integracion robusta de `Qwen3.6 + vLLM + two-pass orchestration` en la que:

- Pass 1 aprovecha el `thinking`,
- el `thinking` no se pierde ni se confunde con un fallo,
- Pass 2 recibe una representacion intermedia util y la convierte a JSON estricto,
- el sistema sigue siendo observable, reproducible y desplegable sin sorpresas.

Aqui "integracion perfecta" no significa "sin fallos posibles", sino una integracion que:

- respeta el contrato oficial de vLLM y Qwen,
- separa bien `reasoning trace` de `final answer`,
- tolera modos degradados sin romper el producto,
- permite evaluar el modelo sin confundir problemas del stack con problemas del LLM.

## Hechos observados

Observado el 2026-04-21:

- El orquestador desplegado en `underpass-runtime` devolvia `pass1_empty_response`.
- El `reasoning` endpoint de vLLM devolvia `200 OK`, pero con:
  - `message.reasoning` poblado,
  - `message.content = null`,
  - `finish_reason = "length"`.
- El mismo upstream de `reasoning` y `structured`, usado desde el orquestador local del workspace, completo el flujo correctamente.

Conclusion operativa:

- el modelo no estaba roto,
- el principal problema estaba en la integracion y/o el artefacto desplegado,
- ademas el tuning de `Pass 1` era demasiado agresivo para usar `thinking` con seguridad.

## Contrato Oficial Que Debemos Respetar

Fuentes oficiales:

- vLLM Reasoning Outputs: `https://docs.vllm.ai/en/latest/features/reasoning_outputs/`
- vLLM OpenAI Chat Completion With Reasoning: `https://docs.vllm.ai/en/v0.18.1/examples/online_serving/openai_chat_completion_with_reasoning/`
- Qwen3.6 model card: `https://huggingface.co/Qwen/Qwen3.6-35B-A3B`

Puntos que el sistema debe asumir como verdad de integracion:

1. En modelos reasoning, vLLM puede devolver dos salidas distintas:
   - `message.reasoning`
   - `message.content`
2. En la serie Qwen3, el thinking esta activado por defecto.
3. `chat_template_kwargs.enable_thinking=false` desactiva thinking por request o por server default.
4. Si no se configura `thinking_token_budget`, el thinking consume del `max_tokens` normal.
5. Un `content` vacio no implica necesariamente "modelo vacio"; puede significar:
   - razonamiento sin cierre final,
   - presupuesto agotado antes de emitir la respuesta final.

## Estado Actual

### Lo que ya esta bien

- Separacion clara entre `reasoning`, `structured` y `orchestrator`.
- Pass 2 ya esta planteado como canonicalizacion JSON con thinking desactivado.
- El endpoint `reasoning` ya usa `--reasoning-parser=qwen3`.

### Lo que hoy es fragil

- No existe un contrato de dominio explicito para distinguir:
  - `reasoning trace`
  - `final answer`
  - `finish_reason`
- El sistema puede tratar un caso de `content` vacio como si fuese "respuesta vacia" sin suficiente contexto.
- `Pass 1` usa `max_tokens` bajo para thinking, y no expone `thinking_token_budget`.
- No hay una politica declarada para modo degradado cuando:
  - hay `reasoning`,
  - no hay `content`,
  - o el cierre final llega truncado.
- No hay suficientes metricas para separar:
  - fallo del modelo,
  - fallo del parser,
  - fallo del prompt,
  - fallo de tuning.

## Arquitectura Objetivo

### Pass 1

Pass 1 debe producir tres artefactos logicos:

- `reasoning_trace`: razonamiento del modelo, si existe
- `final_ir`: representacion intermedia final
- `completion_status`: informacion util para decision operacional
  - `finish_reason`
  - `used_reasoning_fallback`
  - `truncated`

Politica objetivo:

- Camino ideal:
  - `reasoning_trace` existe
  - `final_ir` existe en `content`
  - Pass 2 consume `final_ir`
- Camino degradado aceptable:
  - `content` vacio
  - `reasoning_trace` presente
  - el orquestador puede usar reasoning como IR provisional, pero debe marcarlo explicitamente en metadata
- Camino fallido:
  - no hay `content`
  - no hay `reasoning`
  - o ambos son inutiles

### Pass 2

Pass 2 debe seguir siendo no-thinking por defecto:

- recibe `final_ir` o `reasoning-as-ir` en degradado,
- canonicaliza a JSON estricto,
- no inventa campos,
- deja trazabilidad suficiente para saber si trabajo sobre IR final o sobre fallback.

## Diseño Deseado del Contrato Interno

### CompletionResponse

El cliente outbound hacia OpenAI-compatible debe modelar al menos:

- `content`
- `reasoning`
- `finish_reason`
- `usage`

El dominio no deberia colapsar todo a una sola string demasiado pronto.

### Metadata del two-pass

La respuesta del orquestador deberia poder expresar:

- `pass1.finish_reason`
- `pass1.reasoning_present`
- `pass1.content_present`
- `pass1.used_reasoning_fallback`
- `pass1.truncated`

Eso evita diagnosticos falsos tipo "el modelo no ha respondido".

## Plan de Trabajo

### Fase 1. Congelar el contrato real

Objetivo:

- hacer explicito lo que vLLM y Qwen devuelven,
- dejar de inferir a ciegas.

Trabajo:

- tipar `reasoning`, `content` y `finish_reason` en el cliente outbound
- conservarlos hasta la capa de aplicacion
- registrar de forma estructurada los casos:
  - `content only`
  - `reasoning + content`
  - `reasoning only`
  - `empty`

Salida esperada:

- una unica ruta de parseo soportada y observable

### Fase 2. Definir la politica de fallback

Objetivo:

- decidir explicitamente cuando un `reasoning only` es aceptable.

Politica recomendada:

- preferir siempre `content` como `final_ir`
- usar `reasoning` como fallback solo si:
  - `content` esta vacio,
  - `reasoning` no esta vacio,
  - Pass 1 no ha fallado a nivel de transporte
- marcar ese caso como degradado

Salida esperada:

- no volver a mapear `reasoning only` a "respuesta vacia"

### Fase 3. Ajustar el presupuesto de thinking

Objetivo:

- evitar que Pass 1 agote todo el presupuesto antes del cierre final.

Trabajo:

- exponer `thinking_token_budget` en config y Helm si vLLM lo soporta en nuestra ruta
- revisar `PASS1_MAX_TOKENS`
- ajustar Pass 1 para coding/triage con mas margen que el actual

Regla de diseño:

- `thinking budget` y `output budget` deben pensarse por separado, aunque tecnicamente compartan un mismo request

Salida esperada:

- menos casos con `finish_reason=length` sin `content`

### Fase 4. Endurecer el prompting de Pass 1

Objetivo:

- pedir thinking util sin sacrificar una IR final compacta.

Trabajo:

- reforzar el prompt de Pass 1 para que:
  - piense si hace falta,
  - mantenga el reasoning breve,
  - termine siempre con una IR compacta y separable
- evaluar si conviene introducir un delimitador claro para la IR final, por ejemplo:
  - `FINAL_IR:`
  - bloque JSON delimitado
  - bloque XML simple

Salida esperada:

- Pass 2 recibe entradas mas limpias y con menos ruido de chain-of-thought

### Fase 5. Observabilidad y operacion

Objetivo:

- convertir la integracion en algo auditable.

Metricas minimas:

- ratio de respuestas `reasoning only`
- ratio de respuestas `content only`
- ratio de fallback a reasoning
- ratio de `finish_reason=length`
- latencia de Pass 1 y Pass 2
- tokens de prompt y completion por pass
- tasa de reintentos de Pass 2

Logs minimos:

- request id
- modelo
- finish reason
- presencia de content
- presencia de reasoning
- fallback usado o no

### Fase 6. Despliegue y control de drift

Objetivo:

- que el codigo desplegado coincida con el contrato esperado.

Trabajo:

- versionar la imagen del orquestador de forma trazable
- documentar que commit o artefacto produjo la imagen
- verificar en CI o release que la imagen desplegada corresponde al codigo esperado

Salida esperada:

- eliminar dudas tipo "el repo local lo hace bien, pero prod no"

### Fase 7. Evaluacion del modelo

Solo despues de estabilizar Fases 1 a 6.

Entonces si tiene sentido ejecutar:

- la matriz SWE en modo informe
- comparativas entre modelos
- pruebas de factualidad, ambiguedad y conflicto

Sin esta fase previa, la evaluacion del modelo sigue contaminada por el stack.

## Criterios de Aceptacion

La integracion se considerara lista cuando:

1. Un `reasoning only` ya no se clasifique erroneamente como `pass1_empty_response`.
2. El orquestador pueda distinguir en metadata entre:
   - `content final`
   - `reasoning fallback`
3. El ratio de `finish_reason=length` en Pass 1 quede bajo control para la carga objetivo.
4. El despliegue y el workspace local den el mismo resultado para el mismo request contra los mismos upstreams.
5. La matriz SWE pueda ejecutarse sin confundir errores del stack con errores del modelo.

## Decisiones Recomendadas

### Decisiones que recomiendo tomar ya

- Mantener la arquitectura two-pass.
- Mantener `thinking` en Pass 1.
- Mantener Pass 2 sin thinking.
- Añadir soporte completo a `reasoning` y `finish_reason` como parte del contrato interno.
- Introducir tuning explicito de presupuesto para Pass 1.

### Decisiones que no recomiendo tomar aun

- Desactivar thinking globalmente en Pass 1.
- Concluir que Qwen3.6 "inventa" o "no sirve" a partir del estado actual.
- Ejecutar comparativas serias de modelos antes de estabilizar la integracion.

## Riesgos

- Usar `reasoning` bruto como IR puede inflar mucho el prompt de Pass 2.
- Si no se marca el fallback, se pierde visibilidad y se normaliza un modo degradado como si fuese ideal.
- Si el prompt de Pass 1 no obliga a un cierre claro, Pass 2 seguira absorbiendo demasiado ruido.
- Si no se controla el drift entre imagen y repo, cualquier conclusion tecnica sera dudosa.

## Siguiente Paso Operativo

Orden recomendado:

1. Verificar y fijar el contrato interno `reasoning/content/finish_reason`.
2. Alinear imagen desplegada con el comportamiento correcto del workspace.
3. Ajustar presupuesto de Pass 1.
4. Endurecer prompt de cierre de IR.
5. Solo entonces ejecutar informes y comparativas de modelos.
