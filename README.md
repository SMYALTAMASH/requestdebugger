# Request Debugger

An HTTP echo server that captures full request/response exchanges to a log file for debugging. Point traffic at it, inspect what arrived, and optionally copy a ready-made curl command to replay the call elsewhere.

By default the server echoes the request body back to the client. Each captured exchange is written to **stdout** and appended to the log file at `LOG_PATH` (default `/tmp/requestHeadersQueryParamsAndBody.log`).

---

## Quick start

### Docker

```bash
docker run -d \
  -v /tmp:/tmp \
  -p 1111:5464 \
  --name requestdebugger \
  masteralt/requestdebugger:latest
```

Send a test request:

```bash
curl "http://127.0.0.1:1111/?size=8192&firstkey=firstvalue%40123" \
  -H 'Header1: value1' \
  -d '{"dataKey":"dataValue"}'
```

The curl response body (echoed request body):

```json
{"dataKey":"dataValue"}
```

Tail the log file on the host:

```bash
tail -f /tmp/requestHeadersQueryParamsAndBody.log
```

Stop the container gracefully (sends SIGTERM, drains in-flight requests, then exits):

```bash
docker stop requestdebugger
docker rm requestdebugger
```

### Build and run from source

```bash
git clone git@github.com:SMYALTAMASH/requestdebugger.git
cd requestdebugger

go build -o requestdebugger .

./requestdebugger
```

The server listens on port **5464** by default.

---

## Configuration reference

Settings can be passed as **CLI flags**, **environment variables**, or changed at **runtime via HTTP APIs**.

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-curl` | `false` | Include a replay curl command in exchange logs |
| `-log-level` | (see env) | Log verbosity: `error`, `debug`, or `trace` |
| `-log-format` | (see env) | Exchange log format: `text` or `json` |

Examples:

```bash
# Full detail in logs + curl commands (text format)
./requestdebugger -log-level trace -curl

# JSON lines on stdout and in the log file (good for log aggregators)
./requestdebugger -log-format json -log-level debug

# Only log failed requests (HTTP 4xx/5xx)
./requestdebugger -log-level error
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `5464` | Port the server listens on |
| `LOG_PATH` | `/tmp/requestHeadersQueryParamsAndBody.log` | In-process log file path (same content as stdout) |
| `LOG_LEVEL` | `debug` | Log verbosity: `error`, `debug`, or `trace` |
| `LOG_FORMAT` | `text` | Exchange log format: `text` or `json` |
| `REQUESTDEBUGGER_URL` | (empty) | Base URL used in curl commands; overrides the `Requestdebugger_url` request header |

**Precedence:** CLI flags override environment variables where both apply (e.g. `-log-level` wins over `LOG_LEVEL`).

Each exchange is written to **stdout and `LOG_PATH`** using the same formatted output.

Docker example with all common options:

```bash
docker run -d \
  -v /tmp:/tmp \
  -p 1111:5464 \
  -e LOG_LEVEL=trace \
  -e LOG_FORMAT=json \
  -e REQUESTDEBUGGER_URL=http://api.example.com \
  -e LOG_PATH=/tmp/requestHeadersQueryParamsAndBody.log \
  --name requestdebugger \
  masteralt/requestdebugger:latest
```

To enable curl generation in Docker without rebuilding, use the runtime API below or pass the flag at entrypoint:

```bash
docker run -d \
  -v /tmp:/tmp \
  -p 1111:5464 \
  --entrypoint /app/main \
  masteralt/requestdebugger:latest \
  -curl -log-level debug -log-format text
```

---

## Runtime configuration APIs

These endpoints let you change behaviour without restarting the container.

### Enable or disable curl generation — `/_config/curl`

**Get current setting:**

```bash
curl http://127.0.0.1:5464/_config/curl
```

Response:

```json
{"curl_enabled":false}
```

**Enable curl generation:**

```bash
curl -X PUT http://127.0.0.1:5464/_config/curl \
  -H 'Content-Type: application/json' \
  -d '{"enabled": true}'
```

Response:

```json
{"curl_enabled":true}
```

`POST` is also supported with the same JSON body. Curl commands are **off by default** and are not generated for `/_config/*` routes.

### Change log level — `/_config/log-level`

**Get current level:**

```bash
curl http://127.0.0.1:5464/_config/log-level
```

Response:

```json
{"level":"debug"}
```

**Change level at runtime:**

```bash
curl -X PUT http://127.0.0.1:5464/_config/log-level \
  -H 'Content-Type: application/json' \
  -d '{"level": "trace"}'
```

Response:

```json
{"level":"trace"}
```

Valid values: `error`, `debug`, `trace`.

---

## Log levels

