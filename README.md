# underpass-vllms

`underpass-vllms` contiene el stack de dos pasos para structured output en Underpass AI:

- un endpoint `reasoning` servido con vLLM,
- un endpoint `structured` servido con vLLM,
- un orquestador en Go que llama a ambos endpoints y valida el JSON final.

## Qué hay en el repo

- [charts/vllm](charts/vllm): chart Helm único para desplegar `reasoning`, `structured` y `orchestrator`.
- [cmd/two-pass-server](cmd/two-pass-server): binario principal del orquestador.
- [deploy/kubernetes/e2e-job.yaml](deploy/kubernetes/e2e-job.yaml): job e2e contra el orquestador.
- [Makefile](Makefile): entrada operativa para desplegar cada servicio por separado.

## Orden de lectura

1. [docs/deployment.md](docs/deployment.md)
2. [docs/values-reference.md](docs/values-reference.md)
3. [docs/api.md](docs/api.md)
4. [docs/model-lab-notes-2026-04-21.md](docs/model-lab-notes-2026-04-21.md)
5. [docs/gemma-4-family.md](docs/gemma-4-family.md)
6. [docs/gemma-4-26b-a4b.md](docs/gemma-4-26b-a4b.md)
7. [docs/gemma-4-31b.md](docs/gemma-4-31b.md)
8. [docs/qwopus3.5-27b.md](docs/qwopus3.5-27b.md)
9. [docs/gpt-oss-120b.md](docs/gpt-oss-120b.md)
10. [docs/gpt-oss-20b.md](docs/gpt-oss-20b.md)
11. [docs/qwen-thinking-integration-plan.md](docs/qwen-thinking-integration-plan.md)
12. [docs/qwen-thinking-execution-checklist.md](docs/qwen-thinking-execution-checklist.md)

## Convención de despliegue

La convención del proyecto es desplegar un servicio por release Helm:

- `reasoning`
- `structured`
- `orchestrator`

Los targets `make` fuerzan esa separación:

```bash
make helm-upgrade-reasoning NAMESPACE=<namespace> VALUES=<values-file>
make helm-upgrade-structured NAMESPACE=<namespace> VALUES=<values-file>
make helm-upgrade-orchestrator NAMESPACE=<namespace> VALUES=<values-file>
```

No uses `charts/vllm/values.yaml` como values de entorno. Ese archivo es la base del chart, no una configuración de despliegue cerrada.

## Flujo funcional

1. `reasoning` recibe la entrada y produce una representación intermedia.
2. `structured` transforma esa representación en JSON estricto.
3. `orchestrator` valida el JSON contra el schema recibido en la petición.

La diferencia entre `reasoning` y `structured` no la “adivina” el chart. Debes declarar explícitamente las flags de vLLM adecuadas en `extraArgs`.

## Validación rápida

```bash
make helm-lint-values VALUES=<values-file>
make helm-template-reasoning NAMESPACE=<namespace> VALUES=<values-file>
```
