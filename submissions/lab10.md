# Lab 10 — Cloud Computing: Ship QuickNotes to a Real Cloud

## Task 1 — CI-Automated Push to GHCR

### Release workflow

The release workflow is stored at:

```text
.github/workflows/release.yml
```

It triggers on Git tags matching `v*`, validates semver-style tags such as `v0.1.0`, builds the Docker image from `app/`, logs in to GitHub Container Registry using the repository `GITHUB_TOKEN`, and pushes both the immutable release tag and `latest`.

The workflow uses minimum permissions:

```yaml
permissions:
  contents: read
  packages: write
```

No third-party GitHub Actions are used in this workflow. The workflow uses shell commands directly, so there are no unpinned third-party actions.

### Release tag

```text
v0.1.0
```

### Green release run

```text
https://github.com/giselesikeh/DevOps-Intro/actions/runs/28524079963
```

The release workflow completed successfully:

```text
Status: Success
Duration: 30s
Commit: 2cf5705
Tag: v0.1.0
Workflow: Release QuickNotes Image
```

### Registry image

```text
ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
ghcr.io/giselesikeh/devops-intro/quicknotes:latest
```

### Local Docker build verification

Before pushing the release tag, I tested the Docker image locally:

```bash
docker build -t quicknotes:lab10 ./app
docker run -d --name quicknotes-lab10 -p 8081:8080 quicknotes:lab10
curl -fsS http://localhost:8081/health
curl -fsS http://localhost:8081/notes
docker inspect quicknotes-lab10 --format 'Image={{ .Config.Image }} User={{ .Config.User }} ExposedPorts={{ json .Config.ExposedPorts }} Healthcheck={{ json .Config.Healthcheck.Test }}'
docker rm -f quicknotes-lab10
```

Health output:

```json
{"notes":4,"status":"ok"}
```

Container inspection:

```text
Image=quicknotes:lab10 User=quicknotes ExposedPorts={"8080/tcp":{}} Healthcheck=["CMD-SHELL","wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1"]
```

### Clean public pull evidence

I logged out of GHCR first to verify that the image is publicly pullable without authentication:

```bash
docker logout ghcr.io || true
docker rmi ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0 2>/dev/null || true
docker pull ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
```

Output:

```text
Removing login credentials for ghcr.io

v0.1.0: Pulling from giselesikeh/devops-intro/quicknotes
cc027f30a565: Pull complete
e3f1a09a754b: Pull complete
47eadbc14fb7: Pull complete
cdf53fd9d1cc: Pull complete
a530cd76cb77: Pull complete
Digest: sha256:d19241b2ef1e43cd3f1a37de25eeb01a6456b7087d2296beeabb80d456d713e0
Status: Downloaded newer image for ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
```

### Runtime verification from pulled image

I started the pulled image on host port `8082`:

```bash
docker run -d --name quicknotes-ghcr-test -p 8082:8080 ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
curl -fsS http://localhost:8082/health
curl -fsS http://localhost:8082/notes
docker inspect quicknotes-ghcr-test --format 'Image={{ .Config.Image }} User={{ .Config.User }} ExposedPorts={{ json .Config.ExposedPorts }} Healthcheck={{ json .Config.Healthcheck.Test }}'
docker rm -f quicknotes-ghcr-test
```

Health output:

```json
{"notes":4,"status":"ok"}
```

The `/notes` endpoint returned the seeded QuickNotes data successfully.

Container inspection:

```text
Image=ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0 User=quicknotes ExposedPorts={"8080/tcp":{}} Healthcheck=["CMD-SHELL","wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1"]
```

### Design question a — OIDC vs `GITHUB_TOKEN`

For pushing to GHCR from the same GitHub repository, `GITHUB_TOKEN` with `packages: write` is enough because GitHub issues it automatically for the workflow and scopes it to the repository. I would use OIDC when deploying to an external cloud provider such as AWS, GCP, or Azure. OIDC lets the workflow exchange its GitHub identity for short-lived cloud credentials at runtime, without storing a long-lived service-account JSON key or static cloud secret in GitHub. This reduces the damage if repository secrets or workflow logs are exposed because there is no persistent cloud credential to steal.

### Design question b — `latest` tag vs immutable `v0.1.0` tag

The immutable tag `v0.1.0` is the safe release reference because it identifies exactly one version of the image and supports rollback, auditing, and reproducible deployment. The `latest` tag is still useful as a convenience pointer for humans, demos, simple development environments, and quick smoke tests that should always use the newest release. In production, I would pin deployments to immutable version tags, while still publishing `latest` as a friendly moving alias.

