# Runbook: Tests Y Cobertura

Usa este runbook cuando quieras comprobar rapido que el repo sigue sano y que la cobertura unitaria no ha bajado del objetivo.

Objetivo actual:

- suite verde
- cobertura total `>= 80%`

## 1. Ejecutar toda la suite

```bash
go test ./...
```

Si esto falla, no sigas con cobertura. Corrige primero el fallo funcional.

## 2. Medir cobertura total

```bash
go test ./... -coverprofile=/tmp/underpass-vllms.cover
go tool cover -func=/tmp/underpass-vllms.cover | tail -n 1
```

Resultado esperado:

```text
total: (statements) 80.x%
```

## 3. Ver paquetes flojos

```bash
go tool cover -func=/tmp/underpass-vllms.cover | rg 'cmd/two-pass-server|internal/adapters/inbound/httpapi|internal/application/twopass|internal/config'
```

Lectura rapida:

- `internal/application/twopass`: core del caso de uso
- `internal/adapters/inbound/httpapi`: facade HTTP custom + OpenAI-compatible
- `internal/config`: parsing de entorno
- `cmd/two-pass-server`: wiring y bootstrap

## 4. Qué mirar primero si cae la cobertura

Orden recomendado:

1. `internal/application/twopass`
2. `internal/adapters/inbound/httpapi`
3. `internal/config`
4. `cmd/two-pass-server`

Motivo:

- el core y el facade HTTP dan cobertura util
- `main` suele ser lo menos rentable por statement

## 5. Casos faciles que suelen faltar

- validaciones de request vacia o schema invalido
- overrides `single_pass` vs alias legacy `pass2`
- errores de parseo de payload OpenAI
- ramas de streaming:
  - writer sin `Flusher`
  - error al escribir chunk
  - `stream=true` sobre `two_pass`
- errores de `LoadFromEnv`
  - variables requeridas ausentes
  - enteros y duraciones invalidas
  - budgets negativos

## 6. Comando de control rapido

Si solo quieres una comprobacion corta:

```bash
go test ./... && go test ./... -coverprofile=/tmp/underpass-vllms.cover >/dev/null && go tool cover -func=/tmp/underpass-vllms.cover | tail -n 1
```

## 7. Estado de referencia

Estado documentado de esta pasada:

- `go test ./...` verde
- cobertura total: `80.1%`

Si bajas de ahi, no significa necesariamente regresion funcional, pero si que el repo ya no cumple el objetivo de calidad fijado para unit tests.
