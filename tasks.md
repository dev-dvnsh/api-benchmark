# API Load & Benchmark Tester — Detailed Build Roadmap (`tasks.md`)

**Workflow Rule:** Work strictly top-to-bottom. Do not start the next task until the **Done When** check of the current task passes.

---

## Phase 0: Workspace Setup

### Task 0.1: Repository & Environment Skeleton

- **Problem Statement:** Keep all three runtimes (Go, Node, React) isolated so dependencies and build artifacts never collide.
- **What You Need to Do:**
  1. Create `api-benchmarker/` with three subfolders: `engine/`, `api/`, and `client/`.
  2. Initialize Git at the root and create a `.gitignore` ignoring `node_modules/`, `.env`, `dist/`, and compiled Go binaries.
  3. Inside `engine/`, initialize a Go module (`go mod init engine`).
  4. Verify MongoDB is running locally (or your connection URI is ready).
- **Done When:** `go version`, `node -v`, and your MongoDB connection all work, and `engine/go.mod` exists.

---

## Phase 1: `engine/` (Go Load-Testing Microservice)

> **Primary Go Learning Resources:** Keep **gobyexample.com** and **pkg.go.dev** open in a browser tab. Every single concept below has a 2-minute page on Go by Example.

### Task 1.1: HTTP Server, Routing & JSON Structs

- **Problem Statement:** Before running any load tests, Go needs an HTTP server that can decode incoming JSON into strongly typed structs, validate them, and send JSON back.
- **What to Learn First (15 mins):**
  - _Go by Example:_ **HTTP Servers**, **JSON** (specifically struct tags like `json:"totalRequests"`).
  - _Docs:_ `net/http` (`http.HandleFunc`, `http.ListenAndServe`, `http.ResponseWriter`, `*http.Request`) and `encoding/json` (`json.NewDecoder`, `json.NewEncoder`).
- **How It Works Conceptually:**
  - Unlike Express, Go doesn't parse JSON automatically into `req.body`. You define a `struct` representing the request shape, attach JSON tags to its fields, and stream `r.Body` into that struct using a JSON decoder.
- **What You Need to Do:**
  1. Create `main.go` and register a handler for `/run` on port `8000`.
  2. Define your `RunRequest` struct matching `engine/REQUIREMENTS.md`.
  3. Inside the handler: check that `r.Method` is `POST`, decode the JSON body into `RunRequest`, and validate all fields (`url` non-empty, valid `method`, `totalRequests` between 1–500, `concurrency` between 1–50 and not greater than `totalRequests`).
  4. If validation fails, set the `Content-Type: application/json` header, write status `400`, and return a JSON error object.
  5. If valid, return a temporary dummy JSON response confirming receipt.
- **Done When:** `curl` with valid JSON returns `200`; `curl` with `totalRequests: 9999` or `GET /run` returns `400` / `405` cleanly.

### Task 1.2: Single-Request Execution & Connection Reuse

- **Problem Statement:** You need a clean function that fires one HTTP request, measures its exact latency, enforces a timeout, and never leaks open network connections.
- **What to Learn First (15 mins):**
  - _Docs:_ `net/http` (`http.Client`, `http.Transport`, `http.NewRequest`), `time` (`time.Now`, `time.Since`), `io` (`io.Copy`, `io.Discard`), and `defer`.
- **How It Works Conceptually:**
  - Never use `http.Get()` in production—its default client has **no timeout**. Instead, create a custom `http.Client` with a `Timeout` (e.g., 10 seconds) and a custom `http.Transport` configured to allow enough idle connections per host for your concurrency level.
  - **Crucial Go gotcha:** Even if you don't care about the response body, you MUST drain it (`io.Copy(io.Discard, resp.Body)`) and close it (`defer resp.Body.Close()`). If you skip draining, Go cannot reuse the TCP connection.
- **What You Need to Do:**
  1. Create a second file `runner.go` (same `package main`).
  2. Define a `RequestResult` struct holding: `DurationMs` (float64), `StatusCode` (int), and `Err` (error).
  3. Write a helper function that takes an `*http.Client` and your `RunRequest`, constructs an `*http.Request` (attaching method, URL, headers, and body if present), records `start := time.Now()`, executes `client.Do(req)`, and calculates elapsed milliseconds.
  4. If `client.Do(req)` returns an error, return a `RequestResult` with `Err` populated instead of crashing. Otherwise, drain and close `resp.Body` and return the status code and duration.
- **Done When:** Calling this helper once from your `/run` handler against a live URL prints/returns its real status code and latency in milliseconds, and calling it against a dead port returns an error cleanly.