### Design question c — `packages: write` scope only

This follows the least-privilege principle: the workflow should receive only the permissions required to do its job. For this release workflow, it only needs to read repository contents and write container packages. Using `packages: write` instead of broad permissions such as `write-all` limits the impact of a compromised workflow. An attacker who gets code execution inside this workflow can push or overwrite package images, but should not automatically gain permission to modify repository code, create releases, change issues, alter pull requests, or write unrelated GitHub resources.


## Task 2 — Deploy to Hugging Face Spaces

### Hugging Face Space

The QuickNotes image is deployed to a public Hugging Face Space using the Docker SDK.

```text
https://huggingface.co/spaces/Gisele/quicknotes-lab10
```

Public app URL:

```text
https://gisele-quicknotes-lab10.hf.space
```

### Space repository files

The Space repository contains a small `Dockerfile` that pulls the already-built immutable GHCR image instead of rebuilding from source inside Hugging Face.

I chose to pull from GHCR because the image was already built and tested by the release workflow. This improves reproducibility: Hugging Face runs the same artifact that was produced by CI, rather than building a potentially different image from source.

#### `Dockerfile`

```dockerfile
FROM ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0

ENV ADDR=:8080
EXPOSE 8080
```

#### `README.md`

```markdown
---
title: QuickNotes Lab 10
emoji: 📝
colorFrom: blue
colorTo: green
sdk: docker
app_port: 8080
pinned: false
license: mit
short_description: QuickNotes deployed from GHCR for DevOps Lab 10
---

# QuickNotes Lab 10

This Hugging Face Space runs the QuickNotes container image built and published by the Lab 10 release workflow.

## Image

ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0

## Endpoints

- GET /health
- GET /notes
- GET /metrics
- POST /notes
- DELETE /notes/{id}

The Space uses the Docker SDK and sets app_port: 8080 because QuickNotes listens on port 8080.
```

### Push to Hugging Face Space

The Space repository was cloned, the Docker files were copied in, committed, and pushed:

```text
4448ab1 deploy quicknotes docker image
```

Successful push output:

```text
To https://huggingface.co/spaces/Gisele/quicknotes-lab10
   2546334..4448ab1  main -> main
```

### Public endpoint verification

The root path returned `404`, which is expected because QuickNotes does not define a `/` route.

```text
https://gisele-quicknotes-lab10.hf.space/
HTTP/1.1 404 Not Found
404 page not found
```

The `/health` endpoint worked:

```bash
curl -i https://gisele-quicknotes-lab10.hf.space/health
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 26

{"notes":4,"status":"ok"}
```

The `/notes` endpoint worked:

```bash
curl -i https://gisele-quicknotes-lab10.hf.space/notes
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 635
```

The endpoint returned the seeded QuickNotes data successfully.

The `/metrics` endpoint also worked:

```bash
curl -i https://gisele-quicknotes-lab10.hf.space/metrics
```

Output excerpt:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; version=0.0.4

# HELP quicknotes_notes_total Notes currently stored.
# TYPE quicknotes_notes_total gauge
quicknotes_notes_total 4
# HELP quicknotes_notes_created_total Notes created since process start.
# TYPE quicknotes_notes_created_total counter
quicknotes_notes_created_total 0
```

### Warm latency measurements

I made 5 consecutive warm requests to `/health`:

```bash
HF_URL="https://gisele-quicknotes-lab10.hf.space"

for i in 1 2 3 4 5; do
  curl -o /dev/null -s -w "run_${i} time_total=%{time_total}s http_code=%{http_code}\n" "$HF_URL/health"
