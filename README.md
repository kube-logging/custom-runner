# Custom Runner

A tiny, event-driven process supervisor for container sidecars.

`custom-runner` starts one or more shell commands, watches files for changes, and runs
further actions in response to events — a process finishing, a mounted ConfigMap or Secret
being updated, an HTTP call arriving. Its main use is as a **config-reloader sidecar**:
watch a mounted config file, and restart or signal the main process when it changes.

## Running

Commands run through `sh -c`, so trying this out needs a shell variant, and the
command API binds loopback inside the container — publish it explicitly to reach
it from the host:

```sh
docker build -f Dockerfile.alpine -t custom-runner:alpine .
docker run --rm -p 127.0.0.1:7357:7357 \
  -v "$PWD/examples/config/config.yaml:/config.yaml" \
  custom-runner:alpine -cfgfile /config.yaml -address 0.0.0.0
```

Image variants — pick the base that has the tools your commands need:

| Dockerfile | Base | Use for |
|---|---|---|
| `Dockerfile` | `scratch` | Shipping the binary. **No shell, so it cannot exec anything** — use it as a build stage (`COPY --from`) to drop `/runner` into your own image |
| `Dockerfile.alpine` | `alpine` + `socat` | Signalling another process over a unix socket |
| `Dockerfile.busybox` | `busybox` | A shell and basic coreutils |
| `Dockerfile.node-exporter` | `prometheus/node-exporter` | Running node-exporter with a reloadable config |

Every command runs through `sh -c`, so the `scratch` image is a delivery vehicle
rather than a runtime — which is exactly how logging-operator consumes it. All
images run as uid 65534.

## Flags

| Flag | Default | Description |
|---|---|---|
| `-cfgfile` | _(empty)_ | Path to a YAML config file |
| `-cfgjson` | _(empty)_ | Config as an inline JSON string — handy in a Pod spec, avoids mounting a file |
| `-startup` | _(empty)_ | Shell command to run at startup, registered under the key `startup` |
| `-exec` | _(empty)_ | Register a command as `key -> command`. Repeatable |
| `-address` | `127.0.0.1` | Command API bind address. It executes arbitrary shell — keep it off `0.0.0.0` |
| `-port` | `7357` | Command API port. `0` disables it |
| `-metrics-address` | `0.0.0.0` | Metrics and health bind address |
| `-metrics-port` | `9533` | Metrics and health port. `0` disables it |
| `-debug` | `false` | Debug-level logs, with source locations |
| `-log-format` | `json` | `json` or `text` |

The two listeners are separate on purpose: the command API executes arbitrary
shell, so scraping metrics must not require exposing it. Pods running more than
one runner must give each an explicit `-metrics-port`.

`-cfgfile` and `-cfgjson` can be combined — the JSON is merged over the file.

## Configuration

Config is a map of **events** to the **actions** to run when they fire.

```yaml
events:
  onStart:
    - exec:
        key: nginx
        command: nginx -g 'daemon off;'
  onFileWrite:
    /etc/nginx/nginx.conf:
      - restart:
          key: nginx
```

### Events

Process-lifecycle events are keyed by process key; file events are keyed by watched path.

| Event | Keyed by | Fires when |
|---|---|---|
| `onStart` | _(list)_ | The runner has started |
| `onExit` | _(list)_ | The runner is shutting down, before children are killed |
| `onExec` | process key | A process was started |
| `onFinish` | process key | A process exited |
| `onError` | process key | A process failed to start or exited non-zero |
| `onFileCreate` | file path | The watched path was created |
| `onFileWrite` | file path | The watched path was written |
| `onFileRemove` | file path | The watched path was removed |
| `onFileRename` | file path | The watched path was renamed |
| `onFileChmod` | file path | The watched path's mode changed |

Watches are placed on the **parent directory** of each configured path. Kubernetes
updates a mounted ConfigMap or Secret by swapping the `..data` symlink and never
touches the user-visible filenames, so a `..data` swap fires every path configured
in that directory. Watching `/your/mount/conf` and `/your/mount/..data` both work.

### Actions

Every action takes a `key`; only `exec` uses `command`.

