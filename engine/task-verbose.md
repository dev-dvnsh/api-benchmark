# Go Engine — Mentor-Style Step-by-Step Guide (`engine/task-verbose.md`)

**How to use this guide:** Every time a new Go function appears, read the **What it does** and **Read here** notes before writing it. All docs live on `pkg.go.dev/<package-name>` (e.g., `pkg.go.dev/net/http`) or `gobyexample.com`.

---

## Task 1.1: HTTP Server, JSON Decoding & Validation (`engine/main.go`)

### Step 1: Structs & The `writeJSON` Helper

_(Since you already built `RunRequest`, start here.)_

1. **Define `ErrorResponse` struct:**
   - Needs one field: `Error` of type `string`, with the JSON tag `json:"error"`.
2. **Create a helper function `writeJSON(w http.ResponseWriter, statusCode int, data any)`:**
   - **What `any` means:** `any` is a built-in Go type that means "accept any type of value" (just like `any` in TypeScript). This lets you pass either an `ErrorResponse` or a `RunResponse` into this same helper.
   - **Inside `writeJSON`, do these 3 things in exact order:**
     1. **Set the header:** Call `w.Header().Set("Content-Type", "application/json")`.
        - _What it does:_ `w.Header()` returns the response headers map; `.Set(key, value)` sets a header—identical to `res.setHeader()` in Express.
        - _Read here:_ `pkg.go.dev/net/http#Header.Set`
     2. **Write the HTTP status code:** Call `w.WriteHeader(statusCode)`.
        - _What it does:_ Identical to `res.status(statusCode)` in Express. **Important:** You must set all headers _before_ calling `WriteHeader`, otherwise Go ignores them.
        - _Read here:_ `pkg.go.dev/net/http#ResponseWriter`
     3. **Encode and send the JSON:** Call `json.NewEncoder(w).Encode(data)`.
        - _What it does:_ `json.NewEncoder(w)` creates a JSON writer attached directly to the HTTP response stream `w`. Calling `.Encode(data)` converts your Go struct into JSON bytes and sends it to the client (like `res.json(data)` in Express).
        - _Read here:_ `pkg.go.dev/encoding/json#NewEncoder`

### Step 2: Inside `runHandler(w http.ResponseWriter, r *http.Request)`

Follow this exact sequence:

1. **Check the HTTP method:**
   - Compare `r.Method` (a `string`, like `req.method` in Express) against `http.MethodPost` (a built-in Go constant equal to `"POST"`).
   - If they don't match, call `writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "..."})` and immediately `return`.
2. **Schedule body cleanup:**
   - Write `defer r.Body.Close()`.
   - _What `defer` does:_ It tells Go "run this line automatically right before `runHandler` exits, no matter which `return` statement gets hit" (similar to a `finally` block in JS). Closing `r.Body` frees the network stream.
   - _Read here:_ `gobyexample.com/defer`
3. **Create an empty struct instance:**
   - Declare `var req RunRequest`. This allocates an empty `RunRequest` in memory with default zero values (`""` for strings, `0` for ints, `nil` for maps).
4. **Decode the incoming JSON into `req`:**
   - Call `err := json.NewDecoder(r.Body).Decode(&req)`.
   - _What it does:_ `json.NewDecoder(r.Body)` reads raw bytes from the incoming request body. `.Decode(&req)` parses that JSON and fills in your `req` variable.
   - _Why `&req`?_ The `&` passes the memory address (pointer) of `req` so `Decode` can modify your actual variable instead of a temporary copy. It returns an `error` (which is `nil` if decoding succeeded).
   - _Read here:_ `pkg.go.dev/encoding/json#Decoder.Decode`
5. **Check the decode error:**
   - If `err != nil`, call `writeJSON` with `http.StatusBadRequest` (`400`) and an `ErrorResponse`, then `return`.
6. **Apply default & validate:**
   - If `req.Method == ""`, set `req.Method = "GET"`.
   - Write `if` checks for: empty `req.URL`, invalid `req.Method` (must be GET/POST/PUT/DELETE), `req.TotalRequests` (< 1 or > 500), `req.Concurrency` (< 1 or > 50 or > `req.TotalRequests`). For any failure, call `writeJSON` with `400` and `return`.
7. **Temporary response:**
   - Call `writeJSON(w, http.StatusOK, req)` so you can test that your JSON parsed properly.

### Step 3: Wire Up `main()`

1. **Register the route:** Call `http.HandleFunc("/run", runHandler)`.
   - _What it does:_ Tells Go's default router to run `runHandler` whenever a request hits `/run` (like `app.all('/run', runHandler)` in Express).
   - _Read here:_ `pkg.go.dev/net/http#HandleFunc`
2. **Start the server:** Call `err := http.ListenAndServe(":8000", nil)`.
   - _What it does:_ Identical to `app.listen(8000)` in Express. First argument is the address string `":8000"`. Second argument is a custom router—passing `nil` tells Go to use the default router you just registered `/run` on. It blocks forever unless the server fails to start (e.g., port already in use), in which case it returns an `error`.
   - _Read here:_ `gobyexample.com/http-servers`

