#!/usr/bin/env bash
# Verify vLLM Helm profiles, cluster drift, ingress ownership, and adapters.
# Reference: docs/operations/preflight.md

set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
# shellcheck source=scripts/e2e/lib.sh
source "$SCRIPT_DIR/lib.sh"

parse_common_flags "$@"

NAMESPACE=${NAMESPACE:-underpass-runtime}
CLIENT_CERT=${CLIENT_CERT:-/tmp/client.crt}
CLIENT_KEY=${CLIENT_KEY:-/tmp/client.key}
EXPECTED_OPERATOR_ADAPTER_SHA=${EXPECTED_OPERATOR_ADAPTER_SHA:-43186fa848c5f0e9d71915023f8f01c2341042de8aaf57b0c3c0c574a0f44379}
OPERATOR_ADAPTER_DIR=${OPERATOR_ADAPTER_DIR:-/var/lib/operator-adapters/v8.1.2-sft-v2-canonical}
VLLM_RELEASE_PROFILE_MAP=${VLLM_RELEASE_PROFILE_MAP:-underpass-llm-operator-qwen05=env/prod/operator-qwen05-v812.yaml underpass-llm-gemma-4-31b=env/prod/gemma-4-31b.yaml}
ENDPOINT_MODEL_CHECKS=${ENDPOINT_MODEL_CHECKS:-https://0.5b.llm.underpassai.com/v1/models=operator-v8.1.2}
DEPRECATED_ARG=${DEPRECATED_ARG:---guided-decoding-backend=xgrammar}

if [[ "$ALLOW_REDEPLOY" == "1" ]]; then
  warn "redeploy flag" "--redeploy accepted but no Helm mutations are implemented in this verifier"
fi
if [[ "$ALLOW_REINSTALL_ADAPTERS" == "1" ]]; then
  if [[ -x scripts/install-operator-adapter.sh ]]; then
    warn "adapter reinstall flag" "run sudo bash scripts/install-operator-adapter.sh manually after reviewing output"
  else
    warn "adapter reinstall flag" "no adapter install script found in this repo"
  fi
fi

profile_for_release() {
  local release=$1
  local pair
  for pair in $VLLM_RELEASE_PROFILE_MAP; do
    if [[ ${pair%%=*} == "$release" ]]; then
      printf '%s' "${pair#*=}"
      return 0
    fi
  done
  return 1
}

render_profile() {
  local profile=$1
  local release=$2
  helm template "$release" charts/vllm -f "$profile"
}

first_rendered_image() {
  local profile=$1
  local release=$2
  render_profile "$profile" "$release" | awk '/image:/ {gsub(/"/, "", $2); print $2; exit}'
}

check_profile_render() {
  local profile=$1
  local release
  release=$(basename "$profile" .yaml | tr '_' '-')
  local rendered
  if ! rendered=$(render_profile "$profile" "$release" 2>&1); then
    fail "template $profile" "helm template failed: $rendered"
    return 1
  fi
  if printf '%s' "$rendered" | grep -Fq -- "$DEPRECATED_ARG"; then
    fail "deprecated args $profile" "$DEPRECATED_ARG is present; remove it for vLLM 0.19+"
    return 1
  fi
  ok "template $profile" "renders without deprecated $DEPRECATED_ARG"
  return 0
}

check_release_drift() {
  local release=$1
  local profile=$2
  local tmp=/tmp/${release}-cluster-values.yaml
  if ! helm get values "$release" -n "$NAMESPACE" >"$tmp" 2>"$tmp.err"; then
    fail "values drift $release" "helm get values failed: $(cat "$tmp.err")"
    return 1
  fi
  if diff -u "$profile" "$tmp" >/tmp/${release}-values.diff 2>&1; then
    ok "values drift $release" "cluster values match $profile"
  else
    warn "values drift $release" "cluster values differ from $profile; inspect /tmp/${release}-values.diff"
  fi
}

check_release_image() {
  local release=$1
  local profile=$2
  local expected
  expected=$(first_rendered_image "$profile" "$release" || true)
  if [[ -z "$expected" ]]; then
    warn "vLLM image $release" "could not infer expected image from $profile"
    return 0
  fi
  check_kubectl_pod_image "$NAMESPACE" "app.kubernetes.io/instance=$release" "$expected" "vLLM image $release"
}

check_ingress_hosts() {
  local release=$1
  local hosts
  hosts=$(kubectl -n "$NAMESPACE" get ingress -l "app.kubernetes.io/instance=$release" -o jsonpath='{range .items[*]}{range .spec.rules[*]}{.host}{"\n"}{end}{end}' 2>/tmp/${release}-ingress.err || true)
  if [[ -z "$hosts" ]]; then
    warn "ingress hosts $release" "no ingress hosts found for release $release"
    return 0
  fi
  local duplicates
  duplicates=$(printf '%s\n' "$hosts" | awk 'NF {count[$0]++} END {for (host in count) if (count[host] > 1) print host}')
  if [[ -n "$duplicates" ]]; then
    fail "ingress hosts $release" "duplicate hosts: $duplicates"
    return 1
  fi
  ok "ingress hosts $release" "unique hosts: $(printf '%s' "$hosts" | tr '\n' ' ')"
}

check_local_adapter() {
  local dir=$1
  local expected=$2
  local file=$dir/adapter_model.safetensors
  if [[ ! -d "$dir" ]]; then
    fail "local adapter" "$dir does not exist; run adapter install script manually"
    return 1
  fi
  if [[ ! -s "$file" ]]; then
    fail "local adapter" "$file missing or empty"
    return 1
  fi
  local actual
  actual=$(sha256sum "$file" | awk '{print $1}')
  if [[ "$actual" != "$expected" ]]; then
    fail "local adapter" "SHA mismatch for $file: expected $expected got $actual"
    return 1
  fi
  ok "local adapter" "$file SHA verified"
}

check_branch_state "$REPO_ROOT" "git state"
run_checked "helm lint vllm" "charts/vllm linted" helm lint charts/vllm

shopt -s nullglob
profiles=(env/prod/*.yaml env/lab/*.yaml)
if [[ ${#profiles[@]} -eq 0 ]]; then
  fail "values profiles" "no env/prod/*.yaml or env/lab/*.yaml profiles found"
else
  ok "values profiles" "found ${#profiles[@]} prod/lab profiles"
fi
for profile in "${profiles[@]}"; do
  check_profile_render "$profile"
done

if helm list -n "$NAMESPACE" >/tmp/vllms-helm-list.txt 2>&1; then
  ok "helm list" "helm list succeeded for namespace $NAMESPACE"
  if [[ "$VERBOSE" == "1" ]]; then
    cat /tmp/vllms-helm-list.txt
  fi
else
  fail "helm list" "helm list failed: $(cat /tmp/vllms-helm-list.txt)"
fi

for pair in $VLLM_RELEASE_PROFILE_MAP; do
  release=${pair%%=*}
  profile=${pair#*=}
  if [[ ! -f "$profile" ]]; then
    fail "release profile $release" "$profile not found"
    continue
  fi
  check_release_drift "$release" "$profile"
  check_release_image "$release" "$profile"
  check_ingress_hosts "$release"
done

while read -r release _rest; do
  [[ -z "$release" || "$release" == NAME ]] && continue
  if [[ "$release" == underpass-llm-* ]] && ! profile_for_release "$release" >/dev/null; then
    warn "release mapping $release" "no profile mapping configured; extend VLLM_RELEASE_PROFILE_MAP"
  fi
done </tmp/vllms-helm-list.txt

for check in $ENDPOINT_MODEL_CHECKS; do
  url=${check%%=*}
  expected=${check#*=}
  check_endpoint_model "$url" "$CLIENT_CERT" "$CLIENT_KEY" "$expected" "models $url"
done

if grep -R "operator-adapters" env/prod/*.yaml >/tmp/vllms-adapter-refs.txt 2>/dev/null; then
  check_local_adapter "$OPERATOR_ADAPTER_DIR" "$EXPECTED_OPERATOR_ADAPTER_SHA"
else
  warn "adapter refs" "no operator adapter references found in env/prod/*.yaml"
fi

finish_summary
