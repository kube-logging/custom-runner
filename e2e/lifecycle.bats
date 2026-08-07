#!/usr/bin/env bats
# Process lifecycle: signals, graceful shutdown, and child cleanup.

setup() {
    load helpers
}

teardown() {
    stop_runner
}

@test "SIGTERM shuts the runner down cleanly" {
    start_runner --exec 'server -> while true; do sleep 1; done'

    kill -TERM "${RUNNER_PID}"

    local rc=0
    wait "${RUNNER_PID}" || rc=$?
    RUNNER_PID=""
    [ "$rc" -eq 0 ]
}

@test "SIGINT shuts the runner down cleanly" {
    start_runner --exec 'server -> while true; do sleep 1; done'

    kill -INT "${RUNNER_PID}"

    local rc=0
    wait "${RUNNER_PID}" || rc=$?
    RUNNER_PID=""
    [ "$rc" -eq 0 ]
}

# The command is a shell loop, so `sleep` is a *grandchild*. Killing only the
# shell leaves it orphaned on pid 1, which is why processes run in their own group.
@test "SIGTERM does not orphan supervised children or grandchildren" {
    start_runner --exec 'server -> while true; do sleep 3617; done'

    retry 25 bash -c "[ -n \"\$(curl -sS 127.0.0.1:${PORT}/get/server | jq -r '.response.pid // empty')\" ]"
    local child
    child="$(pid_of server)"
    [ -n "$child" ]

    # Resolve the grandchild by parent pid. Matching on the command line would
    # also match this test's own `pgrep ...` wrapper and always "find" it.
    local grandchild
    retry 25 bash -c "pgrep -P ${child} >/dev/null"
    grandchild="$(pgrep -P "${child}" | head -1)"
    [ -n "$grandchild" ]

    kill -TERM "${RUNNER_PID}"
    wait "${RUNNER_PID}" || true
    RUNNER_PID=""

    retry 40 bash -c "! kill -0 ${child} 2>/dev/null"
    retry 40 bash -c "! kill -0 ${grandchild} 2>/dev/null"
}

@test "onExit actions run during shutdown" {
    start_runner -cfgfile "$(fixture onexit.yaml)"

    kill -TERM "${RUNNER_PID}"
    wait "${RUNNER_PID}" || true
    RUNNER_PID=""

    run grep -qF 'ONEXIT-RAN' "${RUNNER_LOG}"
    [ "$status" -eq 0 ]
}

@test "onExit actions run on the exit endpoint too" {
    start_runner -cfgfile "$(fixture onexit.yaml)"

    api -X POST "127.0.0.1:${PORT}/exit" >/dev/null
    wait "${RUNNER_PID}" || true
    RUNNER_PID=""

    run grep -qF 'ONEXIT-RAN' "${RUNNER_LOG}"
    [ "$status" -eq 0 ]
}

@test "the listening port is released after shutdown" {
    start_runner
    local port="${PORT}"

    kill -TERM "${RUNNER_PID}"
    wait "${RUNNER_PID}" || true
    RUNNER_PID=""

    retry 40 bash -c "! (exec 9<>/dev/tcp/127.0.0.1/${port}) 2>/dev/null"
}
