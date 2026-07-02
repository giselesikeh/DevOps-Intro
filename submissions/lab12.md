# Lab 12 — WebAssembly Containers: QuickNotes Endpoint on Spin

## Environment

Repository branch: `feature/lab12`

Tool versions used:

```text
Go: go version go1.26.4 linux/amd64
Spin: spin 3.4.0
TinyGo: tinygo version 0.41.1 linux/amd64 (using go version go1.26.4 and LLVM version 20.1.1)
Wasmtime: wasmtime 46.0.1
Docker: Docker version 29.4.0
```

---

## Task 1 — Build a WASM Endpoint with the Spin SDK

### 1.1 Scaffold

I scaffolded the Spin application from the current Spin Go template:

```bash
cd "/mnt/f/Innopolis 3 year/Devops/Lab work/DevOps-Intro"

mkdir -p wasm
cd wasm
spin new -t http-go moscow-time --accept-defaults
```

This created the Spin Go component in:

```text
wasm/moscow-time/
```

The generated project uses the Spin Go SDK import path:

```go
github.com/spinframework/spin-go-sdk/v2/http
```

---

### 1.2 `main.go`

Path:

```text
wasm/moscow-time/main.go
```

Final handler implementation:

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	spinhttp "github.com/spinframework/spin-go-sdk/v2/http"
)

func init() {
	spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `{"error":"method not allowed"}`)
			return
		}

		nowUTC := time.Now().UTC()
		moscow := nowUTC.Add(3 * time.Hour)

		iso := moscow.Format("2006-01-02T15:04:05") + "+03:00"
		hourMinute := moscow.Format("15:04")

		fmt.Fprintf(
			w,
			"{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q,\"timezone\":\"Europe/Moscow\",\"utc_offset\":\"+03:00\"}",
			nowUTC.Unix(),
			iso,
			hourMinute,
		)
	})
}
```

The handler returns JSON for `GET /time` with:

- `unix`
- `iso`
- `hour_minute`
- `timezone`
- `utc_offset`

Moscow time is computed as UTC + 3 hours to avoid relying on unavailable timezone database support inside TinyGo/WASI.

---

### 1.3 `spin.toml`

Path:

```text
wasm/moscow-time/spin.toml
```

Final manifest:

```toml
#:schema https://schemas.spinframework.dev/spin/manifest-v2/latest.json

spin_manifest_version = 2

[application]
name = "moscow-time"
version = "0.1.0"
authors = ["Gisele Wiykiynyuy <giselesikeh17@gmail.com>"]
description = ""

[[trigger.http]]
route = "/time"
component = "moscow-time"

[component.moscow-time]
source = "main.wasm"
allowed_outbound_hosts = []

[component.moscow-time.build]
command = "tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o main.wasm ."
watch = ["**/*.go", "go.mod"]
```

Important configuration points:

- The HTTP route is `/time`.
- `allowed_outbound_hosts = []` disables outbound network access.
- The build command uses TinyGo with `-target=wasip1`.
- The build command includes `-buildmode=c-shared`, as required by Spin.

---

### 1.4 Build output

Build command:

```bash
cd "/mnt/f/Innopolis 3 year/Devops/Lab work/DevOps-Intro/wasm/moscow-time"
spin build
```

Build output:

```text
Building component moscow-time with `tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o main.wasm .`
Finished building all Spin components
```

WASM artifact size:

```text
-rwxrwxrwx 1 student student 362K Jul  2 22:59 main.wasm
370497 main.wasm
```

So the generated `main.wasm` size is:

```text
362K / 370497 bytes
```

---

### 1.5 Run and verify

Run command:

```bash
cd "/mnt/f/Innopolis 3 year/Devops/Lab work/DevOps-Intro/wasm/moscow-time"
spin up
```

Spin output:

```text
Logging component stdio to ".spin/logs/"

Serving http://127.0.0.1:3000
Available Routes:
  moscow-time: http://127.0.0.1:3000/time
