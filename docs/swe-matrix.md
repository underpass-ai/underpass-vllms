# SWE Matrix

Esta matriz no es un test de producto ni un benchmark con asserts. Es una bateria manual para perfilar el comportamiento del modelo dentro del flujo del orquestador, tanto en `two_pass` como en `single_pass`.

La idea es mirar:

- cuando extrae bien sin adornar,
- cuando inventa o sobreinfiere,
- cuando usa `null` y `warnings` con disciplina,
- como resuelve conflicto y ambiguedad,
- que tan util queda el JSON para un flujo SWE real.

## Que estresa cada caso

| Case ID | Categoria | Rasgo del modelo que queremos ver |
| --- | --- | --- |
| `bug-regression-triage` | bug triage | fidelidad a hechos explicitos y autocontrol ante una causa solo sospechada |
| `feature-request-acceptance` | feature request | separacion entre problema, acceptance criteria y constraints |
| `incident-prod-checkout` | incident | compresion util de incidentes sin perder impacto, mitigacion ni acciones |
| `pr-review-risk` | PR review | lectura de riesgo, breaking change y orden de rollout |
| `refactor-plan-selector` | refactor | respeto de invariantes y resistencia a expandir el scope |
| `underspecified-dashboard-slowness` | bug triage ambiguo | humildad epistemica: nulls, vacios y warnings |
| `conflicting-incident-evidence` | incident con conflicto | manejo de evidencia contradictoria sin falsa certeza |
| `java-log-connection-timeout` | log analysis | extraccion precisa de stacktrace, retry y next action |

## Artefactos

- Casos: [testdata/swe-matrix/cases.json](/home/tirso/ai/developents/underpass-vllms/testdata/swe-matrix/cases.json)
- Runner: [scripts/run-swe-matrix.sh](/home/tirso/ai/developents/underpass-vllms/scripts/run-swe-matrix.sh)

## Ejecucion

Listar casos:

```bash
bash scripts/run-swe-matrix.sh --list
```

Ejecutar toda la bateria:

```bash
TWO_PASS_SERVER_URL=http://localhost:8080 \
bash scripts/run-swe-matrix.sh
```

Ejecutar un caso concreto y ver tambien la respuesta por terminal:

```bash
TWO_PASS_SERVER_URL=http://localhost:8080 \
bash scripts/run-swe-matrix.sh --case pr-review-risk --show-response
```

## Salida

Cada ejecucion crea un directorio con:

- `REPORT.md`: indice resumido de la tanda
- `<case>.request.json`: payload enviado al orquestador
- `<case>.response.json`: respuesta cruda del orquestador
- `<case>.report.md`: informe por caso con prompts, IR, salida y plantilla de juicio

## Como juzgar

El runner no decide si el caso "pasa". El juicio es humano. La plantilla por caso deja huecos para valorar:

- fidelidad a hechos explicitos,
- alucinaciones o inferencias no soportadas,
- disciplina con `null`, arrays vacios y `warnings`,
- obediencia al schema,
- utilidad real para un flujo SWE.

Si luego quieres comparar modelos, la comparacion deberia hacerse leyendo esos informes lado a lado, no solo mirando un bit de pass o fail.
