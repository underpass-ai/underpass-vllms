# Values Reference

## Regla base

Escribe un values file por release y declara explícitamente todos los campos operativos del componente que actives.

El chart trae un `values.yaml` base, pero no debe tratarse como configuración cerrada de entorno.

## Common Keys

| Key | Uso |
| --- | --- |
| `nameOverride` | override opcional del nombre del chart |
| `fullnameOverride` | override opcional del fullname Helm |
| `cache.*` | cache de modelos para `reasoning` y `structured` |
| `serviceMonitor.*` | `ServiceMonitor` para `reasoning` y `structured` |
| `dns.route53.*` | job opcional para publicar DNS |

## Reasoning Values File

### Campos que debes declarar

| Key | Obligatorio para operar |
| --- | --- |
| `cache.mountPath` | sí |
| `cache.enabled` | sí |
| `cache.size` y `cache.accessMode` | sí, si el chart crea el PVC |
| `cache.existingClaim` | sí, si reutilizas PVC |
| `cache.emptyDirSizeLimit` | sí, si `cache.enabled=false` |
| `reasoning.image.repository` | sí |
| `reasoning.image.tag` | sí |
| `reasoning.image.pullPolicy` | sí |
| `reasoning.model` | sí |
| `reasoning.tensorParallelSize` | sí |
| `reasoning.maxModelLen` | sí |
| `reasoning.maxNumSeqs` | sí |
| `reasoning.gpuMemoryUtilization` | sí |
| `reasoning.extraArgs` | sí, para fijar el rol del endpoint |
| `reasoning.env` | sí, si necesitas variables de runtime |
| `reasoning.huggingface.tokenSecret` | sí, si tu imagen necesita token HF |
| `reasoning.huggingface.tokenKey` | sí, si defines `tokenSecret` |
| `reasoning.shmSizeLimit` | sí |
| `reasoning.resources` | sí |
| `reasoning.probes.*` | sí |
| `reasoning.service.type` | sí |
| `reasoning.service.port` | sí |

### Flags mínimas recomendadas para el rol `reasoning`

Pon estas flags en `reasoning.extraArgs`:

```yaml
reasoning:
  extraArgs:
    - --language-model-only
    - --reasoning-parser=qwen3
```

Si activas `thinking_token_budget` con modelos Qwen/Qwopus servidos por `vLLM`, añade también `--reasoning-config`. Sin eso, el backend devuelve `400`.

Ejemplo:

```yaml
reasoning:
  extraArgs:
    - --language-model-only
    - --reasoning-parser=qwen3
    - '--reasoning-config={"reasoning_start_str":"<think>","reasoning_end_str":"I have to give the solution based on the reasoning directly now.</think>"}'
```

### Example

```yaml
cache:
  enabled: true
  existingClaim: ""
  storageClass: fast-ssd
  size: 100Gi
  accessMode: ReadWriteOnce
  mountPath: /tmp/hf-cache
  emptyDirSizeLimit: 20Gi

reasoning:
  image:
    repository: <reasoning-image>
    tag: <reasoning-tag>
    pullPolicy: IfNotPresent
  model: Qwen/Qwen3.6-35B-A3B
  tensorParallelSize: 4
  maxModelLen: 8192
  maxNumSeqs: 1
  gpuMemoryUtilization: 0.92
  enforceEager: false
  extraArgs:
    - --language-model-only
    - --reasoning-parser=qwen3
  env:
    HF_HOME: /tmp/hf-cache
    LD_LIBRARY_PATH: /usr/lib
    VLLM_ATTENTION_BACKEND: FLASHINFER
  huggingface:
    tokenSecret: huggingface-token
    tokenKey: HF_TOKEN
  shmSizeLimit: 2Gi
  resources:
    requests:
      cpu: "8"
      memory: 64Gi
      nvidia.com/gpu: "4"
    limits:
      cpu: "16"
      memory: 96Gi
      nvidia.com/gpu: "4"
  probes:
    startup:
      periodSeconds: 10
      failureThreshold: 180
    readiness:
      periodSeconds: 10
      failureThreshold: 3
    liveness:
      periodSeconds: 30
      failureThreshold: 3
  service:
    type: ClusterIP
    port: 8000
  ingress:
    enabled: false
    className: nginx
    host: ""
    tls:
      enabled: false
      secretName: ""
      clusterIssuer: ""
    mtls:
      enabled: false
      clientCaSecret: ""
```

## Structured Values File

