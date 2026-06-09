# Kubernetes deployment

Manifests to run [Request Debugger](../README.md) in Kubernetes. The container listens on port **5464**, writes each exchange to **stdout and a file inside the pod** (`LOG_PATH`, default `/tmp/requestHeadersQueryParamsAndBody.log`), and shuts down gracefully on SIGTERM.

## Layout

```
kubernetes/
├── namespace.yaml
├── configmap.yaml
├── deployment.yaml
├── service.yaml
├── service-nodeport.yaml   # optional
├── ingress.yaml            # optional
├── pdb.yaml
└── kustomization.yaml
```

## Prerequisites

- Kubernetes 1.24+
- `kubectl` configured for your cluster
- Container image (default: `masteralt/requestdebugger:latest`)

## Deploy

```bash
kubectl apply -k kubernetes/
```

Verify:

```bash
kubectl -n requestdebugger get pods,svc
kubectl -n requestdebugger wait --for=condition=ready pod -l app.kubernetes.io/name=requestdebugger --timeout=60s
```

Port-forward and test:

```bash
kubectl -n requestdebugger port-forward svc/requestdebugger 5464:5464

curl "http://127.0.0.1:5464/?key=value" \
  -H 'Header1: value1' \
  -d '{"dataKey":"dataValue"}'
```

## Viewing logs

Exchange logs are written to **stdout** (for `kubectl logs`) and to the **in-pod log file**.

```bash
# Stream exchange logs from stdout
kubectl -n requestdebugger logs -f deploy/requestdebugger
```

For JSON format in Kubernetes, set `LOG_FORMAT=json` in the ConfigMap or deployment args:

```yaml
args:
  - -log-level
  - debug
  - -log-format
  - json
```

Each request produces one JSON line on stdout (ideal for log aggregators).

The same content is appended to `LOG_PATH` inside the pod. Logs are ephemeral and are lost when the pod is removed (no PVC).

## Configuration

### ConfigMap

| Key | Default | Description |
|-----|---------|-------------|
| `PORT` | `5464` | Listen port |
| `LOG_PATH` | `/tmp/requestHeadersQueryParamsAndBody.log` | In-pod exchange log file |
| `LOG_LEVEL` | `debug` | `error`, `debug`, or `trace` |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `REQUESTDEBUGGER_URL` | `""` | Base URL for curl replay |

```bash
kubectl -n requestdebugger edit configmap requestdebugger
kubectl -n requestdebugger rollout restart deployment requestdebugger
```

### Deployment args

```yaml
args:
  - -log-level
  - trace
  - -log-format
  - json
  # - -curl
```

CLI flags override ConfigMap env vars when both are set.

## Runtime APIs

```bash
kubectl -n requestdebugger port-forward svc/requestdebugger 5464:5464 &

curl -X PUT http://127.0.0.1:5464/_config/curl \
  -H 'Content-Type: application/json' \
  -d '{"enabled": true}'

curl -X PUT http://127.0.0.1:5464/_config/log-level \
  -H 'Content-Type: application/json' \
  -d '{"level": "trace"}'
```

## Exposing the service

**ClusterIP (default):** `http://requestdebugger.requestdebugger.svc.cluster.local:5464`

**NodePort:**

```bash
kubectl apply -f kubernetes/service-nodeport.yaml
curl http://<node-ip>:30464/
```

**Ingress:** edit `ingress.yaml` host/class, then `kubectl apply -f kubernetes/ingress.yaml`

## Graceful shutdown

`terminationGracePeriodSeconds: 15` — the app drains in-flight requests for up to 10 seconds on SIGTERM.

```bash
kubectl -n requestdebugger delete pod -l app.kubernetes.io/name=requestdebugger
```

## Uninstall

```bash
kubectl delete -k kubernetes/
```

See the [main README](../README.md) for log level behaviour, text vs JSON output examples, and curl configuration.
