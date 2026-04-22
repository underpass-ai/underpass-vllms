# GPT-OSS 120B

Configuracion preparada para desplegar `gpt-oss-120b` en una release separada, pensada para `4x RTX 3090`.

Archivo:

- [env/prod/gpt-oss-120b.yaml](/home/tirso/ai/developents/underpass-vllms/env/prod/gpt-oss-120b.yaml)

Suposiciones:

- hardware objetivo: `4x RTX 3090`
- despliegue objetivo: `4 GPU`
- modo del orquestador: `single_pass`
- backend HTTP: `openai_chat_completions`
- esfuerzo de razonamiento: `high`

Componentes activados en esa release:

- `structured`: `openai/gpt-oss-120b` servido con `vLLM`
- `orchestrator`: ruta `gpt_oss` usando `single_pass`

Decisiones:

- `tensorParallelSize: 4`
  `vLLM` documenta `gpt-oss-120b --tensor-parallel-size 4` como configuracion soportada, y OpenAI documenta que el modelo cabe en una `H100` de `80 GB`; en `4x3090` la apuesta razonable es repartirlo entre las cuatro GPUs.
- `pass2ReasoningEffort: high`
  esto alinea el orquestador con el contrato oficial de `reasoning_effort` para `gpt-oss`.
- `pass2MaxTokens: 16384`
  para laboratorio, deja mucho más margen a la cadena de razonamiento antes de emitir el JSON final.
- `pass2Timeout: 900s`
  acompana el presupuesto de tokens para que el orquestador no corte la request antes de tiempo.
- `maxModelLen: 16384`
  es una cota mas prudente para `4x3090`; deja mas margen de VRAM para pesos, KV cache y grafo CUDA.

Release sugerida:

```bash
helm upgrade --install underpass-llm-gpt-oss-120b charts/vllm \
  -n underpass-runtime \
  -f env/prod/gpt-oss-120b.yaml
```

El `baseURL` del orquestador en ese values file asume exactamente esa release name:

```text
underpass-llm-gpt-oss-120b
```

Si cambias el nombre de la release, cambia también:

```yaml
orchestrator.pass2.baseURL
```
