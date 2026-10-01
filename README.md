# Resilient Cloud-Native API Gateway & Distributed Rate Limiter

A production-grade, cloud-native API Gateway built with Go, Redis, Prometheus, and Kubernetes. Designed to protect backend microservices from burst traffic via distributed token-bucket rate limiting, decoupled self-healing probes, and automated horizontal autoscaling.

---

## Architecture Overview

```text
                            [ Client Traffic ]
                                    │
                                    ▼
                      [ Kubernetes Ingress / Service ]
                                    │
                ┌───────────────────┴───────────────────┐
                ▼                                       ▼
        ┌───────────────────┐                   ┌───────────────────┐
        │  Pod Replica 1    │                   │  Pod Replica 2    │
        │ ┌───────────────┐ │                   │ ┌───────────────┐ │
        │ │ Init Container│ │                   │ │ Init Container│ │
        │ │ (Wait-for-DB) │ │                   │ │ (Wait-for-DB) │ │
        │ └───────┬───────┘ │                   │ └───────┬───────┘ │
        │         ▼         │                   │         ▼         │
        │ ┌───────────────┐ │                   │ ┌───────────────┐ │
        │ │   Go App      │◀┼──(Shared Volume)──┼▶│   Go App      │ │
        │ │ Rate Limiter  │ │                   │ │ Rate Limiter  │ │
        │ └───────┬───────┘ │                   │ └───────┬───────┘ │
        │         ▼         │                   │         ▼         │
        │ ┌───────────────┐ │                   │ ┌───────────────┐ │
        │ │Sidecar (Logs) │ │                   │ │Sidecar (Logs) │ │
        │ └───────────────┘ │                   │ └───────────────┘ │
        └─────────┬─────────┘                   └─────────┬─────────┘
                  │                                       │
                  └───────────────────┬───────────────────┘
                                      │
                                      ▼
                             [ Redis Cluster/Pod ]
                           (Central Token Bucket State)
```

---

## Key Features

* **Distributed Sliding Window Rate Limiting:** Enforces strict limits across multi-replica deployments using atomic Redis pipelines (`ZREMRANGEBYSCORE`, `ZADD`, `ZCARD`, `EXPIRE`) to prevent race conditions.
* **Standards-Compliant Headers:** Injects `X-RateLimit-Remaining`, `X-RateLimit-Reset`, and `Retry-After` headers on throttled requests (`HTTP 429`).
* **Zero-Downtime Lifecycle Management:** Intercepts `SIGTERM` and `SIGINT` signals with a 15-second context timeout drain alongside a Kubernetes `preStop` hook to prevent dropped packets during rolling updates.
* **Decoupled Health Probes:** 
  * `/livez`: Liveness check verifying the HTTP process loop.
  * `/readyz`: Readiness check confirming upstream Redis connectivity before receiving cluster traffic.
* **Full Prometheus Observability:** Instruments request counts, path-based latencies, and rate-limit drops via `/metrics`.
* **Multi-Stage Distroless Container:** Compiles to a statically linked, non-root binary running on `gcr.io/distroless/static-debian12` for a minimal attack surface (~29MB).
* **Kubernetes Patterns:**
  * **Init Container:** Validates Redis TCP availability before app launch.
  * **Sidecar Container:** Tails application logs via shared `emptyDir` volumes.
  * **HPA (Horizontal Pod Autoscaling):** Automatically scales replicas (2 to 5) based on real-time CPU targets.

---

## Directory Structure

```text
resilient-gateway/
├── cmd/
│   └── server/
│       └── main.go          # Dependency injection, router setup, signal traps
├── internal/
│   ├── config/
│   │   └── config.go        # Environment-driven configuration loader
│   ├── handler/
│   │   ├── api.go           # Protected business API endpoints
│   │   └── health.go        # /livez and /readyz probes
│   ├── limiter/
│   │   └── limiter.go       # Redis atomic sliding-window rate limiter
│   └── middleware/
│       ├── metrics.go       # Prometheus instrumentation middleware
│       └── rate_limit.go    # HTTP rate limiting middleware
├── k8s/
│   ├── configmap.yaml       # Runtime configurations
│   ├── deployment.yaml      # Pod spec (Init container, Sidecar, Probes, PreStop)
│   ├── hpa.yaml             # Horizontal Pod Autoscaler manifest
│   ├── redis.yaml           # In-cluster Redis deployment & service
│   └── service.yaml         # NodePort/ClusterIP service
├── Dockerfile               # Multi-stage distroless build
├── docker-compose.yaml      # Local multi-container verification
├── go.mod
└── go.sum
```

---

## Local Development (Docker Compose)

Start the gateway and Redis services locally:

```bash
docker compose up --build -d
```

Verify endpoints:

```bash
# Check probe endpoints
curl -i http://localhost:8080/livez
curl -i http://localhost:8080/readyz

# Test rate limiting (configured to 5 requests per 10s window)
for i in {1..6}; do curl -i http://localhost:8080/api/v1/resource; echo ""; done

# Scrape metrics
curl http://localhost:8080/metrics
```

---

## Kubernetes Deployment

### 1. Apply Manifests

```bash
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml
```

### 2. Verify Pod Health

```bash
kubectl get pods -w
```
Wait until pods show `2/2` containers in the `Running` state (indicating the init container succeeded, and both the gateway and sidecar are active).

### 3. Port-Forward Traffic

```bash
kubectl port-forward svc/resilient-gateway-service 8080:8080
```

### 4. Monitor Autoscaling (HPA)

```bash
kubectl get hpa -w
```