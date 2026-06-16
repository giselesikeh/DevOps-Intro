# Lab 6 — Containers: Dockerize QuickNotes

## Task 1 — Multi-Stage Dockerfile, ≤ 25 MB

### Dockerfile

File: `app/Dockerfile`

```dockerfile
# syntax=docker/dockerfile:1

# ---------- builder stage ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /quicknotes .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /healthcheck ./cmd/healthcheck

# ---------- runtime stage ----------
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /quicknotes /quicknotes
COPY --from=builder /healthcheck /healthcheck
COPY seed.json /seed.json
COPY --chown=nonroot:nonroot data/ /data/

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/quicknotes"]
```

### Build command

Run from `app/`:

```bash
docker build --pull --no-cache -t quicknotes:lab6 .
```

The image built successfully:

```text
[+] Building 136.7s (17/17) FINISHED
=> naming to docker.io/library/quicknotes:lab6
=> unpacking to docker.io/library/quicknotes:lab6
```

After adding the static healthcheck binary and `/data` ownership fix, the final Compose build also succeeded:

```text
[+] up 4/4
 ✔ Image quicknotes:lab6               Built
 ✔ Network devops-intro_default        Created
 ✔ Volume devops-intro_quicknotes-data Created
 ✔ Container devops-intro-quicknotes-1 Started
```

### Image size

Command:

```bash
docker images quicknotes:lab6
```

Output:

```text
IMAGE             ID             DISK USAGE   CONTENT SIZE
quicknotes:lab6   1673ade24efa   22.8MB       5.71MB
```

The final image is **22.8 MB**, which is below the required **25 MB** limit.

### Image configuration excerpt

Command:

```bash
docker inspect quicknotes:lab6 --format 'User={{ .Config.User }} Entrypoint={{ json .Config.Entrypoint }} ExposedPorts={{ json .Config.ExposedPorts }}'
```

Output:

```text
User=nonroot:nonroot Entrypoint=["/quicknotes"] ExposedPorts={"8080/tcp":{}}
```

This confirms that the image runs as a non-root user, declares port `8080`, and uses exec-form `ENTRYPOINT`.

### Base image comparison

Command:

```bash
docker pull golang:1.24-alpine
docker images golang:1.24-alpine
```

Output:

```text
IMAGE                ID             DISK USAGE   CONTENT SIZE
golang:1.24-alpine   8bee1901f1e5   395MB        83.5MB
```

The builder image `golang:1.24-alpine` is **395 MB**, while the final runtime image `quicknotes:lab6` is **22.8 MB**.

This shows why the multi-stage build matters: the Go toolchain and build environment stay in the builder stage, while the final image contains only the compiled static binaries, `seed.json`, seeded `/data` contents, and the minimal distroless runtime.

### Runtime verification

Command run from `app/` using Git Bash path-conversion protection:

```bash
MSYS_NO_PATHCONV=1 docker run -d --name quicknotes-lab6-test \
  -p 8080:8080 \
  -e ADDR=":8080" \
  -e DATA_PATH="/data/notes.json" \
  -e SEED_PATH="/seed.json" \
  -v "$PWD/data:/data" \
  quicknotes:lab6
```

Container status:

```text
CONTAINER ID   IMAGE             COMMAND         CREATED         STATUS         PORTS                                         NAMES
41d0764349f4   quicknotes:lab6   "/quicknotes"   7 seconds ago   Up 7 seconds   0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp   quicknotes-lab6-test
```

Container logs:

```text
2026/06/16 22:55:08 quicknotes listening on :8080 (notes loaded: 7)
```

Health endpoint:

```bash
curl -i http://localhost:8080/health
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 22:55:17 GMT
Content-Length: 26

{"notes":7,"status":"ok"}
```

Notes endpoint:

```bash
curl -i http://localhost:8080/notes
```

Output excerpt:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 22:55:18 GMT
Content-Length: 910