```

Raw `curl` verification:

```bash
curl -i http://127.0.0.1:3000/time
```

Output:

```text
HTTP/1.1 200 OK
content-type: application/json
content-length: 124
date: Thu, 02 Jul 2026 20:35:14 GMT

{"unix":1783024514,"iso":"2026-07-02T23:35:14+03:00","hour_minute":"23:35","timezone":"Europe/Moscow","utc_offset":"+03:00"}
```

Pretty JSON verification:

```bash
curl -s http://127.0.0.1:3000/time | python3 -m json.tool
```

Output:

```json
{
    "unix": 1783024516,
    "iso": "2026-07-02T23:35:16+03:00",
    "hour_minute": "23:35",
    "timezone": "Europe/Moscow",
    "utc_offset": "+03:00"
}
```

This confirms that the Spin component serves valid JSON at `GET /time`.

---

### 1.6 Design questions

#### a) Browser WASM vs server WASM

Browser WASM with `go build` and the `js/wasm` target is designed for execution inside a browser JavaScript environment. It depends on JavaScript glue code, browser APIs, and the browser event loop.

Server WASM with TinyGo and `-target=wasip1` targets WASI instead of the browser. In this target, browser-specific APIs such as the DOM, JavaScript interop, and browser networking are missing. The module runs in a server-side WASI host such as Spin or Wasmtime.

The gain is that the module becomes small, portable, sandboxed, and suitable for server-side execution. It can run outside a browser with a capability-based runtime and can be deployed as a lightweight server-side component.

#### b) Why does the build command need `-buildmode=c-shared`?

Spin expects the compiled WebAssembly component to export the symbols and ABI shape needed by the Spin host. The Go Spin SDK registers the HTTP handler through Spin's expected component interface.

The `-buildmode=c-shared` flag makes TinyGo produce a WebAssembly module with the required exported interface for Spin to call into. Without it, the module may build as a normal WASI module but will not expose the HTTP handler in the way Spin expects, causing runtime failures when Spin tries to serve requests.

#### c) `allowed_outbound_hosts = []` and capability-based security

`allowed_outbound_hosts = []` means the component has no permission to make outbound network requests. This follows a capability-based security model: the component only receives the permissions explicitly granted in the Spin manifest.

This is similar in spirit to running a Docker container with `--network none`, because both prevent outbound networking. The difference is that Spin's model is more fine-grained and application-level. Instead of giving a process a full network namespace and then restricting it, the WASM component starts with no capability and the host grants only specific capabilities, such as selected outbound hosts.

#### d) TinyGo stdlib gaps encountered

During this lab, I avoided `time.LoadLocation("Europe/Moscow")` because TinyGo/WASI does not include the full operating-system timezone database by default. Instead, I computed Moscow time as:

```go
time.Now().UTC().Add(3 * time.Hour)
```

I also avoided reflection-heavy JSON generation using `map[string]any` with `encoding/json`, because TinyGo can have limitations with reflection-heavy standard-library behavior. The handler builds the JSON response directly with `fmt.Fprintf` and `%q`, which is simple and robust for this small fixed response.

---

## Task 2 — Perf Comparison vs Lab 6 Container

### 2.1 Test rig

Measurements were taken on my local Windows machine through WSL/Ubuntu, using Docker Desktop integration.

Environment:

```text
OS/runtime: Ubuntu on WSL, running on Windows
Go: go version go1.26.4 linux/amd64
Spin: spin 3.4.0
TinyGo: tinygo version 0.41.1 linux/amd64 (using go version go1.26.4 and LLVM version 20.1.1)
Hyperfine: hyperfine 1.12.0
Wasmtime: wasmtime 46.0.1
Docker: Docker version 29.4.0
```

The Spin endpoint was measured at:

```text
http://127.0.0.1:3000/time
```

The Lab 6 Docker container was measured at:

```text
http://127.0.0.1:8080/health
```

For Docker cold-start measurements, I used host port `18080` to avoid conflicts with an already running QuickNotes container on port `8080`.

---

### 2.2 Artifact size

Lab 12 Spin WASM artifact:

```bash
cd "/mnt/f/Innopolis 3 year/Devops/Lab work/DevOps-Intro/wasm/moscow-time"
ls -lh main.wasm
wc -c main.wasm
```

Output:

```text
-rwxrwxrwx 1 student student 362K Jul  2 22:59 main.wasm
370497 main.wasm
```

Lab 6 Docker image:

```bash
docker images quicknotes:lab6
docker image inspect quicknotes:lab6 --format '{{ .Size }}'
```

Output:

```text
IMAGE             ID             DISK USAGE   CONTENT SIZE   EXTRA
quicknotes:lab6   1673ade24efa       22.8MB         5.71MB    U

