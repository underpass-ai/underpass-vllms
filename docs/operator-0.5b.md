# Operator 0.5B

Qwen 2.5 0.5B plus the Operator LoRA adapter trained in the operator repo.
This deployment is for KMP memory-navigation actions, consumed directly by
the operator runtime through the vLLM OpenAI-compatible API.

## Profile

- Role: `structured` vLLM endpoint
- Reasoning component: disabled
- Orchestrator: disabled; the operator runtime validates and executes actions
- Base model: `Qwen/Qwen2.5-0.5B-Instruct`
- LoRA adapter id: `operator-v8.1.2`
- Release name: `underpass-llm-operator-qwen05`

## Adapter

- Source: `/tmp/operator-qwen05-lora-v8.1.2-sft-v2`
- Canonical hostPath: `/var/lib/operator-adapters/v8.1.2-sft-v2-canonical`
- SHA256: `43186fa848c5f0e9d71915023f8f01c2341042de8aaf57b0c3c0c574a0f44379`
- Trained: v8.1.2 SFT v2 in the operator repo
- Intended profile: read-side operator tools only

Install the adapter before deploying:

```bash
sudo bash scripts/install-operator-adapter.sh
```

The script copies the adapter from `/tmp`, verifies the safetensors SHA, and
leaves the canonical directory ready for the vLLM hostPath mount.

## Endpoint

- URL: `https://0.5b.llm.underpassai.com/v1/chat/completions`
- Model identifier: `operator-v8.1.2`
- mTLS: required
- Values file: `env/prod/operator-qwen05-v812.yaml`

The endpoint is direct vLLM. Requests should use vLLM guided JSON with the
operator action schema.

## Deploy

Validate first:

```bash
helm lint charts/vllm -f env/prod/operator-qwen05-v812.yaml
helm template underpass-llm-operator-qwen05 charts/vllm \
  -n underpass-runtime \
  -f env/prod/operator-qwen05-v812.yaml > /tmp/operator-qwen05-v812.yaml
kubectl apply --dry-run=client -f /tmp/operator-qwen05-v812.yaml
```

Deploy after validation:

```bash
helm upgrade --install underpass-llm-operator-qwen05 charts/vllm \
  -n underpass-runtime \
  -f env/prod/operator-qwen05-v812.yaml
```

## Smoke

```bash
curl -k --cert /tmp/client.crt --key /tmp/client.key \
  https://0.5b.llm.underpassai.com/v1/models
```

Expected: `operator-v8.1.2` appears in the model list.

## Limitations

- Write profile is not ready. Do not use `kernel_ingest` or
  `kernel_write_memory` in production runtime allowlists.
- Runtime callers must use guided JSON or equivalent structured decoding.
- The existing cluster ingress may already route `0.5b.llm.underpassai.com` to
  an older backend. Confirm the live ingress backend before deployment.