| Action | Effect |
|---|---|
| `exec` | Run `command` under `sh -c`, registered as `key`. Fails if `key` is already running |
| `kill` | Kill the process registered as `key` |
| `restart` | Kill `key`, wait for it to exit, then run it again with its original command |
| `get` | Return the process registered as `key` |
| `list` | Return all running processes |
| `config` | Return the loaded configuration |
| `exit` | Kill every process and shut the runner down |

## HTTP API

Every action is also reachable over HTTP. Mutating verbs are `POST`, reads are `GET`:

| Route | Returns |
|---|---|
| `POST /exec/{key}` | the started process |
| `POST /kill/{key}` | the killed process |
| `POST /restart/{key}` | the replacement process |
| `POST /exit` | shuts the runner down |
| `GET /get/{key}` | one process |
| `GET /list` | every process, sorted by key |
| `GET /config` | the loaded configuration |

```sh
curl -X POST --data 'while true; do date; sleep 1; done' localhost:7357/exec/ticker
curl localhost:7357/list
curl -X POST localhost:7357/restart/ticker
```

For `exec` the **request body is the shell command**, capped at 1 MiB. Responses
share one envelope:

```json
{"success":true,"response":{"key":"ticker","args":["sh","-c","while true; do date; sleep 1; done"],"pid":12}}
```

A wrong method returns `405`, an unknown route `404`. Command failures still
return `200` with `"success": false` and an `error` string.

> [!WARNING]
> The API executes arbitrary shell commands and has no authentication. It binds
> `127.0.0.1` by default for that reason — set `-address 0.0.0.0` only behind a
> NetworkPolicy, or `-port 0` to drive the runner from config alone. Metrics live on
> a separate listener precisely so scraping never requires exposing this one.

## Observability

The metrics listener (`-metrics-port`, default `9533`) serves:

* `GET /metrics` — Prometheus metrics
* `GET /healthz` — liveness. `200` whenever the process is up; a restart cannot fix
  anything this endpoint could report, so it deliberately never fails
* `GET /readyz` — readiness. `503` until startup finishes, and `503` with the
  offending path if a configured watch could not be registered

That last case is the one worth wiring an alert to. If the config volume is not
mounted yet, the watch never registers and the runner stays up looking perfectly
healthy while silently never reloading again. Readiness is what makes that visible.

Metrics are under the `sidecar_reloader` namespace and labelled by `process`:

| Metric | Type | Description |
|---|---|---|
| `sidecar_reloader_config_reloader_requests_total` | counter | Execution attempts |
| `sidecar_reloader_config_reloader_success_reloads_total` | counter | Successful executions |
| `sidecar_reloader_config_reloader_request_errors_total` | counter | Failed executions |
| `sidecar_reloader_config_reloader_last_reload_error` | gauge | `1` if the last execution failed, else `0` |
| `sidecar_reloader_config_reloader_last_request_duration_seconds` | gauge | Duration of the last execution |
| `sidecar_reloader_config_reloader_watcher_errors_total` | counter | Filesystem watcher errors (unlabelled) |

Logs are structured (`log/slog`) and go to **stderr**; the supervised processes' own stdout
and stderr are passed through untouched.

## Examples

See [`examples/`](examples) — a config covering process chaining and file watches, a
node-exporter reload setup, and [`examples/test-alpine.sh`](examples/test-alpine.sh), which
runs the whole thing against a kind cluster with a mounted Secret.

## Development

```sh
make check   # license check, lint, test, e2e
make lint    # golangci-lint
make test    # go test -race ./...
make e2e     # bats end-to-end suite against a real binary
```

`make e2e` builds the runner and drives it over HTTP with [bats](https://github.com/bats-core/bats-core),
pinned in the Makefile and installed into `bin/` on demand. It needs `jq` and `curl`.
[`e2e/compat.bats`](e2e/compat.bats) pins the logging-operator integration contract —
if you change the HTTP surface, that file tells you what breaks.

Upgrading from a pre-1.0 runner: see [MIGRATION.md](MIGRATION.md).

## Contributing

If you find this project useful, help us

* Support the development of this project and star this repo! :star:
* Help new users with issues they may encounter. :muscle:
* Send a pull request with your new features and bug fixes. :rocket:

See [CONTRIBUTING.md](CONTRIBUTING.md) for the DCO sign-off we require on commits.

## License

The project is licensed under the [Apache 2.0 License](LICENSE).
