#!/usr/bin/env bats
# Metrics and health, served on a listener separate from the command API so that
# scraping never requires exposing the exec endpoints.

setup() {
    load helpers
}

teardown() {
    stop_runner
}

@test "healthz is served once listening" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/healthz"
    [ "$output" = "200" ]
}

@test "readyz reports ready after startup" {
    start_runner
    retry 25 bash -c "[ \"\$(curl -sS -o /dev/null -w '%{http_code}' 127.0.0.1:${METRICS_PORT}/readyz)\" = '200' ]"
}

# A watch that never registers means this runner will silently never reload. That
# has to surface as unready, otherwise the sidecar looks perfectly healthy while
# being useless.
@test "readyz reports 503 when a configured watch could not be registered" {
    WATCHDIR="/nonexistent-$$/deep"
    start_runner -cfgfile "$(fixture onfilecreate.yaml)"

    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/readyz"
    [ "$output" = "503" ]

    run curl -sS "127.0.0.1:${METRICS_PORT}/readyz"
    [[ "$output" == *"/nonexistent-$$/deep/trigger"* ]]
}

# Liveness must stay up while readiness fails; restarting cannot fix a missing mount.
@test "healthz stays 200 while readyz is failing" {
    WATCHDIR="/nonexistent-$$/deep"
    start_runner -cfgfile "$(fixture onfilecreate.yaml)"

    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/readyz"
    [ "$output" = "503" ]

    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/healthz"
    [ "$output" = "200" ]
}

# A family only materialises once observed, so drive one success and one failure.
@test "metrics expose the sidecar_reloader namespace" {
    start_runner --exec 'good -> true' --exec 'bad -> exit 1'

    retry 50 bash -c "curl -sS 127.0.0.1:${METRICS_PORT}/metrics | grep -q 'sidecar_reloader_config_reloader_last_request_duration_seconds'"

    run api "127.0.0.1:${METRICS_PORT}/metrics"
    [[ "$output" == *"sidecar_reloader_config_reloader_requests_total"* ]]
    [[ "$output" == *"sidecar_reloader_config_reloader_success_reloads_total"* ]]
    [[ "$output" == *"sidecar_reloader_config_reloader_request_errors_total"* ]]
    [[ "$output" == *"sidecar_reloader_config_reloader_last_reload_error"* ]]
    [[ "$output" == *"sidecar_reloader_config_reloader_last_request_duration_seconds"* ]]
    [[ "$output" == *"sidecar_reloader_config_reloader_watcher_errors_total"* ]]
}

# A failing process must not bump the success counter.
@test "a failing process does not count as a successful reload" {
    start_runner --exec 'bad -> exit 1'

    retry 50 bash -c "curl -sS 127.0.0.1:${METRICS_PORT}/metrics | grep -q 'request_errors_total{process=\"bad\"} 1'"

    run api "127.0.0.1:${METRICS_PORT}/metrics"
    [[ "$output" != *'sidecar_reloader_config_reloader_success_reloads_total{process="bad"}'* ]]
    [[ "$output" == *'sidecar_reloader_config_reloader_last_reload_error{process="bad"} 1'* ]]
}

@test "a succeeding process does not count as an error" {
    start_runner --exec 'good -> true'

    retry 50 bash -c "curl -sS 127.0.0.1:${METRICS_PORT}/metrics | grep -q 'success_reloads_total{process=\"good\"} 1'"

    run api "127.0.0.1:${METRICS_PORT}/metrics"
    [[ "$output" != *'sidecar_reloader_config_reloader_request_errors_total{process="good"}'* ]]
    [[ "$output" == *'sidecar_reloader_config_reloader_last_reload_error{process="good"} 0'* ]]
}

@test "metrics listener does not serve command verbs" {
    start_runner
    run curl -sS -o /dev/null -w '%{http_code}' "127.0.0.1:${METRICS_PORT}/list"
    [ "$output" = "404" ]
}
