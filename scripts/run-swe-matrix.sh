#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CASES_FILE="${CASES_FILE:-$ROOT_DIR/testdata/swe-matrix/cases.json}"
TWO_PASS_SERVER_URL="${TWO_PASS_SERVER_URL:-http://127.0.0.1:8080}"
OUTPUT_ROOT="${OUTPUT_ROOT:-$ROOT_DIR/tmp/swe-matrix}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUTPUT_DIR="${OUTPUT_DIR:-$OUTPUT_ROOT/$TIMESTAMP}"

MODE="run"
CASE_ID=""
SHOW_RESPONSE=0
SKIP_HEALTHZ=0

PASS1_SYSTEM_PROMPT="${PASS1_SYSTEM_PROMPT:-You are a semantic extraction engine. Return a complete intermediate representation. Do not force the output into strict JSON. Do not invent missing values. Mark uncertainty explicitly.}"
PASS2_SYSTEM_PROMPT="${PASS2_SYSTEM_PROMPT:-You are a strict JSON canonicalization engine. Return only valid JSON. Respect the schema exactly. Do not add fields. Do not invent values. Use only facts explicitly present in the intermediate representation. Prefer the most specific literal value from the intermediate representation for each schema field. Do not replace specific attributes with generic entity labels or types such as Person, Organization, Document, Event, or Thing. If information is missing, use null only if the schema allows it.}"
PASS1_USER_PROMPT_TEMPLATE="${PASS1_USER_PROMPT_TEMPLATE:-Given the following input, extract all facts required for the downstream schema.

Requirements:
- Preserve meaning.
- Do not hallucinate.
- Mark uncertain fields explicitly.
- If a field is missing, say it is missing.
- Return the result using the agreed intermediate format.

Input:
{{input}}}"
PASS2_USER_PROMPT_TEMPLATE="${PASS2_USER_PROMPT_TEMPLATE:-Convert the following intermediate representation into the target schema.

Rules:
- Do not add new facts.
- Copy values from the intermediate representation as literally as possible.
- Prefer specific extracted attributes over generic entity labels or types.
- If the intermediate representation contains both an entity type and a more specific field value, use the more specific field value.
- If a schema field expects a scalar and the intermediate representation provides a list, choose the single most relevant literal item from that list.
- Output exactly one JSON value that matches the target schema.

Target JSON schema:
{{schema}}

Intermediate representation:
{{intermediate}}}"
PASS2_RETRY_HINT_TEMPLATE="${PASS2_RETRY_HINT_TEMPLATE:-Previous attempt failed validation. Correct the output using this feedback:
{{hint}}}"
SINGLE_PASS_SYSTEM_PROMPT="${SINGLE_PASS_SYSTEM_PROMPT:-You are a strict JSON extraction engine. Return only valid JSON matching the target schema. Do not add fields. Do not invent values. Use only facts explicitly present in the input.}"
SINGLE_PASS_USER_PROMPT_TEMPLATE="${SINGLE_PASS_USER_PROMPT_TEMPLATE:-Convert the following input into the target schema.

Rules:
- Use only facts explicitly present in the input.
- Do not hallucinate.
- Preserve literal values when possible.
- Output exactly one JSON value that matches the target schema.

Target JSON schema:
{{schema}}

Input:
{{input}}}"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/run-swe-matrix.sh [--list] [--case <id>] [--show-response] [--skip-healthz]

Environment:
  TWO_PASS_SERVER_URL   Base URL del orquestador two-pass. Default: http://127.0.0.1:8080
  CASES_FILE            Ruta al JSON de casos. Default: testdata/swe-matrix/cases.json
  OUTPUT_ROOT           Carpeta raiz para resultados. Default: tmp/swe-matrix
  OUTPUT_DIR            Carpeta final de resultados. Default: OUTPUT_ROOT/<timestamp>
EOF
}