Log levels control **which exchanges** are logged and how much detail each entry contains. The same formatted entry is written to stdout and the log file.

### Exchange logging

| Level | What gets logged |
|-------|------------------|
| `error` | Only requests that return HTTP **400+**. Failed requests use **trace** detail (full headers). |
| `debug` | Every request: timestamp, method, URL, request/response bodies, query params, status. **No headers.** |
| `trace` | Everything in `debug` **plus** request and response headers. |

### Operational stdout (`[ERROR]` / `[DEBUG]`)

Startup, shutdown, config changes, and internal errors are logged separately with `[ERROR]` or `[DEBUG]` prefixes. These are not exchange logs.

---

## Log format (`text` vs `json`)

| Format | Output |
|--------|--------|
| `text` | Multi-line block (human-readable, same as before) |
| `json` | Single JSON object per exchange (one line, ideal for Loki/ELK/CloudWatch) |

```bash
./requestdebugger -log-format json -log-level trace -curl
```

**JSON example** (one line on stdout and in the log file):

```json
{
  "timestamp": "2026-06-09T14:32:01.123456789Z",
  "log_level": "debug",
  "request": {
    "method": "POST",
    "url": "/",
    "body": "{\"dataKey\":\"dataValue\"}",
    "query_params": {
      "firstkey": ["firstvalue@123"],
      "size": ["8192"]
    }
  },
  "response": {
    "status": 200,
    "body": "{\"dataKey\":\"dataValue\"}"
  },
  "curl_command": "curl -XPOST 'http://api.example.com/?firstkey=firstvalue%40123&size=8192' ..."
}
```

At `trace` level, `request.headers` and `response.headers` maps are included. `curl_command` is omitted unless curl generation is enabled.

---

## `Requestdebugger_url` header

Use this header (or the `REQUESTDEBUGGER_URL` env var) to control the host in generated curl commands instead of the `{{host}}` placeholder.

**Priority:**

1. `REQUESTDEBUGGER_URL` environment variable (overrides any incoming header)
2. `Requestdebugger_url` request header
3. `{{host}}` placeholder (manual replacement required)

Example with env var:

```bash
export REQUESTDEBUGGER_URL=http://api.example.com

curl http://127.0.0.1:5464/orders \
  -H 'Requestdebugger_url: http://old-host.com' \
  -H 'Authorization: Bearer token' \
  -d '{"id":1}'
```

Even though the request sends `http://old-host.com`, the log and curl command use `http://api.example.com` because the env var wins.

Example with header only (no env var):

```bash
curl http://127.0.0.1:5464/orders \
  -H 'Requestdebugger_url: http://staging.example.com' \
  -d '{"id":1}'
```

Generated curl target:

```bash
curl -XPOST 'http://staging.example.com/orders' ...
```

---

## Example requests and expected output

Use this curl command throughout the examples:

```bash
curl "http://127.0.0.1:5464/?size=8192&firstkey=firstvalue%40123" \
  -H 'Header1: value1' \
  -H 'Content-Type: application/json' \
  -d '{"dataKey":"dataValue"}'
```

**HTTP response from the server** (body is echoed back):

```json
{"dataKey":"dataValue"}
```

---

### Output at `debug` level (default)

```bash
./requestdebugger -log-level debug
# or: export LOG_LEVEL=debug
```

**Log file and stdout** (same content):

```
###################################################################
TIMESTAMP:     2026-06-09T14:32:01.123456789Z
LOG LEVEL:     debug
---------- REQUEST ----------
HTTP Method:   POST
REQUEST URL:   /
REQUEST BODY:  {"dataKey":"dataValue"}
Query Param:   firstkey = firstvalue@123
Query Param:   size = 8192
---------- RESPONSE ----------
HTTP Status:   200
RESPONSE BODY: {"dataKey":"dataValue"}
###################################################################
```

Curl commands are **not** included unless `-curl` is set or enabled via `/_config/curl`.

---

### Output at `trace` level

```bash
./requestdebugger -log-level trace
```

Same as `debug`, but headers are included:

```
###################################################################
TIMESTAMP:     2026-06-09T14:32:01.123456789Z
LOG LEVEL:     trace
---------- REQUEST ----------
HTTP Method:   POST
REQUEST URL:   /
REQUEST BODY:  {"dataKey":"dataValue"}
Query Param:   firstkey = firstvalue@123
Query Param:   size = 8192
REQUEST HEADER: Accept = */*
REQUEST HEADER: Content-Length = 23
REQUEST HEADER: Content-Type = application/json
REQUEST HEADER: Header1 = value1
REQUEST HEADER: User-Agent = curl/8.x
---------- RESPONSE ----------
HTTP Status:   200
RESPONSE BODY: {"dataKey":"dataValue"}
RESPONSE HEADER: Content-Length = 23
RESPONSE HEADER: Content-Type = text/plain; charset=utf-8
###################################################################
```