[{"id":3,"title":"DevOps mantra","body":"If it hurts, do it more often.","created_at":"2026-01-15T10:10:00Z"}, ...]
```

The container successfully served both `/health` and `/notes`.

### Design question a — Why does layer order matter?

Layer order matters because Docker caches each build layer. If the Dockerfile copies the entire source tree before downloading dependencies, then every source-code change invalidates the dependency layer and forces `go mod download` to run again.

Bad order:

```dockerfile
COPY . .
RUN go mod download
RUN go build
```

Good order:

```dockerfile
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build
```

Measured rebuild times after a small source-code change:

```text
Bad strategy rebuild time: 33.911s
Good strategy rebuild time: 33.582s
```

The measured times were close in this small project because the Go binary still had to rebuild in both cases. However, the good Dockerfile showed the dependency layer as cached:

```text
CACHED [builder 3/7] COPY go.mod ./
CACHED [builder 4/7] RUN go mod download
```

That means normal source-code changes do not force dependency downloads again. This matters more in larger projects with many dependencies or slower package downloads.

### Design question b — Why `CGO_ENABLED=0`?

`CGO_ENABLED=0` builds a static Go binary. This matters because the runtime image uses `gcr.io/distroless/static:nonroot`, which does not contain a normal Linux userspace, shell, package manager, or dynamic linker. If the binary depends on dynamic C libraries and the dynamic linker is missing, the container may fail to start with an error such as `no such file or directory`, even though the binary file exists.

### Design question c — What is `gcr.io/distroless/static:nonroot`?

`gcr.io/distroless/static:nonroot` is a minimal runtime image intended for statically compiled binaries. It includes only the minimal files needed to run the binary and a predefined non-root user. It does not include a shell, `apt`, package manager, compiler, or debugging tools.

This reduces image size and attack surface. Fewer packages also means fewer possible CVEs compared with a full Linux distribution image.

### Design question d — What do `-ldflags='-s -w'` and `-trimpath` do?

`-ldflags='-s -w'` strips symbol-table and debug information from the Go binary. This reduces binary size, but the cost is that debugging and symbol-based inspection become harder.

`-trimpath` removes local filesystem paths from the compiled binary. This improves reproducibility and avoids embedding machine-specific build paths. The cost is that some debugging information becomes less detailed because local source paths are removed.



## Task 2 — Compose + Healthcheck + Persistent Volume

### Compose file

File: `compose.yaml`

```yaml
services:
  quicknotes:
    build:
      context: ./app
    image: quicknotes:lab6
    ports:
      - "8080:8080"
    environment:
      ADDR: ":8080"
      DATA_PATH: "/data/notes.json"
      SEED_PATH: "/seed.json"
    volumes:
      - quicknotes-data:/data
    restart: unless-stopped
    read_only: true
    tmpfs:
      - /tmp
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    healthcheck:
      test: ["CMD", "/healthcheck"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s

volumes:
  quicknotes-data:
```

### Healthcheck verification

Because the runtime image is distroless and has no shell, the first attempted healthcheck using `/quicknotes -healthcheck` failed. It started another QuickNotes server and caused a port conflict:

```text
listen tcp :8080: bind: address already in use
```

The final strategy was to add a small static Go healthcheck binary at `/healthcheck`. The binary sends an HTTP request to `http://127.0.0.1:8080/health` and exits with status `0` only when the app returns `200 OK`.

Healthcheck source file:

```go
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get("http://127.0.0.1:8080/health")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
```

Compose healthcheck result:

```text
NAME                        IMAGE             COMMAND         SERVICE      STATUS                    PORTS
devops-intro-quicknotes-1   quicknotes:lab6   "/quicknotes"   quicknotes   Up 21 seconds (healthy)   0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp

Health=healthy
```

Container logs:

```text
quicknotes-1  | 2026/06/16 23:28:38 quicknotes listening on :8080 (notes loaded: 7)
```

### Persistence test

Command:

```bash
curl -X POST -H 'Content-Type: application/json' \
  -d '{"title":"durable","body":"survive a restart"}' \
  http://localhost:8080/notes

echo
curl -s http://localhost:8080/notes | grep durable

docker compose down
docker compose up -d
sleep 5
curl -s http://localhost:8080/notes | grep durable

docker compose down -v
docker compose up -d
sleep 5
curl -s http://localhost:8080/notes | grep durable || echo "durable note gone after down -v"
```

Step 1: create the durable note.

```text
{"id":8,"title":"durable","body":"survive a restart","created_at":"2026-06-16T23:30:12.865129533Z"}
```

Step 2: confirm the note exists before restart.

```text
{"id":8,"title":"durable","body":"survive a restart","created_at":"2026-06-16T23:30:12.865129533Z"}
```

Step 3: run `docker compose down`, then `docker compose up -d`. The note still exists, proving that the named volume survived normal Compose shutdown.

```text
[+] down 2/2
 ✔ Container devops-intro-quicknotes-1 Removed
 ✔ Network devops-intro_default        Removed

[+] up 2/2
 ✔ Network devops-intro_default        Created
 ✔ Container devops-intro-quicknotes-1 Started

{"id":8,"title":"durable","body":"survive a restart","created_at":"2026-06-16T23:30:12.865129533Z"}
```

Step 4: run `docker compose down -v`, then `docker compose up -d`. The durable note is gone because the named volume was removed.

```text
[+] down 3/3
 ✔ Container devops-intro-quicknotes-1 Removed
 ✔ Volume devops-intro_quicknotes-data Removed
 ✔ Network devops-intro_default        Removed

[+] up 3/3
 ✔ Network devops-intro_default        Created
 ✔ Volume devops-intro_quicknotes-data Created
 ✔ Container devops-intro-quicknotes-1 Started

durable note gone after down -v
```

### Design question e — Distroless has no shell. How do you healthcheck it?

The chosen strategy is to include a small static Go healthcheck binary in the final image. The binary performs an HTTP GET request to `http://127.0.0.1:8080/health` and exits successfully only if the response status is `200 OK`.

This works with a distroless image because it does not depend on `sh`, `curl`, or `wget`. It also keeps the image minimal while still giving Docker Compose a real application-level healthcheck.

### Design question f — Why does `volumes: [quicknotes-data:/data]` survive `docker compose down`?

A named volume is managed by Docker separately from the container lifecycle. When `docker compose down` removes the container and network, it does not remove named volumes by default. Because QuickNotes stores `notes.json` inside `/data`, and `/data` is backed by the named volume, the note remains after `docker compose down && docker compose up`.

The volume is destroyed by `docker compose down -v`, or by manually removing it with `docker volume rm`.

### Design question g — `depends_on` without `condition: service_healthy`

`depends_on` without `condition: service_healthy` waits only for the dependency container to be started, not for the application inside it to be ready. This can cause a race condition where one service tries to connect to another service before it is actually accepting requests.

In this lab, the `quicknotes` service does not depend on another service, but the healthcheck still matters because it proves that the application is really ready, not just that the container process started.

## Bonus Task — The 6 Security Defaults

### Hardened Compose snippet

The `quicknotes` service in `compose.yaml` applies the security defaults through a non-root image, distroless runtime, dropped capabilities, read-only root filesystem, tmpfs, and `no-new-privileges`.

```yaml
services:
  quicknotes:
    build:
      context: ./app
    image: quicknotes:lab6
    ports:
      - "8080:8080"
    environment:
      ADDR: ":8080"
      DATA_PATH: "/data/notes.json"
      SEED_PATH: "/seed.json"
    volumes:
      - quicknotes-data:/data
    restart: unless-stopped
    read_only: true
    tmpfs:
      - /tmp
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    healthcheck:
      test: ["CMD", "/healthcheck"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s
```

### Verification outputs

#### 1. USER nonroot

Command:

```bash
docker inspect quicknotes:lab6 --format 'User={{ .Config.User }}'
```

Output:

```text
User=nonroot:nonroot
```

#### 2. Distroless image has no shell

Command:

```bash
docker compose exec quicknotes sh || true
```

Output:

```text
OCI runtime exec failed: exec failed: unable to start container process: exec: "sh": executable file not found in $PATH
```

This confirms that the final image does not contain a shell.

#### 3. Capabilities dropped

Command:

```bash
docker inspect devops-intro-quicknotes-1 --format 'CapDrop={{ .HostConfig.CapDrop }}'
```

Output:

```text
CapDrop=[ALL]
```

#### 4. Read-only root filesystem

Command:

```bash
docker inspect devops-intro-quicknotes-1 --format 'ReadonlyRootfs={{ .HostConfig.ReadonlyRootfs }}'
```

Output:

```text
ReadonlyRootfs=true
```

Additional command:

```bash
docker compose exec quicknotes touch /etc/test || true
```

Output:

```text
OCI runtime exec failed: exec failed: unable to start container process: exec: "touch": executable file not found in $PATH
```

Because the image is distroless, there is no `touch` binary available. The read-only root setting is still confirmed directly by Docker inspect with `ReadonlyRootfs=true`.

#### 5. no-new-privileges

Command:

```bash
docker inspect devops-intro-quicknotes-1 --format 'SecurityOpt={{ .HostConfig.SecurityOpt }}'
```

Output:

```text
SecurityOpt=[no-new-privileges:true]
```

### Trivy scan

Command:

```bash
MSYS_NO_PATHCONV=1 docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:0.59.1 image --severity HIGH,CRITICAL --no-progress \
  quicknotes:lab6
```

Output summary:

```text
quicknotes:lab6 (debian 13.5)
=============================
Total: 0 (HIGH: 0, CRITICAL: 0)

healthcheck (gobinary)
======================
Total: 12 (HIGH: 12, CRITICAL: 0)

quicknotes (gobinary)
=====================
Total: 12 (HIGH: 12, CRITICAL: 0)
```

The distroless runtime base had **0 HIGH** and **0 CRITICAL** vulnerabilities. Trivy reported HIGH vulnerabilities in the Go standard library embedded in the compiled `quicknotes` and `healthcheck` binaries. The fix would be to rebuild with a Go version that contains the listed fixes, while still keeping the runtime image distroless and non-root.

### Which default gives the most security per line?

The best security per line is `cap_drop: [ALL]` together with `no-new-privileges:true`. Dropping all capabilities removes unnecessary Linux privileges from the container, and QuickNotes does not need any extra capabilities to serve HTTP traffic. `no-new-privileges` is also valuable because it prevents the process from gaining extra privileges through exec or setuid-style behavior. Distroless also gives strong security benefits because it removes the shell and package manager, which reduces the attack surface.