### Campos que debes declarar

`structured` usa la misma estructura que `reasoning`:

- `structured.image.*`
- `structured.model`
- `structured.tensorParallelSize`
- `structured.maxModelLen`
- `structured.maxNumSeqs`
- `structured.gpuMemoryUtilization`
- `structured.extraArgs`
- `structured.env`
- `structured.huggingface.*`
- `structured.shmSizeLimit`
- `structured.resources`
- `structured.probes.*`
- `structured.service.*`
- `structured.lora.*`
- `structured.adapterVolume.*`

### LoRA serving

`reasoning` y `structured` pueden servir adapters LoRA mediante vLLM. Los
campos son opcionales y por defecto están desactivados:

```yaml
structured:
  lora:
    enabled: true
    maxRank: 16
    adapters:
      - name: operator-v8.1.2
        path: /adapters/operator-v8.1.2-sft-v2
  adapterVolume:
    enabled: true
    mountPath: /adapters/operator-v8.1.2-sft-v2
    hostPath: /var/lib/operator-adapters/v8.1.2-sft-v2-canonical
    hostPathType: Directory
```

When enabled, the chart adds:

- `--enable-lora`
- `--max-lora-rank=<maxRank>`
- `--lora-modules <name>=<path>`
- a read-only hostPath mount for the adapter directory

Use this only for adapter directories that already exist on the node. The chart
does not download or verify adapter artifacts.

### Flags mínimas recomendadas para el rol `structured`

Pon estas flags en `structured.extraArgs`:

```yaml
structured:
  extraArgs:
    - --language-model-only
    - '--default-chat-template-kwargs={"enable_thinking": false}'
```

No añadas reasoning parser aquí.

### Example

```yaml
cache:
  enabled: true
  existingClaim: ""
  storageClass: fast-ssd
  size: 50Gi
  accessMode: ReadWriteOnce
  mountPath: /tmp/hf-cache
  emptyDirSizeLimit: 20Gi

structured:
  image:
    repository: <structured-image>
    tag: <structured-tag>
    pullPolicy: IfNotPresent
  model: Qwen/Qwen3.6-35B-A3B
  tensorParallelSize: 4
  maxModelLen: 4096
  maxNumSeqs: 1
  gpuMemoryUtilization: 0.92
  enforceEager: false
  extraArgs:
    - --language-model-only
    - '--default-chat-template-kwargs={"enable_thinking": false}'
  env:
    HF_HOME: /tmp/hf-cache
    LD_LIBRARY_PATH: /usr/lib
  huggingface:
    tokenSecret: huggingface-token
    tokenKey: HF_TOKEN
  shmSizeLimit: 2Gi
  resources:
    requests:
      cpu: "8"
      memory: 64Gi
      nvidia.com/gpu: "4"
    limits:
      cpu: "16"
      memory: 96Gi
      nvidia.com/gpu: "4"
  probes:
    startup:
      periodSeconds: 10
      failureThreshold: 120
    readiness:
      periodSeconds: 10
      failureThreshold: 3
    liveness:
      periodSeconds: 30
      failureThreshold: 3
  service:
    type: ClusterIP
    port: 8000
  ingress:
    enabled: false
    className: nginx
    host: ""
    tls:
      enabled: false
      secretName: ""
      clusterIssuer: ""
    mtls:
      enabled: false
      clientCaSecret: ""
```

## Orchestrator Values File

Declara explícitamente todos los campos de `orchestrator.config` que afecten al comportamiento del modelo.

El binario no debe actuar como fuente de verdad para prompts, sampling, budgets o timeouts. Si falta uno de esos valores, el proceso debe fallar al arrancar.

La selección del adapter ya no se hace dentro del caso de uso. El composition root elige adapter por `orchestrator.modelType`:

- `qwen_reasoning` -> `TwoPassAdapter`
- `gpt_oss` -> `SinglePassAdapter`

### Campos que debes declarar

