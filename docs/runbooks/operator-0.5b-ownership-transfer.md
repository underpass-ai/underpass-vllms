# Runbook: Transfer `0.5b.llm.underpassai.com` to Operator

**Status**: pending execution after merge of
`feature/operator-qwen05-lora-serving`.

## Context

`0.5b.llm.underpassai.com` was historically claimed by the live
`underpass-runtime-vllm` ingress and routed to the Gemma structured service as
preparatory infrastructure while the operator model was not ready.

The `underpass-vllms` Gemma production values now render only
`vllm.underpassai.com`. The new operator values render
`0.5b.llm.underpassai.com` for the Qwen 2.5 0.5B + LoRA v8.1.2 endpoint.

The transfer is therefore an operational ingress ownership change:

- Gemma rendered profile: does not claim `0.5b.llm.underpassai.com`
- Operator rendered profile: claims `0.5b.llm.underpassai.com`
- Live legacy ingress: must release `0.5b.llm.underpassai.com` before operator
  install

## Pre-requisites

- Kubernetes cluster access with `helm` and `kubectl` configured for the
  `underpass-runtime` namespace.
- Adapter installed at canonical location:
  `/var/lib/operator-adapters/v8.1.2-sft-v2-canonical/`.
- Branch with the operator deployment values merged to `main`.
- `env/prod/operator-qwen05-v812.yaml` renders an ingress for
  `0.5b.llm.underpassai.com`.
- `env/prod/gemma-4-31b.yaml` does not render that host.

Install or refresh the adapter:

```bash
sudo bash scripts/install-operator-adapter.sh
```

## Pre-flight validation

Render both profiles and confirm host ownership is disjoint:

```bash
helm template gemma charts/vllm -f env/prod/gemma-4-31b.yaml > /tmp/gemma-rendered.yaml
helm template operator charts/vllm -f env/prod/operator-qwen05-v812.yaml > /tmp/operator-rendered.yaml

grep -A 30 "kind: Ingress" /tmp/gemma-rendered.yaml | grep "host:" | sort -u
grep -A 30 "kind: Ingress" /tmp/operator-rendered.yaml | grep "host:" | sort -u
```

Expected:

```text
Gemma:   host: "vllm.underpassai.com"
Operator: host: "0.5b.llm.underpassai.com"
```

Inspect live ownership before the transfer:

```bash
kubectl -n underpass-runtime get ingress underpass-runtime-vllm -o yaml | \
  grep -A 8 "0.5b.llm.underpassai.com"
```

If the host is still listed in `underpass-runtime-vllm`, continue with the
transfer steps below.

## Required order of operations

Execute these steps in order. Steps 1-3 should run within minutes of each
other to minimize the host-unclaimed window.

### Step 1: Legacy ingress releases the host

Remove only `0.5b.llm.underpassai.com` from the live legacy ingress
`underpass-runtime-vllm`. Prefer updating the Helm values for the release that
owns that ingress if available. If the legacy release cannot be updated
immediately, use a temporary kubectl edit during the transfer window:

```bash
kubectl -n underpass-runtime edit ingress underpass-runtime-vllm
```

Remove `0.5b.llm.underpassai.com` from:

- `spec.rules[]`
- `spec.tls[].hosts[]`

Keep `llm.underpassai.com` and `vllm.underpassai.com` unchanged.

Verify the host was released:

```bash
kubectl -n underpass-runtime get ingress underpass-runtime-vllm -o yaml | \
  grep "0.5b.llm.underpassai.com" || true
```

Expected: no output.

### Step 2: Brief expected downtime window

Between Step 1 and Step 3, `0.5b.llm.underpassai.com` has no backing ingress.
Requests should return 404. Typical window: 30-60 seconds.

```bash
curl -k -o /dev/null -s -w "%{http_code}\n" \
  https://0.5b.llm.underpassai.com/v1/models
```

Expected: `404`.

### Step 3: Operator claims the host

```bash
helm upgrade --install underpass-llm-operator-qwen05 charts/vllm \
  -f env/prod/operator-qwen05-v812.yaml \
  --namespace underpass-runtime

kubectl -n underpass-runtime wait --for=condition=ready pod \
  -l app.kubernetes.io/instance=underpass-llm-operator-qwen05 \
  --timeout=10m
```

### Step 4: Verify operator owns the host end-to-end

```bash
kubectl -n underpass-runtime get ingress -o wide | grep 0.5b.llm

curl -k --cert /tmp/client.crt --key /tmp/client.key \
  https://0.5b.llm.underpassai.com/v1/models | jq -r '.data[].id'
```

Expected: output includes `operator-v8.1.2`.

## Rollback

If Step 3 fails, restore the previous live routing:

```bash
helm uninstall underpass-llm-operator-qwen05 --namespace underpass-runtime
kubectl -n underpass-runtime edit ingress underpass-runtime-vllm
```

Re-add `0.5b.llm.underpassai.com` to:

- `spec.rules[]`, pointing to `underpass-llm-gemma-4-31b-structured:8000`
- `spec.tls[].hosts[]`

Verify:

```bash
kubectl -n underpass-runtime get ingress underpass-runtime-vllm -o yaml | \
  grep -A 8 "0.5b.llm.underpassai.com"
```

This restores the pre-transfer ingress ownership. If the Gemma structured
service has no ready endpoints, the host may still be non-functional, but the
ingress conflict is cleanly reverted.

## Post-deployment validation

After Step 4 succeeds, run a minimal chat completion smoke:

```bash
curl -k --cert /tmp/client.crt --key /tmp/client.key \
  https://0.5b.llm.underpassai.com/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "operator-v8.1.2",
    "messages": [
      {"role": "system", "content": "You are an operator agent for KMP navigation."},
      {"role": "user", "content": "Goal: inspect node:test\nVisible refs: [node:test]\nAllowed: [kernel_inspect]"}
    ],
    "max_tokens": 256,
    "temperature": 0.0
  }'
```

Expected: `choices[0].message.content` contains a valid operator action.

## Notes

- DNS remains unchanged. Route53 keeps pointing the host at the cluster ingress
  IP.
- mTLS remains required via `underpass-demo-client-tls`.
- The operator release creates its own TLS secret:
  `underpass-llm-operator-qwen05-vllm-tls`.
- Do not deploy the operator release until the legacy live ingress has released
  `0.5b.llm.underpassai.com`; otherwise two ingresses claim the same host.