### Task 1.3: Concurrent Load Execution (Worker Pool Pattern)

- **Problem Statement:** Firing requests sequentially is too slow, and spawning 500 goroutines at once ignores the `concurrency` limit. You need a fixed pool of workers pulling from a shared queue.
- **What to Learn First (20 mins):**
  - _Go by Example:_ **Goroutines**, **Channels**, **Channel Buffering**, **Worker Pools**, and **WaitGroups**.
- **How It Works Conceptually:**
  - Think of a **Worker Pool** like a coffee shop with `concurrency` baristas (goroutines) and a ticket rail (`jobs` channel) holding `totalRequests` orders:
    1. Create a buffered `jobs` channel of size `totalRequests`, and a buffered `results` channel of size `totalRequests`.
    2. Create a `sync.WaitGroup`.
    3. Spawn `concurrency` goroutines. Each goroutine runs a loop (`for range jobs`) that calls your Task 1.2 single-request function and pushes the `RequestResult` into the `results` channel. When the `jobs` channel is empty and closed, the goroutine exits and calls `wg.Done()`.
    4. Push `totalRequests` items into `jobs`, then `close(jobs)` so workers know no more work is coming.
    5. Call `wg.Wait()` to block until every worker finishes, then `close(results)`.
- **What You Need to Do:**
  1. In `runner.go`, create a `RunBenchmark(req RunRequest)` function.
  2. Initialize **one** shared `http.Client` (with `Transport` tuned for `MaxIdleConnsPerHost: req.Concurrency`) so all workers share the connection pool.
  3. Record the overall wall-clock start time before launching the worker pool, and calculate `totalDurationMs` right after `wg.Wait()` finishes.
  4. Implement the Worker Pool pattern described above and collect all `RequestResult` items from the `results` channel into a slice.
- **Done When:** Running 50 requests with `concurrency: 10` finishes significantly faster than `concurrency: 1`, and you receive exactly 50 results in your slice without deadlocks or race conditions (verify by running `go run -race .`).

### Task 1.4: Metrics Aggregation & Final JSON Contract

- **Problem Statement:** Turn the raw slice of `RequestResult` structs into the final statistical summary expected by the Node.js API.
- **What to Learn First (5 mins):**
  - _Docs:_ `sort.Float64s`, `math.Round`, `strconv.Itoa`.
- **What You Need to Do:**
  1. Define a `RunResponse` struct matching `engine/REQUIREMENTS.md` with proper JSON tags.
  2. Loop through your collected `RequestResult` slice to:
     - Build a slice of all `DurationMs` values.
     - Count occurrences in a `statusCodes` map (convert status codes to strings; if `Err != nil`, increment `"ERROR"`).
     - Count total failures (`Err != nil` OR status code >= 400).
  3. Sort the durations slice to extract `minMs`, `maxMs`, and `p95Ms` (index `int(0.95 * float64(len(durations)-1))`).
  4. Compute `avgMs`, `rps`, and `errorRate`, and round all floats to 2 decimal places.
  5. Wire `RunBenchmark` into your `POST /run` handler in `main.go`.
- **Done When:** `curl` tests against a working URL and a dead port both return valid, accurate JSON matching `engine/REQUIREMENTS.md`, and `go run -race .` reports zero data races.

---

## Phase 2: `api/` (Node.js + MongoDB Core Backend)

### Task 2.1: Express Server, Config & Health Endpoint

- **Problem Statement:** Establish the Node.js API server, database connection, and a fast local target you can safely benchmark.
- **What You Need to Do:**
  1. Initialize `api/` (`npm init -y`) and install `express`, `mongoose`, `cors`, and `dotenv`.
  2. Create a `.env` file with `PORT=5000`, `MONGO_URI`, and `ENGINE_URL=http://127.0.0.1:8000`.
  3. Set up `server.js` with `cors()`, `express.json()`, and MongoDB connection on startup.
  4. Add `GET /api/health` returning `200 OK` with `{ status: "ok" }`.
- **Done When:** Server connects to MongoDB cleanly and `curl http://127.0.0.1:5000/api/health` returns `200`.

### Task 2.2: Target Presets CRUD

- **Problem Statement:** Persist reusable API endpoint configurations in MongoDB.
- **What You Need to Do:**
  1. Create `models/Target.js` matching `api/REQUIREMENTS.md`.
  2. Create `routes/targets.js` implementing:
     - `POST /api/targets` (validate required fields, return `201`).
     - `GET /api/targets` (return all targets sorted by `createdAt: -1`).
     - `DELETE /api/targets/:id` (validate ObjectId, return `404` if not found).