5711344
```

So the artifact sizes used in the comparison are:

```text
Lab 6 Docker image: 5,711,344 bytes
Lab 12 WASM/Spin module: 370,497 bytes
```

---

### 2.3 Warm latency

Spin warm-latency command:

```bash
hyperfine --warmup 5 --runs 50 --export-json /tmp/lab12_spin_warm.json \
  'curl -s -o /dev/null http://127.0.0.1:3000/time'
```

Spin output:

```text
Benchmark 1: curl -s -o /dev/null http://127.0.0.1:3000/time
  Time (mean ± σ):      16.5 ms ±   1.4 ms    [User: 3.4 ms, System: 4.7 ms]
  Range (min … max):    13.5 ms …  20.4 ms    50 runs
```

Calculated values:

```text
spin_warm_p50_ms=16.480
spin_warm_p95_ms=18.352
```

Docker warm-latency command:

```bash
hyperfine --warmup 5 --runs 50 --export-json /tmp/lab12_docker_warm.json \
  'curl -s -o /dev/null http://127.0.0.1:8080/health'
```

Docker output:

```text
Benchmark 1: curl -s -o /dev/null http://127.0.0.1:8080/health
  Time (mean ± σ):       9.6 ms ±   1.1 ms    [User: 4.4 ms, System: 4.0 ms]
  Range (min … max):     8.2 ms …  14.2 ms    50 runs