render_prompt_template() {
  local template="$1"
  shift

  local rendered="$template"
  while [[ $# -gt 1 ]]; do
    local key="$1"
    local value="$2"
    rendered="${rendered//\{\{$key\}\}/$value}"
    shift 2
  done

  printf '%s' "$rendered"
}

build_pass1_prompt() {
  local input="$1"
  render_prompt_template "$PASS1_USER_PROMPT_TEMPLATE" input "$input"
}

build_pass2_prompt_base() {
  local intermediate="$1"
  local schema="$2"
  render_prompt_template "$PASS2_USER_PROMPT_TEMPLATE" schema "$schema" intermediate "$intermediate"
}

build_single_pass_prompt() {
  local input="$1"
  local schema="$2"
  render_prompt_template "$SINGLE_PASS_USER_PROMPT_TEMPLATE" input "$input" schema "$schema"
}

metadata_execution_mode() {
  local file="$1"
  jq -r '.metadata.execution_mode // ""' "$file" 2>/dev/null || true
}

metadata_attempts() {
  local file="$1"
  local mode="$2"
  if [[ "$mode" == "single_pass" ]]; then
    jq -r '.metadata.single_pass.attempts // "-"' "$file"
    return
  fi
  jq -r '.metadata.pass2.attempts // "-"' "$file"
}

metadata_latency_ms() {
  local file="$1"
  local mode="$2"
  if [[ "$mode" == "single_pass" ]]; then
    jq -r '.metadata.single_pass.latency_ms // "-"' "$file"
    return
  fi
  jq -r '.metadata.pass2.latency_ms // "-"' "$file"
}

pretty_json_or_raw() {
  local file="$1"
  if jq . "$file" >/dev/null 2>&1; then
    jq . "$file"
  else
    cat "$file"
  fi
}

extract_output_json() {
  local file="$1"
  if jq '.output' "$file" >/dev/null 2>&1; then
    jq '.output' "$file"
  else
    printf 'null\n'
  fi
}

write_case_report() {
  local case_json="$1"
  local request_file="$2"
  local response_file="$3"
  local report_file="$4"
  local http_code="$5"
  local curl_exit="$6"
  local curl_error_file="$7"

  local case_name category description request_id execution_mode pass1_attempts downstream_attempts pass1_latency_ms downstream_latency_ms
  local input_text schema_version schema_text pass1_user_prompt intermediate pass2_prompt_base single_pass_prompt response_body output_body

  case_name="$(jq -r '.id' <<<"$case_json")"
  category="$(jq -r '.category' <<<"$case_json")"
  description="$(jq -r '.description' <<<"$case_json")"
  input_text="$(jq -r '.payload.input' <<<"$case_json")"
  schema_version="$(jq -r '.payload.schema_version' <<<"$case_json")"
  schema_text="$(jq '.payload.schema' <<<"$case_json")"
  pass1_user_prompt="$(build_pass1_prompt "$input_text")"

  if jq . "$response_file" >/dev/null 2>&1; then
    execution_mode="$(metadata_execution_mode "$response_file")"
    request_id="$(jq -r '.request_id // "-"' "$response_file")"
    pass1_attempts="$(jq -r '.metadata.pass1.attempts // "-"' "$response_file")"
    downstream_attempts="$(metadata_attempts "$response_file" "$execution_mode")"
    pass1_latency_ms="$(jq -r '.metadata.pass1.latency_ms // "-"' "$response_file")"
    downstream_latency_ms="$(metadata_latency_ms "$response_file" "$execution_mode")"
    intermediate="$(jq -r '.intermediate_representation // ""' "$response_file")"
    response_body="$(pretty_json_or_raw "$response_file")"
    output_body="$(extract_output_json "$response_file")"
  else
    execution_mode=""
    request_id="-"
    pass1_attempts="-"
    downstream_attempts="-"
    pass1_latency_ms="-"
    downstream_latency_ms="-"
    intermediate=""
    response_body="$(cat "$response_file" 2>/dev/null || true)"
    output_body="null"
  fi

  pass2_prompt_base="$(build_pass2_prompt_base "$intermediate" "$schema_text")"
  single_pass_prompt="$(build_single_pass_prompt "$input_text" "$schema_text")"

  {
    printf '# %s\n\n' "$case_name"
    printf -- '- Category: `%s`\n' "$category"
    printf -- '- Description: %s\n' "$description"
    printf -- '- Endpoint: `%s`\n' "$TWO_PASS_SERVER_URL"
    printf -- '- Execution mode: `%s`\n' "${execution_mode:--}"
    printf -- '- HTTP: `%s`\n' "$http_code"
    printf -- '- curl exit: `%s`\n' "$curl_exit"
    printf -- '- Request ID: `%s`\n' "$request_id"
    if [[ "$execution_mode" == "single_pass" ]]; then
      printf -- '- Single-pass attempts: `%s`\n' "$downstream_attempts"
      printf -- '- Single-pass latency ms: `%s`\n' "$downstream_latency_ms"
    else
      printf -- '- Pass 1 attempts: `%s`\n' "$pass1_attempts"
      printf -- '- Pass 2 attempts: `%s`\n' "$downstream_attempts"
      printf -- '- Pass 1 latency ms: `%s`\n' "$pass1_latency_ms"
      printf -- '- Pass 2 latency ms: `%s`\n' "$downstream_latency_ms"
    fi
    printf '\n## Judge Focus\n\n'
    jq -r '.judge_focus[] | "- " + .' <<<"$case_json"
    printf '\n## Failure Modes To Watch\n\n'
    jq -r '.failure_modes[] | "- " + .' <<<"$case_json"
    printf '\n## Request Payload\n\n```json\n'
    jq . "$request_file"
    printf '```\n\n## Prompt Stack\n\n'
    if [[ "$execution_mode" == "single_pass" ]]; then
      printf '### Single-pass System Prompt\n\n```text\n%s\n```\n' "$SINGLE_PASS_SYSTEM_PROMPT"
      printf '\n### Single-pass User Prompt\n\n```text\n%s\n```\n' "$single_pass_prompt"
    else
      printf '### Pass 1 System Prompt\n\n```text\n%s\n```\n' "$PASS1_SYSTEM_PROMPT"
      printf '\n### Pass 1 User Prompt\n\n```text\n%s\n```\n' "$pass1_user_prompt"
      printf '\n### Pass 2 System Prompt\n\n```text\n%s\n```\n' "$PASS2_SYSTEM_PROMPT"
      printf '\n### Pass 2 Base Prompt\n\n```text\n%s\n```\n' "$pass2_prompt_base"
    fi
    printf '\n'
    if [[ "$execution_mode" != "single_pass" && "$downstream_attempts" != "-" && "$downstream_attempts" != "1" ]]; then
      printf 'Pass 2 hizo reintentos. El feedback de validacion no viaja en la respuesta, asi que el prompt final exacto no se puede reconstruir desde la API.\n\n'
    fi
    if [[ "$execution_mode" != "single_pass" ]]; then
      printf '## Intermediate Representation\n\n```text\n%s\n```\n' "$intermediate"
    fi
    printf '\n## Output JSON\n\n```json\n%s\n```\n' "$output_body"
    printf '\n## Raw Response\n\n```json\n%s\n```\n' "$response_body"
    if [[ "$curl_exit" -ne 0 && -s "$curl_error_file" ]]; then
      printf '\n## curl stderr\n\n```text\n'
      cat "$curl_error_file"
      printf '```\n'
    fi
    printf '\n## Judge Notes\n\n'
    printf -- '- Fidelity to explicit facts:\n'
    printf -- '- Hallucinations or unsupported inferences:\n'
    printf -- '- Null and uncertainty discipline:\n'
    printf -- '- Schema obedience:\n'
    printf -- '- Utility for SWE workflow:\n'
    printf -- '- Verdict:\n'
  } >"$report_file"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --list)
      MODE="list"
      shift
      ;;
    --case)
      CASE_ID="${2:-}"
      if [[ -z "$CASE_ID" ]]; then
        echo "--case requires an id" >&2
        exit 2
      fi
      shift 2
      ;;
    --show-response)
      SHOW_RESPONSE=1
      shift
      ;;
    --skip-healthz)
      SKIP_HEALTHZ=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