- **Done When:** You can create, list, and delete a target pointing to `http://127.0.0.1:5000/api/health` via `curl`.

### Task 2.3: Go Engine Integration & Run History

- **Problem Statement:** Trigger benchmarks from Node, store results in MongoDB, and handle Go service downtime gracefully.
- **What You Need to Do:**
  1. Create `models/Run.js` referencing `Target` with an index on `targetId`.
  2. Implement `POST /api/targets/:id/run`:
     - Validate `totalRequests` and `concurrency` from `req.body`.
     - Look up the `Target` by `:id`.
     - Use Node's native `fetch()` to call `${process.env.ENGINE_URL}/run` with the target's details + run config.
     - If the Go engine is unreachable or returns an error, respond with `502 Bad Gateway` and a clear JSON message.
     - Save the returned metrics into the `Run` collection and return `201`.
  3. Implement `GET /api/targets/:id/runs` (sorted newest first).
  4. Update `DELETE /api/targets/:id` to also delete all `Run` documents for that `targetId`.
- **Done When:** Triggering a run via `curl` invokes your Go engine, saves the result in MongoDB, and stopping the Go engine produces a clean `502` response.

### Task 2.4: Before/After Comparison Endpoint

- **Problem Statement:** Quantify whether performance improved or regressed between any two benchmark runs.
- **What You Need to Do:**
  1. Implement `GET /api/runs/compare?base=<id1>&target=<id2>`.
  2. Validate both query params are valid ObjectIds and exist in MongoDB.
  3. Compute percentage deltas from `base` to `target` for `avgMs`, `p95Ms`, `rps`, and `errorRate`, guarding against division by zero.
  4. Return `{ base, target, diff }`.
- **Done When:** Running two benchmarks against `/api/health` and comparing their IDs returns accurate percentage deltas.

---

## Phase 3: `client/` (React Dashboard)

### Task 3.1: Vite Scaffold & API Helper

- **Problem Statement:** Separate HTTP fetch logic from React components so UI code stays readable.
- **What You Need to Do:**
  1. Scaffold React inside `client/` using Vite.
  2. Create `src/api.js` with clean async functions for: `getTargets`, `createTarget`, `deleteTarget`, `runBenchmark`, `getRuns`, and `compareRuns`. Throw readable errors when `res.ok` is false.
  3. Build a clean two-column layout in `App.jsx` (left sidebar + right main workspace) with plain CSS.
- **Done When:** React loads on port `5173`, fetches targets from port `5000`, and shows zero console/CORS errors.

### Task 3.2: Sidebar Target Management

- **Problem Statement:** Let users add, select, and delete targets without reloading the page.
- **What You Need to Do:**
  1. Build the Create Target form (Name, URL, Method dropdown, conditional JSON Body textarea for `POST`/`PUT`).
  2. Render the target list in the sidebar, highlight the active target, and wire up the delete button so it updates both MongoDB and React state.
- **Done When:** Creating, selecting, and deleting targets updates the UI immediately.

### Task 3.3: Benchmark Runner & Run History Table

- **Problem Statement:** Let users execute load tests, see active progress, and review past runs.
- **What You Need to Do:**
  1. Display the selected target's details with numeric inputs for `Total Requests` and `Concurrency` and a "Run Benchmark" button.
  2. Disable the button and show a loading state while `isRunning` is true; render any backend error message visibly on screen.
  3. Fetch runs via `useEffect` whenever `selectedTargetId` changes, and prepend newly finished runs to the top of the table.
- **Done When:** Clicking "Run Benchmark" shows a loading state and prepends the new row to the table on completion.

### Task 3.4: Side-by-Side Comparison Panel

- **Problem Statement:** Automatically highlight improvements and regressions when comparing two runs.
- **What You Need to Do:**
  1. Add a checkbox to each row in the history table, capping selection at 2 runs max.
  2. Clear selected run checkboxes whenever the user switches targets.
  3. Trigger `compareRuns` automatically when 2 runs are checked and render a comparison card below the table with green/red color coding (lower is better for latency/error rate; higher is better for RPS).
- **Done When:** Checking any two runs displays accurate, color-coded deltas immediately.

---

## Phase 4: Final Verification

### Task 4.1: End-to-End Smoke Test & README

- **What You Need to Do:**
  1. Start all three services (`go run .`, `node server.js`, `npm run dev`).
  2. Test the full flow: add `/api/health`, run two benchmarks with different concurrency settings, compare them, then test an unreachable URL to confirm clean error handling.
  3. Write a concise root `README.md` with the architecture overview and exact commands to run all three services.
- **Done When:** Everything runs end-to-end with zero unhandled crashes.
