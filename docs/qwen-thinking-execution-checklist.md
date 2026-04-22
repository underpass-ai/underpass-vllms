# Checklist de Ejecucion de la Integracion Qwen Thinking

Fecha de referencia: 2026-04-21

Documento base:

- [docs/qwen-thinking-integration-plan.md](/home/tirso/ai/developents/underpass-vllms/docs/qwen-thinking-integration-plan.md)

## Uso

Este documento traduce el plan en trabajo ejecutable.

Regla de uso:

- no avanzar a la fase siguiente si la actual no deja evidencia verificable
- no evaluar el modelo en serio hasta cerrar las fases de integracion
- cualquier decision de tuning debe quedar reflejada en values, codigo o docs

## Fase 0. Congelar la linea base

Objetivo:

- fijar el estado actual antes de tocar la integracion

Checklist:

- [ ] Capturar la revision Helm desplegada del orquestador
- [ ] Capturar imagen y tag desplegados de `reasoning`, `structured` y `orchestrator`
- [ ] Guardar un ejemplo reproducible del fallo `pass1_empty_response`
- [ ] Guardar un ejemplo reproducible del mismo request ejecutado desde el workspace local contra los mismos upstreams
- [ ] Dejar escrita la conclusion de drift si aplica

Evidencia minima:

```bash
kubectl get deploy -n underpass-runtime underpass-llm-orchestrator-orchestrator -o yaml
kubectl get deploy -n underpass-runtime underpass-llm-reasoning-reasoning -o yaml
kubectl get deploy -n underpass-runtime underpass-llm-structured-structured -o yaml
```

Cierre:

- existe una prueba clara de comportamiento `deployed vs local`

## Fase 1. Fijar el contrato interno de completion

Objetivo:

- representar correctamente lo que devuelve vLLM para modelos con reasoning

Checklist:

- [ ] El cliente outbound tipa `content`
- [ ] El cliente outbound tipa `reasoning`
- [ ] El cliente outbound tipa `finish_reason`
- [ ] La capa de aplicacion no colapsa prematuramente todo a una sola string
- [ ] Existen tests unitarios de parseo para:
- [ ] `content only`
- [ ] `reasoning + content`
- [ ] `reasoning only`
- [ ] respuesta vacia real

Cierre:

- el contrato interno puede distinguir sin ambiguedad entre respuesta final y trace de reasoning

## Fase 2. Definir politica de fallback

Objetivo:

- evitar que `reasoning only` se trate como vacio por defecto

Checklist:

- [ ] Decidir formalmente si `reasoning` puede usarse como IR provisional
- [ ] Implementar esa decision de forma explicita
- [ ] Marcar en metadata cuando se use fallback desde reasoning
- [ ] Diferenciar en errores:
- [ ] `empty response`
- [ ] `reasoning without final content`
- [ ] `transport error`
- [ ] Añadir tests de servicio para el camino degradado

Cierre:

- un `reasoning only` no se reporta falsamente como `pass1_empty_response`

## Fase 3. Presupuesto de thinking en Pass 1

Objetivo:

- evitar truncado sistematico antes del cierre final

Checklist:

- [ ] Revisar si nuestra version/ruta de vLLM soporta `thinking_token_budget`
- [ ] Si se soporta, exponerlo en config y Helm
- [ ] Si no se soporta, documentar la limitacion
- [ ] Revisar `PASS1_MAX_TOKENS`
- [ ] Elegir un presupuesto inicial razonable para `Pass 1`
- [ ] Volver a probar el caso de bug triage y al menos un caso ambiguo
- [ ] Medir si baja el ratio de `finish_reason=length`

Evidencia minima:

- request real a `reasoning`
- `content`
- `reasoning`
- `finish_reason`
- `usage`

Cierre:

- el truncado de Pass 1 deja de ser el modo normal de operacion

## Fase 4. Cierre fuerte de IR en Pass 1

Objetivo:

- conseguir que el modelo piense pero termine con una IR usable

Checklist:

- [ ] Revisar el prompt de sistema de Pass 1
- [ ] Revisar el prompt de usuario de Pass 1
- [ ] Definir si hace falta un delimitador explicito de salida final
- [ ] Probar al menos dos variantes de cierre final de IR
- [ ] Elegir una variante estable y documentarla
- [ ] Verificar que Pass 2 recibe menos ruido y menos prompt inflado

Opciones aceptables:

- `FINAL_IR:`
- bloque JSON final delimitado
- bloque textual compacto con secciones fijas

Cierre:

- Pass 1 termina de forma consistente con una IR final separable

## Fase 5. Observabilidad

Objetivo:

- diagnosticar el stack sin depender de inspeccion manual puntual

Checklist:

- [ ] Loggear `finish_reason` en Pass 1
- [ ] Loggear presencia de `content`
- [ ] Loggear presencia de `reasoning`
- [ ] Loggear si se ha usado fallback
- [ ] Exponer metricas por pass:
- [ ] latencia
- [ ] prompt tokens
- [ ] completion tokens
- [ ] ratio de truncado
- [ ] ratio de fallback
- [ ] Documentar lectura operativa de esas metricas

Cierre:

- se puede explicar un fallo de Pass 1 sin abrir manualmente respuestas raw

## Fase 6. Control de drift entre repo e imagen

Objetivo:

- que el comportamiento desplegado corresponda al codigo esperado

Checklist:

- [ ] Establecer trazabilidad entre commit y tag de imagen
- [ ] Documentar como validar el artefacto desplegado
- [ ] Añadir verificacion minima en CI o release
- [ ] Evitar despliegues donde el repo local y la imagen se contradicen funcionalmente

Evidencia minima:

- commit o ref origen
- tag de imagen
- release Helm

Cierre:

- deja de ser posible la situacion `local bien / prod mal` sin explicacion trazable

## Fase 7. Validacion funcional del stack

Objetivo:

- confirmar que la integracion ya se comporta como producto antes de medir el modelo

Checklist:

- [ ] Repetir el caso de bug triage end-to-end
- [ ] Repetir un caso underspecified
- [ ] Repetir un caso conflicting evidence
- [ ] Confirmar que el desplegado y el local dan resultados equivalentes en clase de salida
- [ ] Confirmar que Pass 2 sigue produciendo JSON valido y util

Cierre:

- el stack two-pass queda estable para evaluacion

## Fase 8. Informes y evaluacion del modelo

Objetivo:

- evaluar por fin al modelo y no al fallo de integracion

Checklist:

- [ ] Ejecutar la matriz SWE en modo informe
- [ ] Revisar respuestas y hacer de juez
- [ ] Separar hallazgos de modelo frente a hallazgos de integracion
- [ ] Guardar comparativas entre configuraciones de Pass 1 si aplica

Cierre:

- ya se puede hablar de comportamiento del modelo con seriedad

## Definition of Done

La integracion puede considerarse lista cuando:

- [ ] El orquestador desplegado ya no devuelve `pass1_empty_response` para casos normales con thinking
- [ ] El sistema diferencia `content`, `reasoning` y truncado
- [ ] Existe una politica clara de fallback
- [ ] Pass 1 tiene presupuesto suficiente para pensar y cerrar
- [ ] Hay trazabilidad entre codigo, imagen y despliegue
- [ ] La matriz SWE se ejecuta como informe y no como detector accidental de bugs del stack

## Proxima Accion Recomendada

Empezar por Fase 0 y Fase 1 juntas:

- congelar la linea base
- fijar el contrato interno de completion

Sin esas dos, el resto del trabajo sigue apoyado en suposiciones.
