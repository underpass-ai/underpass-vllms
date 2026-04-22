# Deployment Guide

## Alcance

Este repositorio despliega tres roles lógicos:

- `reasoning`: endpoint vLLM para extracción semántica.
- `structured`: endpoint vLLM para canonicalización JSON.
- `orchestrator`: servicio HTTP en Go que coordina los dos pasos.

El chart es único, pero la operación del proyecto asume una release por servicio.

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

## Modelo de releases

Usa una release distinta por servicio:

- `underpass-llm-reasoning`
- `underpass-llm-structured`
- `underpass-llm-orchestrator`

Los targets `make` ya vienen preparados para ese modelo y fuerzan por `--set` que solo quede activo el componente correspondiente.

## Nombres de recursos

Con la convención anterior, los nombres renderizados quedan así:

- release `underpass-llm-reasoning`:
  - Deployment/Service/Ingress/ServiceMonitor: `underpass-llm-reasoning-reasoning`
  - PVC de cache si lo crea el chart: `underpass-llm-reasoning-hf-cache`
- release `underpass-llm-structured`:
  - Deployment/Service/Ingress/ServiceMonitor: `underpass-llm-structured-structured`
  - PVC de cache si lo crea el chart: `underpass-llm-structured-hf-cache`
- release `underpass-llm-orchestrator`:
  - Deployment/Service/Ingress: `underpass-llm-orchestrator-orchestrator`

## Workflow exacto

### 1. Crear un values file por servicio

No reutilices `charts/vllm/values.yaml` como fichero de entorno. Crea un values file propio para cada release.

Referencias:

- [docs/values-reference.md](values-reference.md)

### 2. Validar el values file

```bash
make helm-lint-values VALUES=env/prod/reasoning.yaml
```

### 3. Renderizar antes de instalar

```bash
make helm-template-reasoning NAMESPACE=underpass-runtime VALUES=env/prod/reasoning.yaml
make helm-template-structured NAMESPACE=underpass-runtime VALUES=env/prod/structured.yaml
make helm-template-orchestrator NAMESPACE=underpass-runtime VALUES=env/prod/orchestrator.yaml
```

### 4. Instalar o actualizar

```bash
make helm-upgrade-reasoning NAMESPACE=underpass-runtime VALUES=env/prod/reasoning.yaml
make helm-upgrade-structured NAMESPACE=underpass-runtime VALUES=env/prod/structured.yaml
make helm-upgrade-orchestrator NAMESPACE=underpass-runtime VALUES=env/prod/orchestrator.yaml
```

### 5. Desinstalar

```bash
make helm-uninstall-reasoning NAMESPACE=underpass-runtime
make helm-uninstall-structured NAMESPACE=underpass-runtime
make helm-uninstall-orchestrator NAMESPACE=underpass-runtime
```

## Qué renderiza cada componente

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

## Semántica operativa de `reasoning` y `structured`

El chart no diferencia automáticamente ambos roles a nivel de flags de vLLM. Esa distinción la defines tú con `extraArgs`.

Convención del proyecto:

- `reasoning.extraArgs` debe incluir al menos:
  - `--language-model-only`
  - `--reasoning-parser=qwen3`
- `structured.extraArgs` debe incluir al menos:
  - `--language-model-only`
  - `'--default-chat-template-kwargs={"enable_thinking": false}'`

No metas reasoning parser en `structured`.

El segundo flag contiene `:` dentro del JSON. En YAML debe ir entre comillas simples para que siga siendo un string único.

`structured_outputs` no se configura en el chart de vLLM. Lo envía el orquestador por petición a Pass 2.

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

Si `serviceMonitor.enabled=true`, el chart crea `ServiceMonitor` para `reasoning` y `structured`.

Campos usados:

- `serviceMonitor.labels`
- `serviceMonitor.interval`
- `serviceMonitor.path`

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
