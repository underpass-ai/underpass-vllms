# Deployment Guide

## Alcance

Este repositorio despliega tres roles logicos:

- `reasoning`: endpoint vLLM para extracción semántica.
- `structured`: endpoint vLLM para canonicalización JSON.
- `orchestrator`: servicio HTTP en Go que coordina los dos pasos.

El chart es unico, pero la operacion real del proyecto admite dos patrones:

- `two_pass` por componente: tres releases separadas
- `single_pass` por perfil de modelo: una release con `structured + orchestrator`

## Requisitos mínimos

- Kubernetes accesible con `kubectl`.
- Helm 3.
- `make`.
- Scheduling GPU para las releases `reasoning` y `structured`.
- Reachability desde `orchestrator` hasta dos endpoints OpenAI-compatible.
- Ingress NGINX solo si vas a exponer `*.ingress.enabled=true`.
- cert-manager solo si vas a activar `*.ingress.tls.enabled=true`.
- Secret de credenciales AWS Route53 solo si vas a activar `dns.route53.enabled=true`.

La documentación no fija hardware concreto. CPU, memoria, GPU y storage se declaran en cada values file.

## Modelos de release

### Patron `two_pass`

Usa una release distinta por servicio:

- `underpass-llm-reasoning`
- `underpass-llm-structured`
- `underpass-llm-orchestrator`

Los targets `make` ya vienen preparados para ese modelo y fuerzan por `--set` que solo quede activo el componente correspondiente.

### Patron `single_pass`

Usa una unica release por perfil de modelo. Ejemplos que ya se han usado en este repo:

- `underpass-llm-gemma-4-26b-a4b`
- `underpass-llm-gemma-4-31b`
- `underpass-llm-gpt-oss-120b`

En este patron:

- `reasoning.enabled=false`
- `structured.enabled=true`
- `orchestrator.enabled=true`
- `orchestrator.modelType` decide el adapter (`gemma4` o `gpt_oss`)

## Nombres de recursos

### Patron `two_pass`

Con la convencion anterior, los nombres renderizados quedan asi:

- release `underpass-llm-reasoning`:
  - Deployment/Service/Ingress/ServiceMonitor: `underpass-llm-reasoning-reasoning`
  - PVC de cache si lo crea el chart: `underpass-llm-reasoning-hf-cache`
- release `underpass-llm-structured`:
  - Deployment/Service/Ingress/ServiceMonitor: `underpass-llm-structured-structured`
  - PVC de cache si lo crea el chart: `underpass-llm-structured-hf-cache`
- release `underpass-llm-orchestrator`:
  - Deployment/Service/Ingress: `underpass-llm-orchestrator-orchestrator`

### Patron `single_pass`

Con una release empaquetada, los nombres quedan por componente activo dentro de la misma release.

Ejemplo real:

- release `underpass-llm-gemma-4-31b`:
  - Deployment/Service `structured`: `underpass-llm-gemma-4-31b-structured`
  - Deployment/Service `orchestrator`: `underpass-llm-gemma-4-31b-orchestrator`
  - PVC de cache si lo crea el chart: `underpass-llm-gemma-4-31b-hf-cache`

## Workflow exacto

Si quieres algo corto y operativo antes de entrar en detalle, usa:

- [runbooks/single-pass-release.md](runbooks/single-pass-release.md)
- [runbooks/streaming-debug.md](runbooks/streaming-debug.md)

### 1. Elegir el patron de despliegue

- usa `two_pass` si el modelo necesita `reasoning` separado
- usa `single_pass` si el modelo devuelve JSON estructurado en una sola llamada

### 2. Crear un values file por release

No reutilices `charts/vllm/values.yaml` como fichero de entorno. Crea un values file propio para cada release.

Referencias:

- [docs/values-reference.md](values-reference.md)

### 3. Validar el values file

```bash
make helm-lint-values VALUES=env/components/reasoning.yaml
```

