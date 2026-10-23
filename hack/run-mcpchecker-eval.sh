#!/usr/bin/env bash
# Run mcpchecker evals end-to-end against a local setup.
#
# Unless already reachable, this script starts and later stops:
#   - Prometheus port-forward on localhost:9090 (thanos-querier, requires oc login)
#   - Alertmanager port-forward on localhost:9093 (requires oc login)
#   - The MCP server on localhost:8085 (make run-mcp equivalent)
#
# Environment variables:
#   TASK          run a single task by name (regex match)
#   CATEGORY      run all tasks with a matching category label
#   RUNS          number of runs per task (default: 1)
#   EVAL_CONFIG   eval config file name (default: eval.yaml)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EVAL_DIR="$ROOT_DIR/evals/mcpchecker"
MCPCHECKER_BIN="$ROOT_DIR/bin/mcpchecker"

PROM_URL="http://localhost:9090"
ALERTMANAGER_URL="http://localhost:9093"
MCP_URL="http://localhost:8085/healthz"

PIDS=()

cleanup() {
	if [ "${#PIDS[@]}" -gt 0 ]; then
		echo "==> Stopping background processes"
		for pid in "${PIDS[@]}"; do
			kill "$pid" 2>/dev/null || true
		done
	fi
}
trap cleanup EXIT

# ensure_url <url> <name> <command...>
# Starts the command unless the url already answers, then waits for readiness.
ensure_url() {
	local url="$1" name="$2"
	shift 2

	if curl -sf -o /dev/null "$url"; then
		echo "==> $name already reachable at $url, reusing it"
		return 0
	fi

	echo "==> Starting $name"
	"$@" &
	local pid=$!
	PIDS+=("$pid")

	echo "==> Waiting for $name at $url"
	local i
	for i in $(seq 1 60); do
		if curl -sf -o /dev/null "$url"; then
			echo "==> $name is ready"
			return 0
		fi
		if ! kill -0 "$pid" 2>/dev/null; then
			echo "Error: $name exited unexpectedly (for port-forwards, check 'oc login')" >&2
			return 1
		fi
		sleep 1
	done

	echo "Error: $name not ready at $url within 60s" >&2
	return 1
}

if [ -z "${OPENAI_API_KEY:-}" ]; then
	echo "Error: OPENAI_API_KEY is not set. Export it before running the evals:" >&2
	echo '  export OPENAI_API_KEY="sk-..."' >&2
	exit 1
fi

if [ ! -x "$MCPCHECKER_BIN" ]; then
	echo "Error: $MCPCHECKER_BIN not found or not executable." >&2
	echo "  Run 'make install-mcpchecker' first." >&2
	exit 1
fi

command -v oc >/dev/null 2>&1 || {
	echo "Error: oc is required for the Prometheus/Alertmanager port-forwards." >&2
	exit 1
}

cd "$ROOT_DIR"

ensure_url "$PROM_URL/-/ready" "Prometheus" ./hack/listen-thanos.sh
ensure_url "$ALERTMANAGER_URL/-/ready" "Alertmanager" ./hack/listen-alertmanager.sh
ensure_url "$MCP_URL" "MCP server" \
	go run -tags testonly ./main.go mcp --disable-auth-for-testing

cd "$EVAL_DIR"

RUNS="${RUNS:-1}"
EVAL_CONFIG="${EVAL_CONFIG:-eval.yaml}"

ARGS=(check "$EVAL_CONFIG" --runs "$RUNS")
if [ -n "${TASK:-}" ]; then
	ARGS+=(--run "$TASK" --verbose)
elif [ -n "${CATEGORY:-}" ]; then
	ARGS+=(--label-selector "category=$CATEGORY" --parallel 4)
else
	ARGS+=(--parallel 4)
fi

"$MCPCHECKER_BIN" "${ARGS[@]}"
"$MCPCHECKER_BIN" result summary mcpchecker-cluster-health-tools-out.json
