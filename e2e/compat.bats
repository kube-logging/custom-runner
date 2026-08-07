#!/usr/bin/env bats
# logging-operator integration contract.
#
# Each assertion mirrors a real call site in kube-logging/logging-operator. Where
# v1.0 deliberately breaks the old behaviour the test records the new expectation
# and names the operator change required — see MIGRATION.md.

setup() {
    load helpers
}

teardown() {
    stop_runner
}

# UNCHANGED — images/fluentd-drain-watch/drain-watch.sh:27 probes the listener
# with `curl -so /dev/null $ADDR`, which only needs a reply, not a 2xx.
@test "drain-watch: bare GET / answers, so the probe's curl exits zero" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${PORT}"
    [ "$output" = "404" ]
}

# BREAKING — drain-watch.sh:67 currently calls GET /exit. Mutating verbs are now
# POST, so the operator must switch to `curl -X POST`.
@test "drain-watch: POST /exit succeeds and the runner exits zero" {
    start_runner --exec 'nodeexporter -> while true; do sleep 1; done'
    run api -X POST "127.0.0.1:${PORT}/exit"
    [ "$(jq -r .success <<<"$output")" = "true" ]

    # `wait` is a builtin operating on this shell's jobs, so it cannot go through `run`.
    local rc=0
    wait "${RUNNER_PID}" || rc=$?
    RUNNER_PID=""
    [ "$rc" -eq 0 ]
}

@test "drain-watch: GET /exit is now rejected" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${PORT}/exit"
    [ "$output" = "405" ]
}

# UNCHANGED — pkg/resources/{fluentd,fluentbit,syslogng} pass repeated
# --exec "key -> command" pairs, and syslog-ng overrides --port.
@test "sidecar: repeated --exec pairs plus --port register both processes" {
    start_runner \
        --exec 'nodeexporter -> while true; do sleep 1; done' \
        --exec 'buffersize -> while true; do sleep 1; done'

    run api "127.0.0.1:${PORT}/list"
    [ "$(jq -r '[.response[].key] | join(",")' <<<"$output")" = "buffersize,nodeexporter" ]
}

# UNCHANGED — pkg/resources/syslogng/statefulset.go:392 generateConfigReloaderConfig
# watches <configDir>/..data and execs on ConfigMap/Secret swap.
@test "syslog-ng: cfgjson onFileCreate on ..data fires every action" {
    CONFDIR="${BATS_TEST_TMPDIR}/config"
    mkdir -p "$CONFDIR"
    start_runner -cfgjson "$(fixture_json syslog-ng-reloader.json)"

    mkdir -p "${CONFDIR}/..2026_01_01"
    echo data >"${CONFDIR}/..2026_01_01/syslog-ng.conf"
    ln -sfn '..2026_01_01' "${CONFDIR}/..data"

    wait_for_log 'CONFIG-SECRET-CHANGED'
    wait_for_log 'RELOAD-FIRED'
}

@test "servicemonitor: /metrics serves the sidecar_reloader namespace" {
    start_runner --exec 'nodeexporter -> while true; do sleep 1; done'
    run api "127.0.0.1:${METRICS_PORT}/metrics"
    [[ "$output" == *"sidecar_reloader_config_reloader_requests_total"* ]]
}

# UNCHANGED — statefulset.go:382 passes -cfgjson with a single dash.
@test "flags: -cfgjson and --exec spellings are both accepted" {
    start_runner -cfgjson "$(fixture_json onstart.json)" --exec 'b -> echo DOUBLE-DASH-OK'
    wait_for_log 'ONSTART-JSON'
    wait_for_log 'DOUBLE-DASH-OK'
}

# The command API executes arbitrary shell as root-equivalent; a pod IP is
# routable from every other pod in a default cluster, so it must not bind 0.0.0.0.
@test "security: the command api binds loopback by default" {
    start_runner
    run grep -c '"address":"127.0.0.1:'"${PORT}" "${RUNNER_LOG}"
    [ "$output" -ge 1 ]
}

# Metrics must stay reachable for kubelet probes and Prometheus.
@test "security: the metrics listener binds all interfaces" {
    start_runner
    run grep -c '"address":"0.0.0.0:'"${METRICS_PORT}" "${RUNNER_LOG}"
    [ "$output" -ge 1 ]
}

# kubelet only swaps ..data, so the mounted filename is never written. Watching it
# with the obvious onFileWrite must still fire.
@test "kubernetes: a ConfigMap swap fires onFileWrite on the mounted filename" {
    WATCHDIR="${BATS_TEST_TMPDIR}/mount"
    mkdir -p "$WATCHDIR"
    start_runner -cfgfile "$(fixture configmap-write.yaml)"

    mkdir -p "${WATCHDIR}/..2026_01_01"
    echo v2 >"${WATCHDIR}/..2026_01_01/app.conf"
    ln -sfn '..2026_01_01' "${WATCHDIR}/..data"

    wait_for_log 'CONFIGMAP-RELOADED'
}
