# Lab profiles

Profiles for models that work as references or experiments but are NOT
currently deployed in production.

## Profiles

- `gemma-4-26b-a4b.yaml` - smaller Gemma variant; "quick default" reference
- `gpt-oss-120b.yaml` - large GPT-OSS reference; "premium default"
- `gpt-oss-20b.yaml` - small GPT-OSS reference; single-GPU profile

## Policy

Profiles here:
- compile via `helm template` against the chart;
- are documented in `docs/`;
- are NOT part of CI/CD validation;
- can be promoted to `env/prod/` after a successful production deployment.

To deploy a lab profile experimentally:

```bash
helm upgrade --install <release-name> charts/vllm \
  -f env/lab/<profile>.yaml \
  --namespace <namespace>
```