### 4. Renderizar antes de instalar

```bash
make helm-template-reasoning NAMESPACE=underpass-runtime VALUES=env/components/reasoning.yaml
make helm-template-structured NAMESPACE=underpass-runtime VALUES=env/components/structured.yaml
make helm-template-orchestrator NAMESPACE=underpass-runtime VALUES=env/components/orchestrator.yaml
```

Para una release `single_pass`, usa `helm template` directamente sobre el values file del perfil:

```bash
helm template underpass-llm-gemma-4-31b charts/vllm -n underpass-runtime -f env/prod/gemma-4-31b.yaml
```

### 5. Instalar o actualizar

```bash
make helm-upgrade-reasoning NAMESPACE=underpass-runtime VALUES=env/components/reasoning.yaml
make helm-upgrade-structured NAMESPACE=underpass-runtime VALUES=env/components/structured.yaml
make helm-upgrade-orchestrator NAMESPACE=underpass-runtime VALUES=env/components/orchestrator.yaml
```

Para una release `single_pass`, usa:

```bash
helm upgrade --install underpass-llm-gemma-4-31b charts/vllm \
  -n underpass-runtime \
  -f env/prod/gemma-4-31b.yaml
```

### 6. Desinstalar

```bash
make helm-uninstall-reasoning NAMESPACE=underpass-runtime
make helm-uninstall-structured NAMESPACE=underpass-runtime
make helm-uninstall-orchestrator NAMESPACE=underpass-runtime
```

Para una release `single_pass`:

```bash
helm uninstall underpass-llm-gemma-4-31b -n underpass-runtime
```

## Que renderiza cada componente

### `reasoning`

Cuando `reasoning.enabled=true`, el chart puede renderizar:

- `Deployment`
- `Service`
- `Ingress`, si `reasoning.ingress.enabled=true`
- `Certificate`, si además `reasoning.ingress.tls.enabled=true`
- `ServiceMonitor`, si `serviceMonitor.enabled=true`
- `Job` de Route53, si `dns.route53.enabled=true` y `reasoning.ingress.enabled=true`
- `PersistentVolumeClaim`, si `cache.enabled=true` y `cache.existingClaim=""`

### `structured`

Cuando `structured.enabled=true`, el chart puede renderizar:

- `Deployment`
- `Service`
- `Ingress`, si `structured.ingress.enabled=true`
- `Certificate`, si además `structured.ingress.tls.enabled=true`
- `ServiceMonitor`, si `serviceMonitor.enabled=true`
- `Job` de Route53, si `dns.route53.enabled=true` y `structured.ingress.enabled=true`
- `PersistentVolumeClaim`, si `cache.enabled=true` y `cache.existingClaim=""`

### `orchestrator`

Cuando `orchestrator.enabled=true`, el chart puede renderizar:

- `Deployment`
- `Service`
- `Ingress`, si `orchestrator.ingress.enabled=true`
- `Certificate`, si además `orchestrator.ingress.tls.enabled=true`
- `Job` de Route53, si `dns.route53.enabled=true` y `orchestrator.ingress.enabled=true`

El chart no crea `ServiceMonitor` para `orchestrator`.

## Semantica operativa de `reasoning` y `structured`

El chart no diferencia automáticamente ambos roles a nivel de flags de vLLM. Esa distinción la defines tú con `extraArgs`.

Convencion del proyecto:

- `reasoning.extraArgs` debe incluir al menos:
  - `--language-model-only`
  - `--reasoning-parser=qwen3`
- `structured.extraArgs` debe incluir al menos:
  - `--language-model-only`
  - `'--default-chat-template-kwargs={"enable_thinking": false}'`

No metas reasoning parser en `structured`.

El segundo flag contiene `:` dentro del JSON. En YAML debe ir entre comillas simples para que siga siendo un string único.

`structured_outputs` no se configura en el chart de vLLM. Lo envía el orquestador por petición a Pass 2.

