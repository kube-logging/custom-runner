#!/usr/bin/env bats
# Config loading and event dispatch.

setup() {
    load helpers
}

teardown() {
    stop_runner
}

@test "startup flag runs a command" {
    start_runner -startup 'echo STARTUP-RAN'
    wait_for_log 'STARTUP-RAN'
}

@test "exec flag uses key -> command arrow syntax" {
    start_runner --exec 'greeter -> echo EXEC-FLAG-RAN'
    wait_for_log 'EXEC-FLAG-RAN'
}

@test "repeated exec flags register independently" {
    start_runner --exec 'one -> echo FIRST-RAN' --exec 'two -> echo SECOND-RAN'
    wait_for_log 'FIRST-RAN'
    wait_for_log 'SECOND-RAN'
}

# ExecArgs.Set splits on "->" and rejoins the tail, so an arrow inside the command
# survives. Quoted so the shell does not read "->" as a redirect.
@test "exec flag command may itself contain an arrow" {
    start_runner --exec 'arrow -> echo "A -> B"'
    wait_for_log 'A -> B'
}

@test "onStart actions run from a yaml config file" {
    start_runner -cfgfile "$(fixture onstart.yaml)"
    wait_for_log 'ONSTART-YAML'
}

@test "onStart actions run from inline cfgjson" {
    start_runner -cfgjson "$(fixture_json onstart.json)"
    wait_for_log 'ONSTART-JSON'
}

@test "onExec chains one process into another" {
    start_runner -cfgfile "$(fixture onexec-chain.yaml)"
    wait_for_log 'CHAIN-FIRST'
    wait_for_log 'CHAIN-SECOND'
}

@test "onFinish fires after a process exits" {
    start_runner -cfgfile "$(fixture onfinish.yaml)"
    wait_for_log 'QUICK-RAN'
    wait_for_log 'ONFINISH-RAN'
}

@test "onFileCreate fires when the watched path appears" {
    WATCHDIR="${BATS_TEST_TMPDIR}/watched"
    mkdir -p "$WATCHDIR"
    start_runner -cfgfile "$(fixture onfilecreate.yaml)"

    echo hi >"${WATCHDIR}/trigger"
    wait_for_log 'ONFILECREATE-RAN'
}

@test "onFileWrite fires when the watched path changes" {
    WATCHDIR="${BATS_TEST_TMPDIR}/watched"
    mkdir -p "$WATCHDIR"
    echo initial >"${WATCHDIR}/conf"
    start_runner -cfgfile "$(fixture onfilewrite.yaml)"

    echo changed >>"${WATCHDIR}/conf"
    wait_for_log 'ONFILEWRITE-RAN'
}

@test "onFileWrite can restart a running process" {
    WATCHDIR="${BATS_TEST_TMPDIR}/watched"
    mkdir -p "$WATCHDIR"
    echo initial >"${WATCHDIR}/conf"
    start_runner -cfgfile "$(fixture onfilewrite-restart.yaml)"

    retry 25 bash -c "[ -n \"\$(curl -sS 127.0.0.1:${PORT}/get/server | jq -r '.response.pid // empty')\" ]"
    local before
    before="$(pid_of server)"
    [ -n "$before" ]

    echo changed >>"${WATCHDIR}/conf"

    retry 50 bash -c "p=\$(curl -sS 127.0.0.1:${PORT}/get/server | jq -r '.response.pid // empty'); [ -n \"\$p\" ] && [ \"\$p\" != '${before}' ]"
}

@test "an unmatched event key is ignored rather than fatal" {
    start_runner -cfgjson "$(fixture_json unmatched-event.json)" --exec 'ticker -> echo TICKER-RAN'
    wait_for_log 'TICKER-RAN'
    run grep -c 'NEVER-RUNS' "${RUNNER_LOG}"
    [ "$output" = "0" ]
}