**Container stdout** at trace level includes the same multi-line exchange block plus any `[DEBUG]` operational lines. There is no separate `[TRACE]` summary line per request.

---

### Output at `error` level

```bash
./requestdebugger -log-level error
```

Successful requests (HTTP 2xx/3xx) are **not** written to the log file.

Failed requests (HTTP 400+) are logged with full trace detail. Example after sending a bad config request:

```bash
curl -X PUT http://127.0.0.1:5464/_config/log-level \
  -H 'Content-Type: application/json' \
  -d '{"level": "invalid"}'
```

Log file entry (note `LOG LEVEL: trace` used for error-detail dumps):

```
###################################################################
TIMESTAMP:     2026-06-09T14:35:00.987654321Z
LOG LEVEL:     trace
---------- REQUEST ----------
HTTP Method:   PUT
REQUEST URL:   /_config/log-level
REQUEST BODY:  {"level": "invalid"}
REQUEST HEADER: Content-Type = application/json
...
---------- RESPONSE ----------
HTTP Status:   400
RESPONSE BODY: invalid log level "invalid" (expected error, debug, or trace)
...
###################################################################
```

---

### Output with curl generation enabled

```bash
./requestdebugger -log-level debug -curl
```

Or enable at runtime:

```bash
curl -X PUT http://127.0.0.1:5464/_config/curl \
  -H 'Content-Type: application/json' \
  -d '{"enabled": true}'
```

Log file includes a replay command:

```
...
CURL COMMAND:
curl -XPOST '{{host}}/?firstkey=firstvalue%40123&size=8192' \
-H 'Accept: */*' \
-H 'Content-Type: application/json' \
-H 'Header1: value1' \
-H 'User-Agent: curl/8.x' \
--data-urlencode '{"dataKey":"dataValue"}'

---------- RESPONSE ----------
...
```

With `REQUESTDEBUGGER_URL=http://127.0.0.1:5464`:

```
CURL COMMAND:
curl -XPOST 'http://127.0.0.1:5464/?firstkey=firstvalue%40123&size=8192' \
-H 'Accept: */*' \
...
```

The `Requestdebugger_url` header is omitted from `-H` lines when it is used as the curl URL.

---

## Prebuilt binaries

Prebuilt binaries for Windows, macOS, and Linux are published under `RequestDebuggerBinariesForAllOS/` in releases.

```bash
# Download and unzip the latest release, then:
chmod +x requestDebugger-linux-amd64
./requestDebugger-linux-amd64 -log-level trace -curl
```

Supported architectures: `darwin-amd64`, `linux-386`, `linux-amd64`, `linux-arm`, `linux-arm64`, `windows-386`, `windows-amd64`.

---

## Graceful shutdown

The server handles **SIGTERM** and **SIGINT** (e.g. `docker stop`, Ctrl+C):

1. Stops accepting new connections
2. Waits up to **10 seconds** for in-flight requests to finish
3. Exits cleanly

```bash
docker stop requestdebugger          # default 10s grace period
docker stop -t 30 requestdebugger    # extend grace period if needed
```

> **Note:** SIGKILL (`kill -9`) cannot be caught; the process is terminated immediately with no cleanup.

---

## Kubernetes

Manifests live under [`kubernetes/`](kubernetes/). Deploy with Kustomize:

```bash
kubectl apply -k kubernetes/
kubectl -n requestdebugger port-forward svc/requestdebugger 5464:5464

# Exchange logs stream to stdout (and to /tmp inside the pod)
kubectl -n requestdebugger logs -f deploy/requestdebugger
```

Set JSON format in the deployment or ConfigMap:

```yaml
env:
  - name: LOG_FORMAT
    value: "json"
```

Included resources: `namespace`, `configmap`, `deployment`, `service`, optional `service-nodeport`, `ingress`, `pdb`. No PVC — logs are ephemeral inside the pod; use stdout for aggregation.

See [`kubernetes/README.md`](kubernetes/README.md) for full deployment details.

---

## Typical workflow

1. Start the server or container with the desired `-log-level`, `-log-format`, and env vars.
2. Point your application or curl at the debugger URL.
3. Tail stdout or the log file: `kubectl logs -f ...` or `tail -f /tmp/requestHeadersQueryParamsAndBody.log`
4. Optionally enable curl generation via `/_config/curl` and copy the curl block from the log.
5. Set `REQUESTDEBUGGER_URL` (or pass `Requestdebugger_url`) so curl commands target your real upstream host.
6. Stop the container with `docker stop` when done.
