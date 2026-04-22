# GPT-OSS 20B

Configuracion preparada para desplegar `gpt-oss-20b` en una release separada, sin tocar el stack Qwen.

Archivo:

- [env/prod/gpt-oss-20b.yaml](/home/tirso/ai/developents/underpass-vllms/env/prod/gpt-oss-20b.yaml)

Suposiciones:

- hardware objetivo: `4x RTX 3090`
- despliegue inicial conservador: `1 GPU`
- modo del orquestador: `single_pass`
- backend HTTP: `openai_chat_completions`
- esfuerzo de razonamiento: `high`

Componentes activados en esa release:

- `structured`: `openai/gpt-oss-20b` servido con `vLLM`
- `orchestrator`: ruta `gpt_oss` usando `single_pass`

Decisiones:

- `tensorParallelSize: 1`
  `gpt-oss-20b` entra de sobra en una sola `3090`, así que el primer despliegue prioriza simplicidad y deja libres las otras GPUs.
- `pass2ReasoningEffort: high`
  esto alinea el orquestador con el contrato oficial de `reasoning_effort` para `gpt-oss`.
- `pass2MaxTokens: 16384`
  para laboratorio, deja mucho más margen a la cadena de razonamiento antes de emitir el JSON final.
- `pass2Timeout: 600s`
  acompana el presupuesto de tokens para que el orquestador no corte la request antes de tiempo.
- `maxModelLen: 32768`
  es un valor prudente para arrancar en consumer GPUs; se puede subir después si la latencia y la memoria real lo permiten.

Release sugerida:

```bash
helm upgrade --install underpass-llm-gpt-oss-20b charts/vllm \
  -n underpass-runtime \
  -f env/prod/gpt-oss-20b.yaml
```

El `baseURL` del orquestador en ese values file asume exactamente esa release name:

```text
underpass-llm-gpt-oss-20b
```

Si cambias el nombre de la release, cambia también:

```yaml
orchestrator.pass2.baseURL
```