**Verify Task 1.1:** Run `go run .` and test with `curl` (valid POST, GET request, and `totalRequests: 999`).

---

## Task 1.2: Single-Request Execution (`engine/runner.go`)

### Step 1: Create `runner.go`

1. Create `runner.go` in `engine/` with `package main` at the top. _(Files in the same package share all structs and functions automatically.)_
2. Define a `RequestResult` struct with three fields:
   - `DurationMs` of type `float64`
   - `StatusCode` of type `int`
   - `Err` of type `error`

### Step 2: Write `executeSingleRequest(client *http.Client, req RunRequest) RequestResult`

Follow this exact sequence:

1. **Prepare the request body:**
   - Declare `var bodyReader io.Reader` (starts as `nil`).
   - _What `io.Reader` is:_ Go's standard interface for anything you can read bytes from (like a `Readable` stream in Node).
   - If `len(req.Body) > 0`, assign `bodyReader = bytes.NewReader(req.Body)`.
   - _What `bytes.NewReader` does:_ Takes a byte slice (`[]byte`—which `json.RawMessage` is under the hood) and wraps it into an `io.Reader` so Go's HTTP client can stream it.
   - _Read here:_ `pkg.go.dev/bytes#NewReader`
2. **Build the outgoing request:**
   - Call `httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)`.
   - _What it does:_ Builds an HTTP request object in memory _without sending it yet_. Takes `method` (`string`), `url` (`string`), and `body` (`io.Reader`, or `nil`). Returns `(*http.Request, error)`.
   - If `err != nil`, immediately `return RequestResult{Err: err}`.
   - _Read here:_ `pkg.go.dev/net/http#NewRequest`
3. **Attach custom headers:**
   - Loop through `req.Headers` using `for key, value := range req.Headers` and call `httpReq.Header.Set(key, value)`.
4. **Start the timer:**
   - Call `start := time.Now()`. Returns the current high-precision timestamp (`time.Time`).
5. **Send the request:**
   - Call `resp, err := client.Do(httpReq)`.
   - _What it does:_ Actually fires the HTTP request over the network (like `await fetch()` in Node). Returns `(*http.Response, error)`.
   - _Read here:_ `pkg.go.dev/net/http#Client.Do`
6. **Stop the timer & calculate milliseconds:**
   - Call `elapsed := time.Since(start)` (returns a `time.Duration`).
   - Convert to `float64` milliseconds: `durationMs := float64(elapsed.Microseconds()) / 1000.0` (or `elapsed.Seconds() * 1000.0`). _(Note: `elapsed.Milliseconds()` returns a whole integer and drops decimals, so using `Microseconds` or `Seconds` keeps decimal accuracy.)_
   - _Read here:_ `pkg.go.dev/time#Since`
7. **Handle network failure:**
   - If `err != nil`, immediately `return RequestResult{DurationMs: durationMs, Err: err}`.
8. **Drain and close the response body (Critical Go rule):**
   - Write `defer resp.Body.Close()`.
   - Call `io.Copy(io.Discard, resp.Body)`.
   - _What it does:_ `io.Copy(destination, source)` streams all bytes from `source` to `destination`. `io.Discard` is a built-in black hole (like `/dev/null`). If you don't drain the unread response body before closing it, Go cannot reuse the TCP connection for the next request.
   - _Read here:_ `pkg.go.dev/io#Copy`
9. **Return success:**
   - `return RequestResult{DurationMs: durationMs, StatusCode: resp.StatusCode}` (`Err` stays `nil`).

**Verify Task 1.2:** Temporarily call `executeSingleRequest(&http.Client{Timeout: 10 * time.Second}, req)` once inside `runHandler` and print the result with `fmt.Println`.

---

## Task 1.3: Concurrent Worker Pool (`engine/runner.go`)

Before coding this task, spend 10 minutes reading these 3 short pages on Go by Example:

- `gobyexample.com/goroutines`
- `gobyexample.com/waitgroups`
- `gobyexample.com/worker-pools`

### Step 1: Write `RunBenchmark(req RunRequest)`

Follow this exact order inside `RunBenchmark`:

1. **Create one shared `http.Client` with a tuned `Transport`:**
   - Create a transport pointer: `transport := &http.Transport{MaxIdleConns: req.Concurrency, MaxIdleConnsPerHost: req.Concurrency}`.
   - _What `http.Transport` does:_ Manages the underlying TCP connection pool. By default, Go only keeps 2 idle connections per host; raising `MaxIdleConnsPerHost` to `req.Concurrency` lets all your workers reuse connections smoothly.
   - Create the client: `client := &http.Client{Transport: transport, Timeout: 10 * time.Second}`.
   - _Read here:_ `pkg.go.dev/net/http#Transport`
2. **Create two buffered channels:**
   - `jobs := make(chan int, req.TotalRequests)`
   - `results := make(chan RequestResult, req.TotalRequests)`
   - _What `make(chan Type, size)` does:_ Creates a thread-safe in-memory queue that can hold up to `size` items without blocking. Workers pull job tickets from `jobs` and drop finished `RequestResult` structs into `results`.
   - _Read here:_ `gobyexample.com/channel-buffering`