done
```

Results:

```text
run_1 time_total=0.480838s http_code=200
run_2 time_total=0.491241s http_code=200
run_3 time_total=0.465115s http_code=200
run_4 time_total=0.388214s http_code=200
run_5 time_total=0.465803s http_code=200
```

Sorted values:

```text
0.388214s, 0.465115s, 0.465803s, 0.480838s, 0.491241s
```

Warm p50 latency:

```text
0.465803s
```

### Cold-start measurements

For each cold measurement, I left the Space idle for at least 35 minutes before sending a request to `/health`.

#### Cold measurement 1

```text
Wed Jul  1 18:56:35 RTZST 2026
cold_1 time_total=0.777476s http_code=200
after_cold_1_warm time_total=0.407310s http_code=200
```

#### Cold measurement 2

```text
Wed Jul  1 19:32:55 RTZST 2026
cold_2 time_total=0.863421s http_code=200
after_cold_2_warm time_total=0.807086s http_code=200
```

#### Cold measurement 3

```text
Wed Jul  1 20:41:07 RTZST 2026
cold_3 time_total=0.782021s http_code=200
after_cold_3_warm time_total=0.539640s http_code=200
```

Cold-start summary:

| Measurement | Cold latency | HTTP code |
|---|---:|---:|
| cold 1 | 0.777476s | 200 |
| cold 2 | 0.863421s | 200 |
| cold 3 | 0.782021s | 200 |

### Design question d — HF Spaces sleep vs Cloud Run scale-to-zero

HF Spaces sleep and Cloud Run scale-to-zero follow the same general idea: when there is no traffic, the platform stops keeping an active container ready, and the next request has to wake or start the workload again.

HF Spaces wake-up can be slower because the platform is optimized for free hosted demos, ML apps, notebooks, and community Spaces rather than low-latency production serving. A sleeping Space may need to be scheduled again, pull or prepare the container environment, and start the app before traffic is served. Cloud Run is designed as a production serverless container platform, so it optimizes more heavily for fast request routing, autoscaling, and predictable cold starts.

### Design question e — Why `app_port: 8080` is needed

QuickNotes listens on port `8080`. Hugging Face Docker Spaces default to port `7860`, which is common for Gradio-style demo apps. Without setting `app_port: 8080`, Hugging Face would route traffic to the wrong port and the app would not be reachable. The `app_port: 8080` setting tells Hugging Face to send external requests to the port where QuickNotes is actually listening.

### Design question f — Pulling from GHCR vs building inside the Space

Pulling the image from GHCR makes the Space run the exact image produced by the release CI workflow. This is better for reproducibility because the same immutable `v0.1.0` artifact is tested locally, pulled from GHCR, and deployed to Hugging Face.

Building inside the Space can be easier for debugging because the source and build logs are directly in the Space, but it duplicates the build process and can produce a different artifact if the build environment, base images, or dependencies change. Pulling from GHCR also makes deployment faster when the image is already available and cached.


## Bonus Task — Cloudflare Tunnel + Cross-Platform Comparison

### Goal

The goal of the bonus task was to expose the same QuickNotes image through a Cloudflare quick tunnel and compare its latency against the Hugging Face Spaces deployment.

The local QuickNotes container was started from the same immutable GHCR image used for Hugging Face Spaces:

```text
ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
```

### Local QuickNotes container verification

I first stopped the old container that was using port `8080`, then started QuickNotes from the GHCR image:

```bash
docker rm -f quicknotes-cloudflare 2>/dev/null || true

docker run -d \
  --name quicknotes-cloudflare \
  -p 8080:8080 \
  ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0
```

Local `/health` worked:

```bash
curl -fsS http://localhost:8080/health
```

Output:

```json
{"notes":4,"status":"ok"}
```

Local `/notes` also returned the seeded QuickNotes data successfully.

The running container was confirmed:

```bash
docker ps --filter name=quicknotes-cloudflare --format "table {{.ID}}\t{{.Image}}\t{{.Names}}\t{{.Ports}}"
```

Output:

```text
CONTAINER ID   IMAGE                                                     NAMES                   PORTS
2748e24a63a7   ghcr.io/giselesikeh/devops-intro/quicknotes:v0.1.0        quicknotes-cloudflare   0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp
```

### Tool versions

`cloudflared` and `hyperfine` were installed with `winget` and verified:

```bash
cloudflared --version
hyperfine --version
```

Output:

```text
cloudflared version 2026.6.1
hyperfine 1.20.0
```

### Cloudflare quick tunnel attempt 1

I started a Cloudflare quick tunnel:

```bash
cloudflared tunnel --url http://localhost:8080
```

Cloudflare generated a quick tunnel URL:

```text
https://science-supplier-allocated-real.trycloudflare.com
```

However, the tunnel logs showed connectivity pre-check failures:

```text
UDP Connectivity region2.v2.argotunnel.com FAIL
TCP Connectivity region2.v2.argotunnel.com FAIL
ERROR: Allow outbound QUIC traffic on port 7844 or use HTTP2.
ERROR: Allow outbound TCP on port 7844.
SUMMARY: Environment has critical failures. cloudflared may not be able to establish a tunnel.
```

Testing the generated public URL failed:

```bash
curl -i https://science-supplier-allocated-real.trycloudflare.com/health
curl -i https://science-supplier-allocated-real.trycloudflare.com/notes
```

Output showed Cloudflare tunnel errors:

```text
HTTP/1.1 530
<title>Cloudflare Tunnel error | science-supplier-allocated-real.trycloudflare.com | Cloudflare</title>
curl: (56) Recv failure: Connection was reset
```

Later the tunnel hostname also stopped resolving:

```text
curl: (6) Could not resolve host: science-supplier-allocated-real.trycloudflare.com
```

### Cloudflare quick tunnel attempt 2 — HTTP/2 mode

Because the first attempt showed QUIC/UDP issues, I retried using HTTP/2 mode:

```bash
cloudflared tunnel --protocol http2 --edge-ip-version 4 --url http://localhost:8080
```

This failed before producing a usable tunnel URL:

```text
failed to request quick Tunnel: Post "https://api.trycloudflare.com/tunnel": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