```

Calculated values:

```text
docker_warm_p50_ms=9.472
docker_warm_p95_ms=11.226
```

---

### 2.4 Cold start

For cold start, I measured the time from starting the runtime to the first successful HTTP response. I collected five samples for each runtime.

Spin cold-start command pattern:

```bash
spin up > "/tmp/lab12_spin_cold_${i}.log" 2>&1 &
curl -s http://127.0.0.1:3000/time
```

Spin cold-start samples:

```text
spin_cold_sample_1_ms=159
spin_cold_sample_2_ms=158
spin_cold_sample_3_ms=203
spin_cold_sample_4_ms=188
spin_cold_sample_5_ms=456
```

Spin cold-start summary:

```text
spin_cold_samples_ms=159, 158, 203, 188, 456
spin_cold_p50_ms=188
```

Docker cold-start command pattern:

```bash
docker run -d --name quicknotes-lab12-cold -p 18080:8080 quicknotes:lab6
curl -s http://127.0.0.1:18080/health
```

Docker cold-start samples:

```text
docker_cold_sample_1_ms=877
docker_cold_sample_2_ms=699
docker_cold_sample_3_ms=689
docker_cold_sample_4_ms=719
docker_cold_sample_5_ms=658
```

Docker cold-start summary:

```text
docker_cold_samples_ms=877, 699, 689, 719, 658
docker_cold_p50_ms=699
```

---

### 2.5 Comparison table

| Dimension              | Lab 6 Docker        | Lab 12 WASM/Spin |
|------------------------|--------------------:|-----------------:|
| Artifact size          | 5,711,344 bytes     | 370,497 bytes    |
| Artifact size readable | 5.71 MB             | 362 KB           |
| Cold start samples     | 877, 699, 689, 719, 658 ms | 159, 158, 203, 188, 456 ms |
| Cold start p50         | 699 ms              | 188 ms           |
| Warm latency p50       | 9.472 ms            | 16.480 ms        |
| Warm latency p95       | 11.226 ms           | 18.352 ms        |

In my local measurements, the WASM/Spin artifact was much smaller and had faster cold start. The Docker endpoint had lower warm latency for this specific test, likely because the Lab 6 container is a native Go binary already running as a normal HTTP server, while the Spin request path includes the Spin/Wasmtime HTTP component runtime.

---

### 2.6 Design questions

#### e) What dominates each platform's cold start?

For the Docker container, cold start is dominated by container startup work: Docker has to create the container process, configure the namespace/cgroup environment, set up port forwarding, attach the filesystem layers, and start the native QuickNotes binary.

For Spin, cold start is dominated by starting the Spin host, loading the WebAssembly module, instantiating it in Wasmtime, and preparing the wasi-http handler. Since the WASM module is very small, this is lighter than starting the Docker container in my measurements.

#### f) For what workloads is WASM clearly better, and where is Docker still right?

WASM is clearly better for small, stateless, security-sensitive services where fast cold start, small artifacts, and strong sandboxing are important. Examples include edge functions, request filters, small API handlers, plugin systems, and multi-tenant serverless workloads.

Docker is still right for larger applications that need full operating-system features, mature networking support, existing Linux binaries, background processes, databases, sidecars, shells, package managers, or libraries that are not easy to compile to WASI. Docker is also the safer default when the app depends heavily on the normal Go standard library, filesystem behavior, or native C libraries.

#### g) Multi-tenant safety: what concrete attack does WASM make harder?

WASM makes host escape and cross-tenant attacks harder because the module does not receive direct access to the host operating system by default. It cannot freely open files, create sockets, spawn processes, or inspect the host environment unless the runtime explicitly grants those capabilities.

A concrete example is a malicious tenant trying to scan internal network services or read host files. In Docker, this depends on Linux namespaces, seccomp, AppArmor, and container configuration. In a WASM capability model like Spin, outbound hosts and other capabilities are denied by default unless explicitly granted, so the same attack is harder because the module simply does not have those capabilities.

---

## Bonus Task — Two WASM Execution Models

### B.1 Goal

For the bonus task, I rebuilt the same Moscow-time logic as a standalone WASI CLI module. This version does not use the Spin SDK. Instead, it reads request-like values from environment variables and writes the JSON response to stdout.

The standalone module is located at:

```text
wasm-cli/main.go
```

---

### B.2 Standalone WASI CLI implementation

```go
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	method := os.Getenv("REQUEST_METHOD")
	path := os.Getenv("PATH_INFO")

	if method == "" {
		method = "GET"
	}

	if path == "" {
		path = "/time"
	}

	if method != "GET" {
		fmt.Fprintln(os.Stderr, "method not allowed")
		os.Exit(1)
	}

	if path != "/time" {
		fmt.Fprintln(os.Stderr, "not found")
		os.Exit(1)
	}

	nowUTC := time.Now().UTC()
	moscow := nowUTC.Add(3 * time.Hour)

	iso := moscow.Format("2006-01-02T15:04:05") + "+03:00"
	hourMinute := moscow.Format("15:04")

	fmt.Fprintf(
		os.Stdout,
		"{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q,\"timezone\":\"Europe/Moscow\",\"utc_offset\":\"+03:00\"}\n",
		nowUTC.Unix(),
		iso,
		hourMinute,
	)
}
```

This program implements the same Moscow-time JSON logic as the Spin component, but it is shaped like a CLI program instead of a wasi-http component.

---

### B.3 Build command

Build command:

```bash
cd "/mnt/f/Innopolis 3 year/Devops/Lab work/DevOps-Intro/wasm-cli"

