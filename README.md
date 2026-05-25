# underpass-vllms

`underpass-vllms` contiene el stack de structured output de Underpass AI.

Soporta dos perfiles de ejecucion:

- `two_pass`: `reasoning` + `structured` + `orchestrator`
- `single_pass`: `structured` + `orchestrator`

Los perfiles hoy soportados en el orquestador son:

- `qwen_reasoning` -> `two_pass`
- `gpt_oss` -> `single_pass`
- `gemma4` -> `single_pass`

## Qué hay en el repo

- [charts/vllm](charts/vllm): chart Helm unico para desplegar `reasoning`, `structured` y `orchestrator`.
- [cmd/two-pass-server](cmd/two-pass-server): binario principal del orquestador.
- [env/prod/](env/prod/): profiles de production activos.
- [env/lab/](env/lab/): profiles de referencia, no en production.
- [env/components/](env/components/): profiles parciales de `two_pass`.
- [env/archive/](env/archive/): profiles archivados, no en uso.
- [docs/archive/](docs/archive/): documentacion archivada, no en uso.
- [deploy/kubernetes/e2e-job.yaml](deploy/kubernetes/e2e-job.yaml): job e2e contra el orquestador.
- [Makefile](Makefile): entrada operativa para desplegar cada servicio.

## Orden de lectura

1. [docs/deployment.md](docs/deployment.md)
2. [docs/runbooks/README.md](docs/runbooks/README.md)
3. [docs/values-reference.md](docs/values-reference.md)
4. [docs/api.md](docs/api.md)
5. [docs/model-lab-notes-2026-04-21.md](docs/model-lab-notes-2026-04-21.md)
6. [docs/gemma-4-family.md](docs/gemma-4-family.md)
7. [docs/gemma-4-26b-a4b.md](docs/gemma-4-26b-a4b.md)
8. [docs/gemma-4-31b.md](docs/gemma-4-31b.md)
9. [docs/gpt-oss-120b.md](docs/gpt-oss-120b.md)
10. [docs/gpt-oss-20b.md](docs/gpt-oss-20b.md)
11. [docs/qwen-thinking-integration-plan.md](docs/qwen-thinking-integration-plan.md)
12. [docs/qwen-thinking-execution-checklist.md](docs/qwen-thinking-execution-checklist.md)

## Convencion de despliegue

Hay dos patrones validos:

1. `two_pass` separado por componente:
   - una release para `reasoning`
   - una release para `structured`
   - una release para `orchestrator`
2. `single_pass` empaquetado por perfil de modelo:
   - una release con `structured + orchestrator`
   - `reasoning` desactivado

Los targets `make` cubren bien el primer caso:

```bash
make helm-upgrade-reasoning NAMESPACE=<namespace> VALUES=<values-file>
make helm-upgrade-structured NAMESPACE=<namespace> VALUES=<values-file>
make helm-upgrade-orchestrator NAMESPACE=<namespace> VALUES=<values-file>
```

Para el segundo caso, usa `helm upgrade --install <release> charts/vllm -f <values-file>` con `reasoning.enabled=false`.

No uses `charts/vllm/values.yaml` como values de entorno. Ese archivo es la base del chart, no una configuracion de despliegue cerrada.

## Flujo funcional

### `two_pass`

1. `reasoning` recibe la entrada y produce una representacion intermedia.
2. `structured` transforma esa representacion en JSON estricto.
3. `orchestrator` valida el JSON contra el schema recibido en la peticion.

### `single_pass`

1. `structured` recibe la entrada original y el schema.
2. `orchestrator` pide una sola salida JSON y la valida contra el schema.

La ruta HTTP sigue siendo `/v1/two-pass/structured` por compatibilidad historica, pero la metadata de respuesta expone el `execution_mode` real.

La diferencia entre `reasoning` y `structured` no la adivina el chart. Debes declarar explicitamente las flags de vLLM adecuadas en `extraArgs`.

## Superficie API

- API publica del orquestador:
  - custom en `/v1/two-pass/structured`
  - OpenAI-compatible en `/v1/chat/completions`, `/v1/responses` y `/v1/models`
  - `stream=true` en `/v1/chat/completions` y `/v1/responses` solo cuando el backend activo es `single_pass`
- APIs upstream de inferencia: OpenAI-compatible (`/v1/chat/completions`)

## Runbooks utiles

- despliegue y validacion `single_pass`: [docs/runbooks/single-pass-release.md](docs/runbooks/single-pass-release.md)
- smoke test para consumidores OpenAI: [docs/runbooks/openai-consumer-smoke.md](docs/runbooks/openai-consumer-smoke.md)
- debug de streaming: [docs/runbooks/streaming-debug.md](docs/runbooks/streaming-debug.md)
- tests y cobertura: [docs/runbooks/test-and-coverage.md](docs/runbooks/test-and-coverage.md)

## Estado recomendado del laboratorio

- `default rapido`: [docs/gemma-4-26b-a4b.md](docs/gemma-4-26b-a4b.md)
- `default premium`: [docs/gpt-oss-120b.md](docs/gpt-oss-120b.md)
- `comparador Gemma mas grande`: [docs/gemma-4-31b.md](docs/gemma-4-31b.md)
- `two_pass` de referencia para Qwen: [docs/qwen-thinking-integration-plan.md](docs/qwen-thinking-integration-plan.md)

## Validacion rapida

```bash
make helm-lint-values VALUES=<values-file>
make helm-template-reasoning NAMESPACE=<namespace> VALUES=<values-file>
```

## Calidad minima del repo

- `go test ./...` debe pasar
- cobertura unitaria total objetivo: `>= 80%`
- runbook asociado: [docs/runbooks/test-and-coverage.md](docs/runbooks/test-and-coverage.md)
