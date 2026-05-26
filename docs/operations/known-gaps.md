# Known Gaps

## vLLM 0.19 structured output flag drift

`env/prod/operator-qwen05-v812.yaml` currently renders the deprecated
`--guided-decoding-backend=xgrammar` argument. vLLM 0.19 rejects this flag; the
live hotfix removed it via Helm override. The E2E regen script intentionally
fails this profile render check until the values file is corrected in a separate
configuration PR.