| Key | Uso |
| --- | --- |
| `orchestrator.replicaCount` | réplicas del deployment |
| `orchestrator.image.repository` | imagen |
| `orchestrator.image.tag` | tag |
| `orchestrator.image.pullPolicy` | pull policy |
| `orchestrator.addr` | dirección HTTP del server |
| `orchestrator.modelType` | tipo de modelo que decide el adapter |
| `orchestrator.pass1.provider` | perfil HTTP del endpoint Pass 1 |
| `orchestrator.pass1.baseURL` | base URL del endpoint Pass 1 |
| `orchestrator.pass1.model` | modelo de Pass 1 |
| `orchestrator.pass1.apiKey` | bearer token de Pass 1 |
| `orchestrator.pass2.provider` | perfil HTTP del endpoint Pass 2 |
| `orchestrator.pass2.baseURL` | base URL del endpoint Pass 2 |
| `orchestrator.pass2.model` | modelo de Pass 2 |
| `orchestrator.pass2.apiKey` | bearer token de Pass 2 |
| `orchestrator.config.maxIntermediateBytes` | límite del IR |
| `orchestrator.config.pass2RetryCount` | reintentos de Pass 2 |
| `orchestrator.config.pass1PromptVersion` | versión lógica del prompt de Pass 1 |
| `orchestrator.config.pass2PromptVersion` | versión lógica del prompt de Pass 2 |
| `orchestrator.config.singlePassPromptVersion` | versión lógica del prompt single-pass |
| `orchestrator.config.irVersion` | versión lógica del IR |
| `orchestrator.config.pass1SystemPrompt` | system prompt de Pass 1 |
| `orchestrator.config.pass2SystemPrompt` | system prompt de Pass 2 |
| `orchestrator.config.singlePassSystemPrompt` | system prompt single-pass |
| `orchestrator.config.pass1UserPromptTemplate` | plantilla de prompt de usuario para Pass 1 |
| `orchestrator.config.pass2UserPromptTemplate` | plantilla de prompt de usuario para Pass 2 |
| `orchestrator.config.singlePassUserPromptTemplate` | plantilla de prompt de usuario single-pass |
| `orchestrator.config.pass2RetryHintTemplate` | plantilla del hint de reintento de Pass 2 |
| `orchestrator.config.pass1Temperature` | temperatura Pass 1 |
| `orchestrator.config.pass1TopP` | `top_p` de Pass 1 |
| `orchestrator.config.pass1TopK` | `top_k` de Pass 1 |
| `orchestrator.config.pass1PresencePenalty` | `presence_penalty` de Pass 1 |
| `orchestrator.config.pass1RepetitionPenalty` | `repetition_penalty` de Pass 1 |
| `orchestrator.config.pass1ReasoningEffort` | `reasoning_effort` de Pass 1 si el backend lo soporta |
| `orchestrator.config.pass2Temperature` | temperatura Pass 2 |
| `orchestrator.config.pass2TopP` | `top_p` de Pass 2 |
| `orchestrator.config.pass2TopK` | `top_k` de Pass 2 |
| `orchestrator.config.pass2PresencePenalty` | `presence_penalty` de Pass 2 |
| `orchestrator.config.pass2RepetitionPenalty` | `repetition_penalty` de Pass 2 |
| `orchestrator.config.pass2ReasoningEffort` | `reasoning_effort` de Pass 2 si el backend lo soporta |
| `orchestrator.config.pass1MaxTokens` | max tokens Pass 1 |
| `orchestrator.config.pass2MaxTokens` | max tokens Pass 2 |
| `orchestrator.config.pass1ThinkingTokenBudget` | presupuesto de thinking Pass 1 |
| `orchestrator.config.pass2ThinkingTokenBudget` | presupuesto de thinking Pass 2 |
| `orchestrator.config.pass1PreserveThinking` | activa `preserve_thinking` en Pass 1 |
| `orchestrator.config.pass2PreserveThinking` | activa `preserve_thinking` en Pass 2 |
| `orchestrator.config.pass1Timeout` | timeout Pass 1 |
| `orchestrator.config.pass2Timeout` | timeout Pass 2 |
| `orchestrator.resources` | recursos |
| `orchestrator.service.type` | tipo de service |
| `orchestrator.service.port` | puerto HTTP |

### Campos por `modelType`

- `qwen_reasoning` requiere `pass1.*`, `pass1PromptVersion`, `pass2PromptVersion`, prompts de Pass 1 y Pass 2, y los knobs de Pass 1 y Pass 2.
- `gpt_oss` no usa `pass1.*`; requiere `singlePassPromptVersion`, `singlePassSystemPrompt`, `singlePassUserPromptTemplate` y `pass2.*` como backend único.
- `gemma4` usa el mismo contrato que `gpt_oss`: no usa `pass1.*`; requiere `singlePassPromptVersion`, `singlePassSystemPrompt`, `singlePassUserPromptTemplate` y `pass2.*` como backend único.

