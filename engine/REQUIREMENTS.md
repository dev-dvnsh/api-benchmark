# Requirements: Benchmark Engine (`engine/`)

## 1. Objective

Build a standalone Go HTTP microservice running on port `8000` that executes concurrent HTTP load tests against a target URL and returns latency and throughput statistics.

## 2. Tech Stack

- Go 1.22+
- **Standard library only** (`net/http`, `encoding/json`, `sync`, `time`, `sort`, `io`, `math`). Zero third-party packages.

## 3. Endpoint Specification

### `POST /run`

Executes a burst of HTTP requests against the specified target.

**Expected Request Body (JSON):**

- `url` (string, required): Valid HTTP/HTTPS URL to test.
- `method` (string, default `"GET"`): Allowed values: `GET`, `POST`, `PUT`, `DELETE`.
- `headers` (map of string to string, optional): Headers to attach to every outgoing request.
- `body` (raw JSON or string, optional): Request payload for `POST`/`PUT`.
- `totalRequests` (int, required): Minimum `1`, maximum `500` (safety cap).
- `concurrency` (int, required): Minimum `1`, maximum `50` (cannot exceed `totalRequests`).

**Expected Response Body (JSON):**

- `totalRequests` (int): Total requests attempted.
- `totalDurationMs` (float64): Wall-clock time from first request start to last request finish.
- `avgMs` (float64): Average latency per request in milliseconds.
- `minMs` (float64): Fastest single request in milliseconds.
- `maxMs` (float64): Slowest single request in milliseconds.
- `p95Ms` (float64): 95th percentile latency in milliseconds.
- `rps` (float64): Requests per second (`totalRequests / (totalDurationMs / 1000)`).
- `errorRate` (float64): Percentage of requests that failed (network error, timeout, or 4xx/5xx status).
- `statusCodes` (map of string to int): Count of each status code received (e.g., `"200": 95`, `"500": 3`, `"ERROR": 2`).

## 4. Edge Cases to Handle

- **Method enforcement:** Reject non-POST requests to `/run` with `405 Method Not Allowed`.
- **Input validation:** Reject malformed JSON, missing URL, invalid HTTP methods, or out-of-bounds `totalRequests`/`concurrency` with `400 Bad Request` and a JSON error message.
- **Individual request failures:** A timeout, DNS error, or connection refusal on one request must NOT crash the server or abort the benchmark. Catch it, record it under `statusCodes` as `"ERROR"` (or `"TIMEOUT"`), and continue.
- **Per-request timeout:** Enforce a 10-second timeout on outgoing requests so a hanging target cannot lock up a worker forever.

## 5. Definition of Done

- `go run .` starts cleanly on port `8000` with zero external dependencies.
- Sending a `curl` `POST` request to `http://127.0.0.1:8000/run` targeting a live URL returns accurate, rounded JSON metrics.
- Sending a request to a dead local port returns `200 OK` with a valid JSON report showing `errorRate: 100`.