3. **Create a `WaitGroup`:**
   - Declare `var wg sync.WaitGroup`.
   - _What it does:_ A thread-safe counter (like `Promise.all()` in JS). You increment it before starting each background worker, each worker decrements it when finished, and `wg.Wait()` pauses until the counter hits `0`.
4. **Start the overall wall-clock timer:**
   - `overallStart := time.Now()`
5. **Spawn `req.Concurrency` worker goroutines:**
   - Write a standard `for i := 0; i < req.Concurrency; i++` loop:
     1. **First**, call `wg.Add(1)` (increments the counter _before_ the goroutine starts).
     2. **Second**, launch a goroutine: `go func() { ... }()`.
     3. **Inside the goroutine:**
        - First line: `defer wg.Done()` (decrements the counter when this worker exits).
        - Next: `for range jobs { ... }`. Each iteration grabs one ticket from the `jobs` channel. Inside this loop, call `res := executeSingleRequest(client, req)` and push it into the results channel: `results <- res`.
6. **Fill the `jobs` channel & close it:**
   - Write a loop `for i := 0; i < req.TotalRequests; i++` that sends `jobs <- i`.
   - Immediately after the loop, call `close(jobs)`.
   - _What `close(jobs)` does:_ Tells every worker's `for range jobs` loop: "No more tickets are coming—exit cleanly once the queue is empty."
7. **Wait for all workers to finish:**
   - Call `wg.Wait()`. Execution pauses here until every worker has called `wg.Done()`.
8. **Stop the overall timer:**
   - Calculate `totalDurationMs` from `time.Since(overallStart)`.
9. **Close `results` and collect into a slice:**
   - Call `close(results)`.
   - Declare `var collected []RequestResult`.
   - Loop `for res := range results` and append each `res` to `collected` using Go's built-in `append(collected, res)`.

**Verify Task 1.3:** Run `go run -race .` (the `-race` flag checks for concurrency bugs) and confirm 50 requests at concurrency 10 return 50 results with zero race warnings.

---

## Task 1.4: Metrics Aggregation & Final Wiring (`runner.go` + `main.go`)

### Step 1: Define `RunResponse` (`runner.go`)

Define the `RunResponse` struct with Capitalized fields and JSON tags matching `engine/REQUIREMENTS.md`:

- `TotalRequests` (`int`)
- `TotalDurationMs`, `AvgMs`, `MinMs`, `MaxMs`, `P95Ms`, `RPS`, `ErrorRate` (`float64`)
- `StatusCodes` (`map[string]int`)

### Step 2: Aggregate `collected` into `RunResponse` (`runner.go`)

1. **Initialize variables:**
   - `durations := make([]float64, 0, len(collected))` — creates an empty `float64` slice with pre-allocated memory.
   - `statusCodes := make(map[string]int)` — **must use `make()`**; writing to a `nil` map in Go causes a runtime crash.
   - `var totalLatencyMs float64` and `var errorCount int`.
2. **Loop through `collected` (`for _, res := range collected`):**
   - Append `res.DurationMs` to `durations` and add it to `totalLatencyMs`.
   - If `res.Err != nil`: increment `statusCodes["ERROR"]` and `errorCount`.
   - Else:
     - Convert `res.StatusCode` (`int`) to a `string` using `strconv.Itoa(res.StatusCode)`.
       - _What `strconv.Itoa` does:_ "Integer to ASCII" (like `String(200)` in JS). _Read here:_ `pkg.go.dev/strconv#Itoa`
     - Increment `statusCodes[codeStr]`.
     - If `res.StatusCode >= 400`, increment `errorCount`.
3. **Sort `durations`:**
   - Call `sort.Float64s(durations)`.
   - _What it does:_ Sorts a `[]float64` slice in-place from smallest to largest. _Read here:_ `pkg.go.dev/sort#Float64s`
4. **Compute the stats:**
   - `minMs` = `durations[0]`
   - `maxMs` = `durations[len(durations)-1]`
   - `p95Index` = `int(0.95 * float64(len(durations)-1))`, then `p95Ms = durations[p95Index]`
   - `avgMs` = `totalLatencyMs / float64(len(durations))`
   - `rps` = `float64(req.TotalRequests) / (totalDurationMs / 1000.0)` (if `totalDurationMs > 0`)
   - `errorRate` = `(float64(errorCount) / float64(req.TotalRequests)) * 100.0`
5. **Round floats to 2 decimals:**
   - Write a tiny helper `round2(val float64) float64` that returns `math.Round(val*100) / 100`.
   - _Read here:_ `pkg.go.dev/math#Round`
6. **Return the `RunResponse` struct** from `RunBenchmark`.

### Step 3: Wire into `main.go`

1. In `runHandler`, replace your temporary test response with `report := RunBenchmark(req)` and pass `report` into `writeJSON(w, http.StatusOK, report)`.

**Verify Task 1.4:** Run `go run -race .` and test `POST http://127.0.0.1:8000/run` with `curl` against a live URL and a dead local port.
