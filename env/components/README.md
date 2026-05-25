# Two-pass component profiles

Partial profiles for assembling a `two-pass` deployment (reasoning +
structured + orchestrator). Each profile is incomplete on its own.

## Profiles

- `reasoning.yaml` - reasoning component values
- `structured.yaml` - structured component values
- `orchestrator.yaml` - orchestrator component values

## Use

The Makefile targets `helm-upgrade-reasoning`, `helm-upgrade-structured`,
and `helm-upgrade-orchestrator` consume these profiles directly.

For a complete two-pass deployment, deploy all three releases with values
from this directory plus model-specific overrides.

See [docs/deployment.md](../../docs/deployment.md) for assembly details.