command -v jq >/dev/null 2>&1 || { echo "jq is required" >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }

if [[ ! -f "$CASES_FILE" ]]; then
  echo "cases file not found: $CASES_FILE" >&2
  exit 1
fi

FILTER='.cases[]'
if [[ -n "$CASE_ID" ]]; then
  FILTER=".cases[] | select(.id == \"$CASE_ID\")"
fi

if [[ "$MODE" == "list" ]]; then
  jq -r "$FILTER | [.id, .category, .description] | @tsv" "$CASES_FILE" \
    | awk 'BEGIN { printf "%-36s %-18s %s\n", "CASE ID", "CATEGORY", "DESCRIPTION" } { printf "%-36s %-18s %s\n", $1, $2, substr($0, index($0,$3)) }'
  exit 0
fi

mkdir -p "$OUTPUT_DIR"

if [[ "$SKIP_HEALTHZ" -ne 1 ]]; then
  curl -fsS "$TWO_PASS_SERVER_URL/healthz" >/dev/null
fi

mapfile -t CASES < <(jq -c "$FILTER" "$CASES_FILE")

if [[ "${#CASES[@]}" -eq 0 ]]; then
  echo "no cases selected" >&2
  exit 1
fi

SUMMARY_FILE="$OUTPUT_DIR/REPORT.md"