tinygo build -o main.wasm -target=wasi -no-debug ./main.go
```

Output size:

```text
-rwxrwxrwx 1 student student 193K Jul  3 00:11 main.wasm
196707 main.wasm
```

So the standalone WASI CLI module size is:

```text
193K / 196707 bytes
```

---

### B.4 Run command

Run command:

```bash
wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm
```

Output:

```json
{"unix":1783026731,"iso":"2026-07-03T00:12:11+03:00","hour_minute":"00:12","timezone":"Europe/Moscow","utc_offset":"+03:00"}
```

Pretty JSON verification:

```bash
wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm | python3 -m json.tool
```

Output:

```json
{
    "unix": 1783026733,
    "iso": "2026-07-03T00:12:13+03:00",
    "hour_minute": "00:12",
    "timezone": "Europe/Moscow",
    "utc_offset": "+03:00"
}
```

This confirms that the standalone WASI CLI module runs under bare `wasmtime run`.

---

### B.5 Per-invocation `wasmtime run` cold-start measurement

Command:

```bash
hyperfine --warmup 3 --runs 20 --export-json /tmp/lab12_wasmtime_cli_cold.json \
  'wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm >/dev/null'
```

Output:

```text
Benchmark 1: wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm >/dev/null
  Time (mean ± σ):      16.6 ms ±   1.0 ms    [User: 5.4 ms, System: 8.6 ms]
  Range (min … max):    14.5 ms …  18.6 ms    20 runs
```

Calculated values:

```text
wasmtime_cli_cold_p50_ms=16.379
wasmtime_cli_cold_p95_ms=18.193
```

---

### B.6 Spin component vs standalone WASI CLI comparison

| Dimension | Spin wasi-http component | Standalone WASI CLI module |
|----------|--------------------------:|----------------------------:|
| Directory | `wasm/moscow-time/` | `wasm-cli/` |
| Runtime | `spin up` | `wasmtime run` |
| Build command | `tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o main.wasm .` | `tinygo build -o main.wasm -target=wasi -no-debug ./main.go` |
| Run command | `spin up`, then `curl http://127.0.0.1:3000/time` | `wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm` |
| Module size | 370497 bytes / 362K | 196707 bytes / 193K |
| Cold start p50 | 188 ms to start Spin and get first HTTP response | 16.379 ms per `wasmtime run` invocation |
| Cold start p95 | Not separately computed for Spin cold start | 18.193 ms |
| Execution model | Persistent HTTP server using wasi-http | Per-invocation CLI execution |

The standalone CLI module is smaller because it does not include the Spin HTTP SDK and does not need to export a wasi-http handler. The Spin version is larger, but it provides an HTTP server model, routing, manifest-based configuration, and capability policy such as `allowed_outbound_hosts = []`.

---

### B.7 Design questions

#### h) Why can't the Task 1 Spin component run under bare `wasmtime run`?

The Task 1 Spin component is a wasi-http component. It exports an HTTP handler interface that a wasi-http host can call. Bare `wasmtime run` expects a normal WASI CLI module with a `_start` entrypoint.

Because the Spin component is not shaped like a CLI program, bare `wasmtime run` does not know how to call the HTTP handler directly. It needs a wasi-http host such as Spin, or another compatible HTTP component host.

#### i) Spin uses wasmtime internally. So what does Spin add on top of bare wasmtime?

Spin adds the application runtime around Wasmtime. It provides:

- an HTTP server loop
- wasi-http request/response handling
- routing from `spin.toml`
- component loading based on the manifest
- capability policy such as `allowed_outbound_hosts`
- developer commands such as `spin build` and `spin up`

So Wasmtime executes the WebAssembly, but Spin turns the module into a deployable HTTP service.

#### j) Two execution models — when does each fit?

The per-invocation `wasmtime run` model fits command-style workloads where each invocation receives input, performs a small job, writes output, and exits. Examples include batch transforms, CLI filters, test helpers, and CGI-style request execution.

The Spin persistent wasi-http server model fits HTTP services that should stay available and respond to many requests through a normal server interface. Examples include small APIs, edge functions, webhooks, request filters, and serverless HTTP endpoints.
