#!/usr/bin/env bash
# Shared helpers for the custom-runner e2e suite.

RUNNER_BIN="${RUNNER_BIN:-${BATS_TEST_DIRNAME}/../bin/runner}"

# bats reserves file descriptor 3, so the probe runs in a subshell on fd 9 and
# lets the subshell exit close it.
port_open() {
    (exec 9<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}

free_port() {
    local port
    for _ in $(seq 1 50); do
        port=$(( (RANDOM % 20000) + 20000 ))
        if ! port_open "${port}"; then
            echo "${port}"
            return 0
        fi
    done
    echo "no free port found" >&2
    return 1
}

# fixture <name> — copy a fixture into the test tmpdir, expanding __WATCHDIR__
# and __CONFDIR__, and echo the resulting path.
fixture() {
    local dest="${BATS_TEST_TMPDIR}/$1"
    mkdir -p "$(dirname "${dest}")"
    sed -e "s|__WATCHDIR__|${WATCHDIR:-/nonexistent}|g" \
        -e "s|__CONFDIR__|${CONFDIR:-/nonexistent}|g" \
        "${BATS_TEST_DIRNAME}/fixtures/$1" >"${dest}"
    echo "${dest}"
}

# fixture_json <name> — same expansion, but echo the contents for -cfgjson.
fixture_json() {
    cat "$(fixture "$1")"
}

# start_runner <args...> — launch the runner on free ports.
#
# free_port probes and the runner binds a moment later, so two tests can pick the
# same port. Retry on a lost race rather than failing the test for it.
start_runner() {
    RUNNER_LOG="${BATS_TEST_TMPDIR}/runner.log"

    for _ in 1 2 3 4 5; do
        PORT="$(free_port)"
        METRICS_PORT="$(free_port)"
        while [[ "${METRICS_PORT}" == "${PORT}" ]]; do
            METRICS_PORT="$(free_port)"
        done

        # Supervised commands are arbitrary shell; run from the tmpdir so a stray
        # redirect cannot write into the repo. exec keeps $! as the runner's own pid.
        # 3>&- is mandatory: bats reports through fd 3, and any descendant holding it
        # open keeps bats from ever seeing EOF.
        (cd "${BATS_TEST_TMPDIR}" && exec "${RUNNER_BIN}" \
            -port "${PORT}" -metrics-port "${METRICS_PORT}" "$@") >"${RUNNER_LOG}" 2>&1 3>&- &
        RUNNER_PID=$!
        export PORT METRICS_PORT RUNNER_PID RUNNER_LOG

        if wait_for_port "${PORT}" quiet; then
            return 0
        fi
        if ! grep -q 'address already in use' "${RUNNER_LOG}"; then
            break
        fi
        stop_runner
    done

    echo "runner never came up; log:" >&2
    cat "${RUNNER_LOG}" >&2 || true
    return 1
}

wait_for_port() {
    for _ in $(seq 1 100); do
        port_open "$1" && return 0
        # Give up early if the runner already died; nothing will open the port.
        if [[ -n "${RUNNER_PID:-}" ]] && ! kill -0 "${RUNNER_PID}" 2>/dev/null; then
            break
        fi
        sleep 0.1
    done

    if [[ "${2:-}" != "quiet" ]]; then
        echo "port $1 never opened; runner log:" >&2
        cat "${RUNNER_LOG}" >&2 || true
    fi

    return 1
}

stop_runner() {
    if [[ -n "${RUNNER_PID:-}" ]] && kill -0 "${RUNNER_PID}" 2>/dev/null; then
        pkill -P "${RUNNER_PID}" 2>/dev/null || true
        kill -TERM "${RUNNER_PID}" 2>/dev/null || true
        for _ in $(seq 1 30); do
            kill -0 "${RUNNER_PID}" 2>/dev/null || return 0
            sleep 0.1
        done
        kill -KILL "${RUNNER_PID}" 2>/dev/null || true
    fi
}

api() {
    curl -sS --max-time 10 "$@"
}

retry() {
    local tries="$1"; shift
    for _ in $(seq 1 "${tries}"); do
        if "$@"; then return 0; fi
        sleep 0.2
    done
    return 1
}

log_contains() {
    grep -qF "$1" "${RUNNER_LOG}"
}

wait_for_log() {
    retry "${2:-50}" log_contains "$1"
}

# pid_of <key> — current pid for a registered process, or empty.
pid_of() {
    api "127.0.0.1:${PORT}/get/$1" | jq -r '.response.pid // empty'
}
