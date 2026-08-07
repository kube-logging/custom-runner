#!/usr/bin/env bats
# HTTP API contract: mutating verbs are POST, reads are GET.

setup() {
    load helpers
}

teardown() {
    stop_runner
}

@test "list and get marshal running processes instead of erroring" {
    start_runner --exec 'ticker -> while true; do sleep 1; done'

    run api "127.0.0.1:${PORT}/list"
    [ "$(jq -r .success <<<"$output")" = "true" ]
    [ "$(jq -r '.response[0].key' <<<"$output")" = "ticker" ]
    [ "$(jq -r '.response[0].pid' <<<"$output")" -gt 0 ]

    run api "127.0.0.1:${PORT}/get/ticker"
    [ "$(jq -r '.response.key' <<<"$output")" = "ticker" ]
}

@test "list is ordered by key" {
    start_runner \
        --exec 'charlie -> while true; do sleep 1; done' \
        --exec 'alpha -> while true; do sleep 1; done' \
        --exec 'bravo -> while true; do sleep 1; done'

    run api "127.0.0.1:${PORT}/list"
    [ "$(jq -r '[.response[].key] | join(",")' <<<"$output")" = "alpha,bravo,charlie" ]
}

@test "get on an unknown key reports not found" {
    start_runner
    run api "127.0.0.1:${PORT}/get/nope"
    [ "$(jq -r .success <<<"$output")" = "false" ]
    [[ "$(jq -r .error <<<"$output")" == *"process not found"* ]]
}

@test "exec twice on the same key is rejected" {
    start_runner --exec 'dupe -> while true; do sleep 1; done'
    run api -X POST --data 'while true; do sleep 1; done' "127.0.0.1:${PORT}/exec/dupe"
    [ "$(jq -r .success <<<"$output")" = "false" ]
    [[ "$(jq -r .error <<<"$output")" == *"already running"* ]]
}

@test "restart replaces the process with a new pid" {
    start_runner --exec 'ticker -> while true; do sleep 1; done'
    local before
    before="$(pid_of ticker)"
    [ -n "$before" ]

    run api -X POST "127.0.0.1:${PORT}/restart/ticker"
    [ "$(jq -r .success <<<"$output")" = "true" ]
    [ "$(pid_of ticker)" -ne "$before" ]
}

@test "kill removes the process from the list" {
    start_runner --exec 'ticker -> while true; do sleep 1; done'
    run api -X POST "127.0.0.1:${PORT}/kill/ticker"
    [ "$(jq -r .success <<<"$output")" = "true" ]

    retry 25 bash -c "curl -sS 127.0.0.1:${PORT}/list | jq -e '.response | length == 0' >/dev/null"
}

@test "unknown command returns 404" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${PORT}/bogus/key"
    [ "$output" = "404" ]
}

@test "mutating verbs reject GET with 405" {
    start_runner --exec 'ticker -> while true; do sleep 1; done'

    for path in "exec/ticker" "kill/ticker" "restart/ticker" "exit"; do
        run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${PORT}/${path}"
        [ "$output" = "405" ]
    done
}

@test "read verbs reject POST with 405" {
    start_runner
    for path in "list" "config"; do
        run curl -sS -o /dev/null -w '%{http_code}' -X POST "127.0.0.1:${PORT}/${path}"
        [ "$output" = "405" ]
    done
}

@test "config endpoint returns the loaded configuration" {
    start_runner -cfgfile "$(fixture onstart.yaml)"
    run api "127.0.0.1:${PORT}/config"
    [ "$(jq -r .success <<<"$output")" = "true" ]
    [ "$(jq -r '.response.events.onStart[0].exec.key' <<<"$output")" = "hello" ]
}

@test "command api does not serve metrics" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${PORT}/metrics"
    [ "$output" = "404" ]
}

@test "port 0 disables the command api but keeps metrics" {
    METRICS_PORT="$(free_port)"
    RUNNER_LOG="${BATS_TEST_TMPDIR}/runner.log"
    (cd "${BATS_TEST_TMPDIR}" && exec "${RUNNER_BIN}" \
        -port 0 -metrics-port "${METRICS_PORT}" --exec 'ticker -> while true; do sleep 1; done') \
        >"${RUNNER_LOG}" 2>&1 3>&- &
    RUNNER_PID=$!
    export METRICS_PORT RUNNER_PID RUNNER_LOG

    wait_for_port "${METRICS_PORT}"
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/healthz"
    [ "$output" = "200" ]
}

# Process keys become Prometheus label values, so an unvalidated key lets a caller
# mint unbounded series on the metrics listener.
@test "invalid process keys are rejected with 400" {
    start_runner
    local long
    long="$(printf 'x%.0s' $(seq 1 65))"

    for key in "a%20b" "a%3Bb" "a%22b" "${long}"; do
        run curl -sS -o /dev/null -w '%{http_code}' -X POST --data 'true' \
            "127.0.0.1:${PORT}/exec/${key}"
        [ "$output" = "400" ]
    done
}

@test "valid process keys are accepted" {
    start_runner
    for key in "node-exporter" "buffer_size" "app.v2" "A1"; do
        run api -X POST --data 'true' "127.0.0.1:${PORT}/exec/${key}"
        [ "$(jq -r .success <<<"$output")" = "true" ]
    done
}
