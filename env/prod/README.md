# Production profiles

Profiles for models currently deployed in production clusters.

## Active

- `gemma-4-31b.yaml` - primary structured-output model. Helm release:
  `underpass-llm-gemma-4-31b` in namespace `underpass-runtime`.

## Adding a new production profile

A profile graduates here when:
1. The model has a successful deployment lifecycle (deploy + uninstall both
   tested in a real cluster);
2. Documentation exists in `docs/`;
3. A Makefile target exists for it (preferred: `make helm-upgrade-<model>`).