{
  printf '# SWE Matrix Report\n\n'
  printf -- '- Generated at: `%s`\n' "$TIMESTAMP"
  printf -- '- Endpoint: `%s`\n' "$TWO_PASS_SERVER_URL"
  printf -- '- Cases file: `%s`\n' "$CASES_FILE"
  printf '\n| Case | HTTP | curl | Request ID | Attempts | Report |\n'
  printf '| --- | --- | --- | --- | --- | --- |\n'
} >"$SUMMARY_FILE"

echo "Running SWE matrix against: $TWO_PASS_SERVER_URL"
echo "Results dir: $OUTPUT_DIR"
echo
printf "%-36s %-8s %-8s %-14s %s\n" "CASE ID" "HTTP" "CURL" "REQUEST ID" "REPORT"

TOTAL=0
HTTP_ERRORS=0
TRANSPORT_ERRORS=0

for CASE_JSON in "${CASES[@]}"; do
  TOTAL=$((TOTAL + 1))

  CASE_NAME="$(jq -r '.id' <<<"$CASE_JSON")"
  PAYLOAD="$(jq -c '.payload' <<<"$CASE_JSON")"
  REQUEST_FILE="$OUTPUT_DIR/$CASE_NAME.request.json"
  RESPONSE_FILE="$OUTPUT_DIR/$CASE_NAME.response.json"
  REPORT_FILE="$OUTPUT_DIR/$CASE_NAME.report.md"
  CURL_ERROR_FILE="$OUTPUT_DIR/$CASE_NAME.curl.stderr.log"

  jq '.payload' <<<"$CASE_JSON" >"$REQUEST_FILE"

  set +e
  HTTP_CODE="$(
    curl -sS \
      -o "$RESPONSE_FILE" \
      -w '%{http_code}' \
      "$TWO_PASS_SERVER_URL/v1/two-pass/structured" \
      -H 'content-type: application/json' \
      -d "$PAYLOAD" \
      2>"$CURL_ERROR_FILE"
  )"
  CURL_EXIT=$?
  set -e

  if [[ "$CURL_EXIT" -ne 0 ]]; then
    TRANSPORT_ERRORS=$((TRANSPORT_ERRORS + 1))
    HTTP_CODE="000"
    : >"$RESPONSE_FILE"
  elif [[ "$HTTP_CODE" != "200" ]]; then
    HTTP_ERRORS=$((HTTP_ERRORS + 1))
  fi

  write_case_report "$CASE_JSON" "$REQUEST_FILE" "$RESPONSE_FILE" "$REPORT_FILE" "$HTTP_CODE" "$CURL_EXIT" "$CURL_ERROR_FILE"

  if jq . "$RESPONSE_FILE" >/dev/null 2>&1; then
    REQUEST_ID="$(jq -r '.request_id // "-"' "$RESPONSE_FILE")"
    EXECUTION_MODE="$(metadata_execution_mode "$RESPONSE_FILE")"
    PASS2_ATTEMPTS="$(metadata_attempts "$RESPONSE_FILE" "$EXECUTION_MODE")"
  else
    REQUEST_ID="-"
    PASS2_ATTEMPTS="-"
  fi

  printf "%-36s %-8s %-8s %-14s %s\n" "$CASE_NAME" "$HTTP_CODE" "$CURL_EXIT" "$REQUEST_ID" "$(basename "$REPORT_FILE")"

  printf '| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n' \
    "$CASE_NAME" \
    "$HTTP_CODE" \
    "$CURL_EXIT" \
    "$REQUEST_ID" \
    "$PASS2_ATTEMPTS" \
    "$(basename "$REPORT_FILE")" >>"$SUMMARY_FILE"

  if [[ "$SHOW_RESPONSE" -eq 1 ]]; then
    echo
    echo "===== $CASE_NAME ====="
    pretty_json_or_raw "$RESPONSE_FILE"
    echo
  fi
done

{
  printf '\n## Totals\n\n'
  printf -- '- Cases run: `%s`\n' "$TOTAL"
  printf -- '- HTTP errors: `%s`\n' "$HTTP_ERRORS"
  printf -- '- Transport errors: `%s`\n' "$TRANSPORT_ERRORS"
} >>"$SUMMARY_FILE"

echo
echo "Cases run: $TOTAL"
echo "HTTP errors: $HTTP_ERRORS"
echo "Transport errors: $TRANSPORT_ERRORS"
echo "Summary: $SUMMARY_FILE"
