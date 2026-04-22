# Runbooks

Runbooks cortos para operar el repo sin leer toda la documentacion general.

## Orden recomendado

1. [single-pass-release.md](single-pass-release.md)
2. [openai-consumer-smoke.md](openai-consumer-smoke.md)
3. [streaming-debug.md](streaming-debug.md)
4. [test-and-coverage.md](test-and-coverage.md)

## Cuando usar cada uno

- `single-pass-release.md`
  - desplegar o actualizar una release tipo `Gemma` o `GPT-OSS`
  - validar que el release ha quedado sano
  - hacer smoke tests basicos
  - rollback rapido si algo sale mal

- `streaming-debug.md`
  - `stream=true` no devuelve deltas
  - parece que todo llega de golpe
  - quieres ver logs en tiempo real del orquestador
  - quieres distinguir si el problema esta en el orquestador o en un proxy/cliente

- `openai-consumer-smoke.md`
  - quieres validar el facade OpenAI-compatible desde fuera
  - necesitas ejemplos minimos de `chat.completions` y `responses`
  - quieres comprobar el shape de errores y streaming

- `test-and-coverage.md`
  - quieres validar rapido que el repo sigue sano
  - necesitas medir cobertura total
  - quieres saber donde atacar si cae por debajo de `80%`
