# Gemma 4 31B

Configuracion preparada para desplegar `google/gemma-4-31B-it` en `4x RTX 3090`.

Archivo:

- [env/prod/gemma-4-31b.yaml](../env/prod/gemma-4-31b.yaml)

Modelo oficial:

- <https://hf.co/google/gemma-4-31B-it>

## Perfil elegido

- hardware objetivo: `4x RTX 3090`
- despliegue: `4 GPU`
- modo del orquestador: `single_pass`
- backend HTTP: `vllm_chat_completions`
- estado en este repo: configuracion preparada para laboratorio

## Decisiones de configuracion

- `structured.model: google/gemma-4-31B-it`
- `tensorParallelSize: 4`
- `maxModelLen: 32768`
- `orchestrator.modelType: gemma4`
- `pass2.provider: vllm_chat_completions`
- `pass2MaxTokens: 4096`
- `pass2Timeout: 90s`

## Notas especificas

### 1. Misma familia operativa que `26B-A4B-it`

La ruta es la misma:

- `structured` con `Gemma 4`
- `orchestrator` en `single_pass`
- `reasoning` desactivado

Eso permite comparar `31B` contra `26B-A4B` sin mezclar arquitectura de integracion.

### 2. Flag multimodal en formato JSON

Igual que en `26B-A4B-it`, para `vLLM v0.19.1` se dejo:

```text
--limit-mm-per-prompt={"image":0,"audio":0}
```

No usar:

```text
--limit-mm-per-prompt=image=0,audio=0
```

## Lo que falta

Este values file esta listo, pero esta ficha no afirma una validacion de laboratorio completa. Si se quiere cerrar de verdad el documento, faltaria:

- smoke test real contra el endpoint
- bateria SWE
- comparativa directa contra `Gemma 4 26B A4B`

## Lectura operativa

`Gemma 4 31B-it` es la opcion natural si `26B-A4B-it` te ha gustado y quieres subir un escalon de calidad sin salirte de la familia `Gemma 4`.

Si el criterio principal sigue siendo `speed/quality`, la referencia actual en este repo sigue siendo:

- [docs/gemma-4-26b-a4b.md](gemma-4-26b-a4b.md)

Si el objetivo pasa a ser calidad dentro de `Gemma 4`, la siguiente comparativa que tiene sentido es:

- `gemma-4-26B-A4B-it` vs `gemma-4-31B-it`