### Cloudflare quick tunnel attempt 3 — no autoupdate + HTTP/2

I also retried with `--no-autoupdate`:

```bash
cloudflared tunnel --no-autoupdate --protocol http2 --edge-ip-version 4 --url http://localhost:8080 --loglevel info
```

This also failed at the quick tunnel request stage:

```text
failed to request quick Tunnel: Post "https://api.trycloudflare.com/tunnel": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

### Cloudflare API reachability check

I checked whether the Cloudflare quick tunnel API host was reachable:

```bash
curl -I https://api.trycloudflare.com
```

Output:

```text
HTTP/1.1 404 Not Found
Date: Wed, 01 Jul 2026 19:18:21 GMT
Connection: keep-alive
Server: cloudflare
CF-RAY: a147c1912d35b497-ARN
```

This shows that the host was reachable over HTTPS, but the quick tunnel creation request still timed out. Therefore, the failure appears to be specific to Cloudflare quick tunnel creation/connectivity from the current network environment, not a QuickNotes application failure.

### Bonus result

The local QuickNotes container was successfully running from the GHCR release image, but Cloudflare quick tunnel could not be kept reachable from this network. Because the public tunnel could not be established reliably, I could not verify it from a phone/cellular network or collect valid Cloudflare Tunnel latency measurements.

### Cross-platform comparison table

| Metric | HF Spaces (hosted) | Cloudflare Tunnel (local-via-edge) |
|--------|-------------------:|-----------------------------------:|
| Warm p50 | 0.465803s | Not measured — tunnel failed |
| Warm p95 | Not measured with 50-run benchmark | Not measured — tunnel failed |
| Cold start | 0.777476s, 0.863421s, 0.782021s | N/A — local container would remain running |
| Public URL stability | Stable: `https://gisele-quicknotes-lab10.hf.space` | Ephemeral; generated URL failed with Cloudflare 530 |
| Cost | Free | Free |
| Status | Completed successfully | Attempted, blocked by tunnel connectivity |

### Design question g — Architectural difference

In Hugging Face Spaces, the container runs in Hugging Face infrastructure, so the application is hosted by the cloud platform itself. In Cloudflare Tunnel, the container runs locally on my laptop, while Cloudflare only provides the public edge URL and proxy path back to the local machine.

For users, the distinction matters less than reachability, latency, and reliability. If the public URL works and the service is reliable, users may not care where the container physically runs. For operations, the distinction matters a lot: in the Hugging Face case, uptime depends mainly on Hugging Face; in the tunnel case, uptime depends on my laptop, Docker, local network, power, and the Cloudflare tunnel connection.

### Design question h — Latency dominator

For Hugging Face Spaces, warm latency is mainly dominated by the hosted platform routing path, geographic distance to the Space, and the Hugging Face proxy/container runtime overhead. Cold latency is dominated by waking the sleeping Space and starting or preparing the container.

For Cloudflare Tunnel, if it had worked, the latency would be dominated by the route from the user to Cloudflare’s edge, then from Cloudflare back through the tunnel to my laptop, plus my local network upload path. Unlike HF Spaces, the application would already be running locally, so there would be no container cold start, but there would be an extra edge-to-local tunnel hop.

### Design question i — When Cloudflare Tunnel is the right production pick

Cloudflare Tunnel can be a good production choice for exposing private or on-prem services without opening inbound firewall ports. It is useful for home labs, internal tools, demos, staging environments, stakeholder reviews, and services that must stay inside a private network but still need controlled external access.

It is not the right production pick when the service depends on a developer laptop, an unstable local network, or an ephemeral quick tunnel URL. For serious production use, I would use a named Cloudflare Tunnel with a managed domain, proper access controls, monitoring, and a reliable always-on host. I would not use a temporary quick tunnel as the production endpoint for a public user-facing application.

