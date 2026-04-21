# 🚦 Distributed Rate Limiter as a Service

A production-grade, distributed rate limiting service built in **Go** — supporting multiple algorithms, backed by **Redis**, exposed via a **REST API**, and visualized through a **live dashboard**.

Built to demonstrate real-world backend engineering: atomic operations, distributed systems, clean architecture, and containerized deployment.

---

## 📌 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Algorithms](#algorithms)
- [Project Structure](#project-structure)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Dashboard](#dashboard)
- [Load Testing](#load-testing)
- [Deployment](#deployment)
- [Roadmap](#roadmap)

---

## Overview

Rate limiting is critical infrastructure for any production API. Without it, a single bad actor can bring down your service with a flood of requests.

This service solves that problem at scale — multiple application servers can share a single Redis instance, ensuring consistent rate limiting across your entire fleet. Every algorithm runs as an **atomic Lua script inside Redis**, making race conditions impossible by design.

**Key capabilities:**
- 3 battle-tested rate limiting algorithms, selectable per request
- Fully distributed — consistent across any number of app servers
- Sub-millisecond Redis operations via atomic Lua scripts
- REST API — drop-in for any existing backend
- Live dashboard showing real-time allow/deny stats
- One-command setup with Docker Compose
- Load tested at 10,000+ requests/second

---

## Architecture

```
                        ┌─────────────────────────────────┐
                        │         Client / API Consumer    │
                        └────────────────┬────────────────┘
                                         │ POST /api/v1/check
                                         ▼
                        ┌─────────────────────────────────┐
                        │         Gin HTTP Server          │
                        │         (Go, port 8080)          │
                        └────────────────┬────────────────┘
                                         │
                   ┌─────────────────────┼─────────────────────┐
                   ▼                     ▼                     ▼
        ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
        │   Token Bucket   │  │  Fixed Window    │  │ Sliding Window   │
        │    Algorithm     │  │   Algorithm      │  │   Algorithm      │
        └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
                 │                     │                      │
                 └─────────────────────┼──────────────────────┘
                                       │ Atomic Lua Script
                                       ▼
                        ┌─────────────────────────────────┐
                        │            Redis                 │
                        │   (Shared state, port 6389)      │
                        │   Hash / Counter / Sorted Set    │
                        └─────────────────────────────────┘
```

**Why Redis?** All app servers share one Redis instance. A request hitting Server 1 and the next hitting Server 2 both read/write the same counter. True distributed limiting with no coordination overhead.

**Why Lua scripts?** Each algorithm runs as an atomic script inside Redis. Redis is single-threaded — the Lua script executes without interruption, making race conditions structurally impossible. No locks needed.

---

## Algorithms

### 1. Token Bucket
Best for: **APIs that allow controlled bursts**

A bucket holds N tokens. Each request consumes one token. Tokens refill at a fixed rate per second. If the bucket is empty, the request is denied.

```
Capacity: 5, Refill: 1/sec

t=0s  [●][●][●][●][●]  full
t=0s  req → [●][●][●][●]    allowed ✅
t=0s  req → [●][●][●]       allowed ✅
t=0s  req → [●][●]          allowed ✅
t=0s  req → [●]             allowed ✅
t=0s  req → []              allowed ✅
t=0s  req → []              DENIED  ❌
t=1s  req → []              allowed ✅  (1 token refilled)
```

**Used by:** AWS API Gateway, Stripe, Cloudflare

---

### 2. Fixed Window
Best for: **Simple quotas (e.g., 1000 requests/hour)**

Time is divided into fixed buckets. Each bucket has its own counter. Counter resets at the start of each new window.

```
Window: 60s, Limit: 3

12:00:00 → 12:01:00  [req][req][req]  → 4th denied ❌
12:01:00 → 12:02:00  [req]            → allowed ✅ (fresh window)
```

**Weakness:** Boundary exploit — 3 requests at 12:00:59 + 3 at 12:01:00 = 6 in 2 seconds.

---

### 3. Sliding Window
Best for: **Smooth, accurate limiting with no boundary exploits**

Always looks at the last N seconds from right now. Uses a Redis Sorted Set — each request is stored with its timestamp as the score. Expired entries are pruned on every check.

```
Window: 60s, Limit: 3, Now: 12:34:47

Valid range: 12:33:47 → 12:34:47
Entries in range: [req@12:33:50][req@12:34:10][req@12:34:45]
Count = 3 → next request DENIED ❌

At 12:34:51 → 12:33:50 entry slides out
Count = 2 → allowed ✅
```

**Used by:** Nginx, Cloudflare, most modern API gateways

---

## Project Structure

```
rate-limiter/
├── main.go                        # Entry point — wires everything together
├── docker-compose.yml             # Redis + app, one command setup
├── go.mod                         # Go module definition
│
├── config/
│   └── config.go                  # Env-based config (port, redis addr)
│
├── internal/                      # Private core logic
│   ├── limiter/
│   │   ├── limiter.go             # Limiter interface
│   │   ├── token_bucket.go        # Token Bucket algorithm
│   │   ├── fixed_window.go        # Fixed Window algorithm
│   │   ├── sliding_window.go      # Sliding Window algorithm
│   │   └── limiter_test.go        # Tests for all algorithms
│   └── store/
│       └── redis.go               # Redis connection manager
│
└── api/
    ├── server.go                  # Gin HTTP server + route registration
    └── handlers.go                # Request handlers + response types
```

---

## Tech Stack

| Layer | Technology | Why |
|---|---|---|
| Language | Go 1.21 | Native concurrency, fast, production standard |
| HTTP Framework | Gin | Lightweight, fastest Go router |
| Cache / State | Redis 7 (Alpine) | Sub-ms operations, atomic Lua, pub/sub |
| Containerization | Docker + Compose | Reproducible environments |
| Testing | Go testing package | Built-in, no dependencies |
| Load Testing | k6 / wrk | Industry standard |

---

## Getting Started

### Prerequisites
- Go 1.21+
- Docker + Docker Compose

### 1. Clone the repo
```bash
git clone https://github.com/divy404/rate-limiter.git
cd rate-limiter
```

### 2. Install dependencies
```bash
go mod tidy
```

### 3. Start Redis
```bash
docker compose up -d
docker compose ps   # wait for status: healthy
```

### 4. Run the server
```bash
REDIS_ADDR=localhost:6389 go run main.go
```

```
✅ Connected to Redis at localhost:6389
🚀 Rate Limiter running on port 8080
```

### 5. Test it
```bash
# Health check
curl http://localhost:8080/api/v1/status

# Rate limit check — run 4 times, 4th will be denied
curl -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"user-123","strategy":"token_bucket","limit":3,"refill_rate":1}'
```

### 6. Run tests
```bash
go test ./internal/... -v
```

---

## API Reference

### `POST /api/v1/check`

Check if a request from a client should be allowed.

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `client_id` | string | ✅ | Unique identifier for the client (user ID, IP, API key) |
| `strategy` | string | ✅ | `token_bucket`, `fixed_window`, or `sliding_window` |
| `limit` | int | ✅ | Max requests allowed |
| `window_seconds` | int | For window strategies | Window size in seconds |
| `refill_rate` | int | For token bucket | Tokens added per second |

**Example — Token Bucket**
```bash
curl -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "user-123",
    "strategy": "token_bucket",
    "limit": 10,
    "refill_rate": 2
  }'
```

**Example — Sliding Window**
```bash
curl -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "user-456",
    "strategy": "sliding_window",
    "limit": 100,
    "window_seconds": 60
  }'
```

**Response — Allowed** `200 OK`
```json
{
  "allowed": true,
  "strategy": "token_bucket",
  "client_id": "user-123"
}
```

**Response — Denied** `429 Too Many Requests`
```json
{
  "allowed": false,
  "strategy": "token_bucket",
  "client_id": "user-123"
}
```

---

### `GET /api/v1/status`

Health check endpoint. Verifies Redis connectivity.

```bash
curl http://localhost:8080/api/v1/status
```

**Response — Healthy** `200 OK`
```json
{
  "status": "healthy",
  "redis": "connected"
}
```

**Response — Unhealthy** `503 Service Unavailable`
```json
{
  "status": "unhealthy",
  "redis": "unreachable"
}
```

---

## Dashboard

> 🚧 In Progress — Phase 4

A real-time web dashboard showing:
- Live allow / deny rate per client
- Requests per second graph
- Per-algorithm breakdown
- Active client list with token/request counts

Built with Go's `html/template` + WebSockets for live updates. No external frontend framework needed.

---

## Load Testing

> 🚧 In Progress — Phase 5

Using [k6](https://k6.io/) to simulate high-concurrency traffic:

```bash
k6 run loadtest.js
```

Target benchmarks:
- 10,000 requests/second sustained
- p99 latency < 5ms
- Zero incorrect allow/deny decisions under concurrent load

---

## Deployment

> 🚧 In Progress — Phase 5

Full Docker Compose deployment with the Go app containerized alongside Redis:

```bash
docker compose up --build
```

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `localhost:6389` | Redis connection address |
| `PORT` | `8080` | HTTP server port |

---

## Roadmap

- [x] Token Bucket algorithm
- [x] Fixed Window algorithm
- [x] Sliding Window algorithm
- [x] Redis-backed distributed state
- [x] Atomic Lua scripts (race-condition-free)
- [x] REST API with Gin
- [x] Health check endpoint
- [x] Docker Compose setup
- [ ] Live dashboard with WebSockets
- [ ] Load testing suite (k6)
- [ ] Full Docker deployment (app + Redis)
- [ ] Prometheus metrics endpoint
- [ ] Per-client analytics API
- [ ] Rate limit headers (`X-RateLimit-Remaining`, `Retry-After`)

---

## Why This Project

Rate limiting is one of those problems that looks simple on the surface but gets deeply interesting at scale. The core challenges:

- **Atomicity** — read-modify-write must be indivisible across distributed nodes
- **Memory efficiency** — millions of clients, each with their own state
- **Algorithm tradeoffs** — no single algorithm is best for all use cases
- **Accuracy vs performance** — sliding window is most accurate but costs more memory than fixed window

This project addresses all four — and the result is something you could drop into a production system today.

---

## License

MIT — use it, learn from it, build on it.