### Nota sobre presupuestos y contexto

No fijes `pass1MaxTokens` igual a `reasoning.maxModelLen` si el modelo recibe prompt no vacío. `vLLM` lo rechazará porque no queda hueco para tokens de entrada.

Regla práctica:

- `pass1MaxTokens + prompt_tokens` debe quedar claramente por debajo de `maxModelLen`
- si además usas `pass1ThinkingTokenBudget`, mantenlo también por debajo de `pass1MaxTokens`

En la práctica, para un `maxModelLen: 16384`, una cota razonable de laboratorio es:

```yaml
orchestrator:
  config:
    pass1MaxTokens: 8192
    pass1ThinkingTokenBudget: 4096
```

### Nota sobre `baseURL`

`baseURL` debe apuntar al root OpenAI-compatible, por ejemplo:

```text
http://underpass-llm-reasoning-reasoning.<namespace>.svc.cluster.local:8000/v1
```

El cliente añade `/chat/completions` internamente.

Si el upstream no valida bearer token, declara igualmente `apiKey: EMPTY` de forma explícita. No dejes el campo implícito.

### Example

```yaml
orchestrator:
  replicaCount: 1
  image:
    repository: <orchestrator-image>
    tag: <orchestrator-tag>
    pullPolicy: IfNotPresent
  addr: :8080
  modelType: qwen_reasoning
  pass1:
    provider: vllm_chat_completions
    baseURL: http://underpass-llm-reasoning-reasoning.underpass-runtime.svc.cluster.local:8000/v1
    model: Qwen/Qwen3.6-35B-A3B
    apiKey: EMPTY
  pass2:
    provider: vllm_chat_completions
    baseURL: http://underpass-llm-structured-structured.underpass-runtime.svc.cluster.local:8000/v1
    model: Qwen/Qwen3.6-35B-A3B
    apiKey: EMPTY
  config:
    maxIntermediateBytes: 65536
    pass2RetryCount: 1
    pass1PromptVersion: 2026-04-21.2
    pass2PromptVersion: 2026-04-21.1
    irVersion: 1.0.0
    pass1SystemPrompt: |-
      ...
    pass2SystemPrompt: |-
      ...
    pass1UserPromptTemplate: |-
      ...
    pass2UserPromptTemplate: |-
      ...
    pass2RetryHintTemplate: |-
      ...
    pass1Temperature: "0.6"
    pass1TopP: "0.95"
    pass1TopK: "20"
    pass1PresencePenalty: "0"
    pass1RepetitionPenalty: "1.0"
    pass2Temperature: "0"
    pass1MaxTokens: 4096
    pass2MaxTokens: 4096
    pass1ThinkingTokenBudget: 2048
    pass1PreserveThinking: "true"
    pass1Timeout: 45s
    pass2Timeout: 45s
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: false
    className: nginx
    host: ""
    tls:
      enabled: false
      secretName: ""
      clusterIssuer: ""
    mtls:
      enabled: false
      clientCaSecret: ""
```

## Ingress, TLS y mTLS

La misma forma aplica a `reasoning`, `structured` y `orchestrator`.

### Ingress mínimo

```yaml
<component>:
  ingress:
    enabled: true
    className: nginx
    host: <fqdn>
```

### TLS con cert-manager

```yaml
<component>:
  ingress:
    enabled: true
    className: nginx
    host: <fqdn>
    tls:
      enabled: true
      secretName: <tls-secret>
      clusterIssuer: <cluster-issuer>
```

### mTLS en NGINX

```yaml
<component>:
  ingress:
    mtls:
      enabled: true
      clientCaSecret: <ca-secret>
```

## Route53

Si activas `dns.route53.enabled=true`, declara todos estos campos:

```yaml
dns:
  route53:
    enabled: true
    target: <ingress-ip-or-lb-address>
    ttl: 300
    region: <aws-region>
    credentialsSecret: <route53-secret>
    accessKeyIdKey: <access-key-field>
    secretAccessKeyKey: <secret-key-field>
    hostedZoneIdKey: <hosted-zone-id-field>
    image:
      repository: public.ecr.aws/aws-cli/aws-cli
      tag: "2.17.47"
      pullPolicy: IfNotPresent
```

## ServiceMonitor

```yaml
serviceMonitor:
  enabled: true
  labels:
    release: kube-prometheus-stack
  interval: 30s
  path: /metrics
```

Aplica solo a `reasoning` y `structured`.
