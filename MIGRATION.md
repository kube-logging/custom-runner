# Migrating to v1.0

v1.0 rewrites the runner's internals and makes a small number of deliberate
breaking changes to the HTTP contract. Everything below was verified end to end
by the suite in [`e2e/`](e2e) — `make e2e`.

## Breaking changes

### 1. Mutating verbs are now POST

Routing moved to the standard library's method-aware `ServeMux`. Previously every
verb answered on any method, so a stray `GET` could start or stop a process.

| Verb | Before | After |
|---|---|---|
| `exec` | any method | `POST /exec/{key}` |
| `kill` | any method | `POST /kill/{key}` |
| `restart` | any method | `POST /restart/{key}` |
| `exit` | any method | `POST /exit` |
| `get` | any method | `GET /get/{key}` |
| `list` | any method | `GET /list` |
| `config` | any method | `GET /config` |

A method mismatch returns `405`; an unknown path returns `404`.

**logging-operator change required** — `images/fluentd-drain-watch/drain-watch.sh:67`:

```diff
-curl --silent --show-error http://$CUSTOM_RUNNER_ADDRESS/exit
+curl --silent --show-error -X POST http://$CUSTOM_RUNNER_ADDRESS/exit
```

The readiness probe on line 27 (`curl -so /dev/null $ADDR`) is unaffected — a bare
`GET /` still answers (404), so curl still exits 0. Prefer `GET /healthz` on the
metrics port going forward.

### 2. Metrics moved to their own listener

`/metrics` is no longer served on the command port. It now has a dedicated
listener controlled by `-metrics-port`, **defaulting to 9533**, alongside
`/healthz` and `/readyz`.

`/readyz` returns `503` naming the path when a configured watch could not be
registered — the failure mode where the sidecar stays up but silently stops
reloading. Worth wiring to the pod's readiness probe.

This closes a real security gap: scraping metrics previously required exposing a
port that also executes arbitrary shell commands.

**This fixes an existing logging-operator bug.** `pkg/resources/model/common.go`
already declares `ConfigReloaderMetricsPort = 9533` and the syslog-ng
ServiceMonitor (`pkg/resources/syslogng/service.go:156`) already scrapes it — but
the runner served metrics on 7357, so nothing was ever collected. The new default
matches what the operator already expects.

> [!WARNING]
> Two runners in the same pod share a network namespace and will now both try to
> bind 9533. Any pod running more than one runner must give each an explicit
> `-metrics-port`, or `-metrics-port 0` to disable. A conflict fails loudly at
> startup with `metrics server: listen tcp :9533: bind: address already in use`.

The syslog-ng buffer-metrics sidecar (`pkg/resources/syslogng/statefulset.go:293`)
passes `--port 7358` and shares a pod with the config-reloader, so it needs a
distinct metrics port.

### 3. The command API binds loopback by default

New `-address` (default `127.0.0.1`) and `-metrics-address` (default `0.0.0.0`).

Previously both listeners bound `0.0.0.0` with no way to change it — while the
README claimed you could bind to localhost. A pod IP is routable from every other
pod in a default cluster with no NetworkPolicy, so **any workload could POST a
shell command to `<pod-ip>:7357/exec/x`** and get execution inside the logging
sidecar, which mounts the output Secrets. The `scratch` image degraded that to DoS
and config disclosure (no `sh`); the alpine, busybox and node-exporter variants
were full RCE.

No logging-operator change needed — `drain-watch.sh` uses `127.0.0.1` and sidecars
share the pod network namespace. Set `-address 0.0.0.0` only if you genuinely need
cross-pod access, and pair it with a NetworkPolicy.

### 4. Process keys are validated

Keys must match `^[A-Za-z0-9._-]{1,64}$`; anything else is `400`. Keys become
Prometheus label values, so unvalidated keys let a caller mint unbounded series —
3000 distinct keys measured at ~12k series and a 915 KB scrape body, permanent.

All operator-supplied keys (`nodeexporter`, `buffersize`, `info`, `reload`) pass.

### 5. Images run as uid 65534

`Dockerfile`, `Dockerfile.alpine` and `Dockerfile.busybox` now set `USER
65534:65534`; the node-exporter variant already inherited `nobody`. Supervised
commands no longer run as root.

This does **not** affect logging-operator, which does `COPY --from=custom-runner
/runner /` into its own images and sets its own user. If you run these images
directly, check that mounted volumes are readable by 65534.

### 6. Single-object responses for single-process verbs

`get`, `exec`, `kill` and `restart` previously wrapped one process in a
single-element array. They now return the object directly. `list` still returns an
array, now sorted by key.

```diff
-{"success":true,"response":[{"key":"nginx","pid":12}]}
+{"success":true,"response":{"key":"nginx","pid":12}}
```

### 7. Config validation is stricter

Three things now fail at startup instead of misbehaving silently:

* A misspelled event name (`onFileWirte`) — previously ignored forever.
* A second YAML document — previously parsed and dropped without a word.
* An action carrying more than one verb (`{exec: ..., kill: ...}`) — `Action` is a
  map, so YAML accepts it, and dispatch then picked one at random per Go's map
  iteration order.

Conversely, an **empty or comment-only config file is now accepted** rather than
crashlooping the sidecar with `parse yaml config: EOF`.

Check any hand-written config before upgrading.

### 8. `/config` returns a typed document

The endpoint used to echo the raw parsed map. It now returns the typed config, so
empty sections are omitted and key order is stable.

## Behaviour changes you may notice

- **Dashboards**: a deliberate `restart` no longer reports as a crash, and a failing
  process no longer increments the success counter. `last_reload_error` used to
  stick at 1 for the life of a supervised daemon.
- **Shutdown**: `SIGTERM`/`SIGINT` now drain the servers, run `onExit`, and kill the
  process group. Previously the runner died and orphaned its children.
- **Reloads**: watching a mounted filename (not just `..data`) now fires on a
  ConfigMap/Secret swap.
- **`onExit` now fires.** It was previously unreachable, so any config using it was
  a no-op.

## Unchanged

Verified by [`e2e/compat.bats`](e2e/compat.bats):

- `-cfgfile`, `-cfgjson`, `-startup`, `-port`, `-debug`, `-log-format`
- `--exec "key -> command"`, repeatable, arrow preserved inside the command
- Single-dash flag spellings (`-cfgjson`) as used by the operator
- The config schema: `events`, all ten event kinds, all seven actions
- `sidecar_reloader_*` metric names and labels
- The `{success, error, response}` envelope
- The binary lives at `/runner` in every image