Para `gemma4`, `structured` puede ir en `single_pass` con `--reasoning-parser=gemma4` y sin componente `reasoning` separado. En ese perfil, la canonicalizacion JSON la hace la unica llamada del backend estructurado y la validacion final queda en el orquestador.

## TLS

Si `*.ingress.enabled=true`, el chart crea un `Ingress`.

Si además `*.ingress.tls.enabled=true`, el chart:

- añade la sección `tls` al `Ingress`,
- crea un `Certificate` de cert-manager,
- usa `*.ingress.tls.secretName` como secret final del certificado,
- usa `*.ingress.tls.clusterIssuer` como `ClusterIssuer`.

Si `*.ingress.mtls.enabled=true`, el chart añade las anotaciones NGINX:

- `nginx.ingress.kubernetes.io/auth-tls-verify-client: "on"`
- `nginx.ingress.kubernetes.io/auth-tls-secret: <namespace>/<clientCaSecret>`
- `nginx.ingress.kubernetes.io/auth-tls-verify-depth: "1"`

## DNS

Si `dns.route53.enabled=true`, el chart crea un `Job` post-install/post-upgrade por cada componente activo que además tenga `ingress.enabled=true`.

Ese job hace un `UPSERT` de un registro `A`:

- nombre: `*.ingress.host`
- valor: `dns.route53.target`
- TTL: `dns.route53.ttl`

Necesitas un secret con estas claves:

- `dns.route53.accessKeyIdKey`
- `dns.route53.secretAccessKeyKey`
- `dns.route53.hostedZoneIdKey`

El nombre del secret va en `dns.route53.credentialsSecret`.

## Cache

`reasoning` y `structured` montan un volumen `hf-cache` en `cache.mountPath`.

Opciones:

- PVC creado por el chart:
  - `cache.enabled=true`
  - `cache.existingClaim=""`
- PVC existente:
  - `cache.enabled=true`
  - `cache.existingClaim=<claim>`
- `emptyDir`:
  - `cache.enabled=false`
  - `cache.emptyDirSizeLimit=<size>`

Si quieres compartir cache entre releases distintas, usa `cache.existingClaim` y asegúrate de que tu storage soporta el patrón de montaje que necesitas.

## Observabilidad

Para seguir logs en tiempo real del orquestador y comprobar streaming, no improvises:

- usa [runbooks/streaming-debug.md](runbooks/streaming-debug.md)

Regla operativa importante:

- valida streaming primero desde dentro del cluster
- si dentro del cluster ves deltas y fuera no, el problema suele estar en el cliente, proxy o ingress

Si `serviceMonitor.enabled=true`, el chart crea `ServiceMonitor` para `reasoning` y `structured`.

Campos usados:

- `serviceMonitor.labels`
- `serviceMonitor.interval`
- `serviceMonitor.path`

En este cluster, la etiqueta que usa Prometheus Operator para descubrir `ServiceMonitor` es:

- `serviceMonitor.labels.release=kube-prometheus-stack`

## E2E del orquestador

El job [deploy/kubernetes/e2e-job.yaml](../deploy/kubernetes/e2e-job.yaml) valida:

- `/healthz`
- `/v1/two-pass/structured`
- presencia de `request_id`, `output` e `invoice_number` en la respuesta

Por defecto apunta a:

```text
http://two-pass-server.default.svc.cluster.local:8080
```

Antes de usarlo en un entorno real, ajusta `TWO_PASS_SERVER_URL` al `Service` o `Ingress` real del orquestador desplegado.

## Estado documentado hoy

- perfil `two_pass` de referencia: `env/components/reasoning.yaml`, `env/components/structured.yaml`, `env/components/orchestrator.yaml`
- perfil `single_pass` activo de production: `env/prod/gemma-4-31b.yaml`
- contrato HTTP actual: [docs/api.md](api.md)
