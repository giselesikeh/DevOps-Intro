# Lab 8 — SRE & Monitoring: Golden Signals Dashboard + One Good Alert

## Task 1 — Prometheus + Grafana with a Provisioned Dashboard

### Summary

For Task 1, I extended the Lab 6 QuickNotes Compose setup by adding Prometheus and Grafana services. Prometheus scrapes the QuickNotes `/metrics` endpoint using the Compose service name `quicknotes:8080`, and Grafana is provisioned from files with a Prometheus data source and a Golden Signals dashboard.

The stack was started with:

```bash
docker compose up -d --build
```

The running services were:

```text
devops-intro-grafana-1      grafana/grafana:13.0      Up
devops-intro-prometheus-1   prom/prometheus:v3.11.3   Up
devops-intro-quicknotes-1   quicknotes:lab8           Up (healthy)
```

---

### Files added or changed

```text
compose.yaml
app/Dockerfile
app/cmd/healthcheck/main.go
monitoring/prometheus/prometheus.yml
monitoring/grafana/provisioning/datasources/datasource.yml
monitoring/grafana/provisioning/dashboards/dashboard.yml
monitoring/grafana/dashboards/golden-signals.json
submissions/lab8.md
```

`app/Dockerfile` and `app/cmd/healthcheck/main.go` are reused from the Lab 6 container setup so that QuickNotes can build correctly and expose a healthy Compose service for Prometheus to depend on.

---

### `compose.yaml`

```yaml
services:
  quicknotes:
    build:
      context: ./app
    image: quicknotes:lab8
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

  prometheus:
    image: prom/prometheus:v3.11.3
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
    depends_on:
      quicknotes:
        condition: service_healthy
    restart: unless-stopped

  grafana:
    image: grafana/grafana:13.0
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_USER: "lab8admin"
      GF_SECURITY_ADMIN_PASSWORD: "lab8-change-me-2026"
      GF_USERS_ALLOW_SIGN_UP: "false"
    volumes:
      - ./monitoring/grafana/provisioning:/etc/grafana/provisioning:ro
      - ./monitoring/grafana/dashboards:/var/lib/grafana/dashboards:ro
    depends_on:
      - prometheus
    restart: unless-stopped

volumes:
  quicknotes-data:
```

---

### `monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "quicknotes"
    metrics_path: /metrics
    static_configs:
      - targets:
          - "quicknotes:8080"
```

This config sets the global scrape interval to 15 seconds and defines one scrape job for QuickNotes. The target uses the Compose service name `quicknotes` instead of `localhost`, because Prometheus runs inside the Compose network.

---

### `monitoring/grafana/provisioning/datasources/datasource.yml`

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    uid: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

This file provisions Prometheus as the default Grafana data source. The URL uses the Compose service name `prometheus`.

---

### `monitoring/grafana/provisioning/dashboards/dashboard.yml`

```yaml
apiVersion: 1

providers:
  - name: "QuickNotes Golden Signals"
    orgId: 1
    folder: "QuickNotes"
    type: file
    disableDeletion: false
    editable: true
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
```

This file tells Grafana to load dashboards from `/var/lib/grafana/dashboards`, where the dashboard JSON is mounted by Compose.

---

### `monitoring/grafana/dashboards/golden-signals.json`

The dashboard contains four panels:

| Golden signal | Panel                                                                      | PromQL                                                             |                                                                                |
| ------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| Latency       | Request-rate proxy because QuickNotes does not expose a duration histogram | `sum(rate(quicknotes_http_requests_total[1m]))`                    |                                                                                |
| Traffic       | Request rate                                                               | `sum(rate(quicknotes_http_requests_total[1m]))`                    |                                                                                |
| Errors        | 4xx + 5xx error ratio                                                      | `100 * sum(rate(quicknotes_http_responses_by_code_total{code=~"4.. | 5.."}[5m])) / clamp_min(sum(rate(quicknotes_http_requests_total[5m])), 0.001)` |
| Saturation    | Notes stored                                                               | `quicknotes_notes_total`                                           |                                                                                |

QuickNotes exposes request counters and note gauges, but it does not expose a request-duration histogram. Because of that, the latency panel uses the allowed request-rate proxy.

---

### Traffic generation

I generated traffic using:

```bash
for i in {1..200}; do
  curl -s http://localhost:8080/health > /dev/null
  curl -s http://localhost:8080/notes > /dev/null
done
```

Earlier, I also generated mixed traffic with `POST /notes` requests to make the traffic and error panels non-empty.

---

### Prometheus target evidence

Prometheus target page:

```text
http://localhost:9090/targets
```

The Prometheus target page showed:

```text
quicknotes — 1 / 1 up
Endpoint: http://quicknotes:8080/metrics
State: UP
```

Command used:

```bash
curl -s http://localhost:9090/api/v1/targets | grep -o '"health":"[^"]*"'
```

Output:

```text
"health":"up"
```

Screenshot evidence:

```text
Prometheus showed the QuickNotes target as UP at http://quicknotes:8080/metrics.
```

---

### Grafana dashboard evidence

Grafana was available at:

```text
http://localhost:3000
```

Login used:

```text
username: lab8admin
password: lab8-change-me-2026
```

The provisioned dashboard loaded automatically under:

```text
Dashboards → QuickNotes → QuickNotes Golden Signals
```

The dashboard showed four panels:

1. Latency — request-rate proxy because duration histogram is unavailable
2. Traffic — request rate
3. Errors — 4xx + 5xx ratio
4. Saturation — notes stored

After traffic generation, the dashboard showed non-trivial graphs for request traffic and notes stored, and the error panel showed 0% when only healthy requests were sent.

Screenshot evidence:

```text
Grafana showed the QuickNotes Golden Signals dashboard with traffic data in the panels.
```

---

### Design questions

#### a) Pull vs push

Prometheus uses a pull model, so Prometheus must be able to reach the QuickNotes `/metrics` endpoint. In this Compose setup, Prometheus reaches QuickNotes through the internal Compose DNS name `quicknotes:8080`. QuickNotes does not push metrics to Prometheus; it only exposes them. If Prometheus cannot reach QuickNotes, the scrape target becomes `DOWN`, the `up` metric becomes `0`, and Grafana panels depending on those metrics either stop updating or show no recent data.

#### b) `scrape_interval: 15s`, `5s`, and `5m`

A 15-second scrape interval is a reasonable default because it gives enough data points for short PromQL windows without creating too much overhead. If the scrape interval is reduced to 5 seconds, Prometheus collects more data and produces more detailed graphs, but this increases storage, CPU usage, and query cost. It can also make noisy short-term changes look more important than they are. If the scrape interval is increased to 5 minutes, many short incidents can be missed entirely, and `rate()` over short windows like `[1m]` becomes unreliable because there may be too few samples to calculate a meaningful rate.

#### c) `rate()` vs `irate()` vs `delta()`

For the Traffic panel, `rate()` is the right choice because `quicknotes_http_requests_total` is a counter. `rate()` calculates the average per-second increase over a time window and smooths short spikes, which makes it suitable for dashboard traffic trends. `irate()` only uses the last two samples, so it is more jumpy and better for very short-lived debugging rather than a stable dashboard. `delta()` gives the raw difference over a time range, but it is not the best fit for a requests-per-second traffic panel.

#### d) Why provision Grafana from files?

Provisioning Grafana from files makes the dashboard reproducible. A fresh `docker compose up` can recreate the same data source and dashboard without manual clicking. It also means the dashboard JSON and provisioning YAML can be committed, reviewed in a PR, versioned with the code, and restored if the stack is recreated. This is better than manually building dashboards in the UI every time because manual setup is easy to forget, hard to review, and hard to reproduce.


---

## Task 2 — One Good Alert + Runbook

### Summary

For Task 2, I configured one Prometheus alert rule for QuickNotes. The alert fires when the HTTP error ratio is greater than 5% for 5 minutes. The alert has a `severity: page` label and links to the runbook at `docs/runbook/high-error-rate.md`.

The alert was deliberately triggered by sending malformed JSON requests to `POST /notes` for more than 5 minutes. The alert first entered the pending state and then moved to the firing state.

---

### Alert rule file

File:

```text
monitoring/prometheus/rules/quicknotes-alerts.yml
```

Content:

```yaml
groups:
  - name: quicknotes-alerts
    rules:
      - alert: QuickNotesHighErrorRate
        expr: |
          (
            sum(rate(quicknotes_http_responses_by_code_total{code=~"4..|5.."}[5m]))
            /
            clamp_min(sum(rate(quicknotes_http_requests_total[5m])), 0.001)
          ) > 0.05
        for: 5m
        labels:
          severity: page
        annotations:
          summary: "QuickNotes high HTTP error rate"
          description: "More than 5% of QuickNotes HTTP requests have returned 4xx or 5xx responses for at least 5 minutes."
          runbook: "docs/runbook/high-error-rate.md"
```

This alert is symptom-based because it watches the user-visible HTTP error ratio instead of a lower-level cause such as CPU usage or container restart count.

---

### Prometheus rule loading

Prometheus was configured to load alert rules from:

```yaml
rule_files:
  - /etc/prometheus/rules/*.yml
```

The Compose file mounts the local rules directory into Prometheus:

```yaml
volumes:
  - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
  - ./monitoring/prometheus/rules:/etc/prometheus/rules:ro
```

I verified that Prometheus loaded the alert rule using:

```bash
curl -s http://localhost:9090/api/v1/rules | grep -o 'QuickNotesHighErrorRate'
```

Output:

```text
QuickNotesHighErrorRate
```

---

### Triggering the alert deliberately

I triggered the alert by sending malformed JSON to `POST /notes` for more than 5 minutes:

```bash
START=$(date +%s)

while [ $(( $(date +%s) - START )) -lt 420 ]; do
  curl -s -o /dev/null -w "bad POST status=%{http_code}\n" \
    -X POST http://localhost:8080/notes \
    -H "Content-Type: application/json" \
    -d '{"title":'
  sleep 1
done
```

The malformed JSON requests produced 4xx responses. Because this traffic continued for more than the alert's `for: 5m` duration, the alert changed from inactive to pending and then to firing.

---

### Alert firing evidence

Prometheus alerts page:

```text
http://localhost:9090/alerts
```

The alert page showed:

```text
quicknotes-alerts
QuickNotesHighErrorRate
FIRING
```

Terminal confirmation:

```bash
curl -s http://localhost:9090/api/v1/alerts | grep -o '"state":"[^"]*"'
```

Output:

```text
"state":"firing"
```

I also checked the live error ratio with:

```bash
curl -G -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=100 * sum(rate(quicknotes_http_responses_by_code_total{code=~"4..|5.."}[5m])) / clamp_min(sum(rate(quicknotes_http_requests_total[5m])), 0.001)'
```

Output showed the error ratio was approximately:

```text
79.65%
```

This is above the required 5% threshold, so the firing state is expected.

Screenshot evidence:

```text
The Prometheus alerts page showed QuickNotesHighErrorRate in FIRING state.
```

---

### Runbook

File:

```text
docs/runbook/high-error-rate.md
```

Content:

```markdown
# Runbook — QuickNotes High Error Rate

## What this alert means

This alert means more than 5% of QuickNotes HTTP requests are returning 4xx or 5xx responses for at least 5 minutes.

## Triage steps

1. Open the Prometheus alerts page at `http://localhost:9090/alerts` and confirm that `QuickNotesHighErrorRate` is firing.
2. Open the Grafana dashboard at `http://localhost:3000` and check the error ratio, traffic rate, and notes stored panels.
3. Check whether the errors are mostly caused by bad client requests or server failures by querying this metric in Prometheus: `sum by (code) (rate(quicknotes_http_responses_by_code_total[5m]))`.
4. Check whether QuickNotes is still healthy by running `curl http://localhost:8080/health` and `docker compose ps`.
5. Check recent QuickNotes logs by running `docker compose logs --tail=100 quicknotes`.

## Mitigations

1. If the errors are caused by a bad deploy or bad configuration, roll back to the last known working Compose configuration or image.
2. If the errors are caused by malformed traffic, reduce or stop that traffic source and verify that healthy requests still work.
3. If QuickNotes is unhealthy or stuck, restart only the QuickNotes service with `docker compose restart quicknotes`.
4. If the data file or volume looks corrupted, stop writes temporarily, back up the current volume, and restore from a known good seed or backup.

## Post-incident

After the service is stable, write a blameless postmortem using the Lecture 1 postmortem format. Include the timeline, impact, root cause, what worked, what did not work, and concrete action items to prevent the same incident from happening again.
```

---

### Design questions

#### e) Why sustained for 5 minutes?

The alert uses a 5-minute sustained breach so that it does not page on one bad request or a very short burst of malformed traffic. A single 4xx response can happen because of normal user mistakes, browser retries, or one bad client. Waiting for 5 minutes makes the alert more reliable because it only fires when the error rate stays high long enough to suggest real user impact.

#### f) Symptom alert vs cause alert

The high error rate alert is a symptom alert because it watches what users experience: failed HTTP requests. A cause alert someone might write for QuickNotes is `CPU > 80%` or `container restarted once`. This is worse as a paging alert because high CPU may happen during normal traffic spikes, backups, or short maintenance tasks without users seeing errors. A symptom alert is better because it focuses on user-visible failure instead of guessing that a low-level metric is harmful.

#### g) Alert fatigue threshold

I would consider this alert too noisy if more than 20% of its pages happen when users were not actually affected. For a paging alert with `severity: page`, most alerts should represent real user impact or an urgent risk to the service. If one out of every five pages is false or non-actionable, on-call engineers will start trusting the alert less, which creates alert fatigue.


---

## Bonus Task — Synthetic Monitoring from the Outside

### Summary

For the bonus task, I exposed the local QuickNotes service using ngrok and configured a Checkly API check to monitor the service externally. The check polls the public QuickNotes `/health` endpoint every 1 minute from two regions: Frankfurt and Singapore.

The goal was to compare what Prometheus sees from inside the Docker Compose network with what Checkly sees from outside the local machine.

---

### Public QuickNotes URL

The local QuickNotes service was exposed with ngrok:

```bash id="smppnu"
"/f/Innopolis 3 year/Devops/ngrok.exe" http 8080
```

ngrok forwarding URL:

```text id="ak9d5k"
https://grudge-catalyst-overfed.ngrok-free.dev -> http://localhost:8080
```

Public health endpoint:

```text id="fhi0k3"
https://grudge-catalyst-overfed.ngrok-free.dev/health
```

I verified the public endpoint with:

```bash id="4l0gx1"
curl https://grudge-catalyst-overfed.ngrok-free.dev/health
```

Output:

```json id="qeh5oy"
{"notes":7,"status":"ok"}
```

---

### Checkly configuration

I configured a Checkly API check with the following settings:

```text id="xazvdf"
Name: API Check #1
Method: GET
URL: https://grudge-catalyst-overfed.ngrok-free.dev/health
Frequency: 1 minute
Scheduling strategy: Parallel runs
Regions: Frankfurt and Singapore
```

Request header used to bypass the ngrok browser warning:

```text id="9puyum"
ngrok-skip-browser-warning: true
```

Assertions:

```text id="z4pk6o"
Status code equals 200
Response time less than 2000 ms
```

The Checkly test request returned:

```text id="gi6obz"
Status: 200
Response time: 154 ms
Response body: {"notes": 7, "status": "ok"}
```

After running, the Checkly results showed:

```text id="td1shs"
Availability: 100%
Average latency: 316 ms
P95 latency: 529 ms
Failing checks: 0
Frequency: 1 min
```

---

### Prometheus internal measurements

I collected internal Prometheus values using:

```bash id="bq0bvf"
curl -G -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=sum(rate(quicknotes_http_requests_total[30m]))'

curl -G -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=100 * sum(rate(quicknotes_http_responses_by_code_total{code=~"4..|5.."}[30m])) / clamp_min(sum(rate(quicknotes_http_requests_total[30m])), 0.001)'

curl -G -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=scrape_duration_seconds{job="quicknotes"}'

curl -G -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=quicknotes_notes_total'
```

Results:

```text id="jnx9fc"
Internal request rate: 0.216 req/s
Internal error ratio: 0%
Internal scrape duration: 0.000944481 s
Notes stored: 7
```

QuickNotes does not expose a request-duration histogram, so Prometheus cannot calculate real internal p50 or p95 application latency. For this reason, I used Checkly for external p50/p95-style response-time evidence and Prometheus for internal request rate, error ratio, scrape health, and service metrics.

---

### Internal vs external comparison

| Metric                   |                                           Prometheus inside Compose network |                                         Checkly from 2 regions |
| ------------------------ | --------------------------------------------------------------------------: | -------------------------------------------------------------: |
| Avg latency p50          | Not available because QuickNotes does not expose request-duration histogram | External response time observed by Checkly; average was 316 ms |
| Avg latency p95          | Not available because QuickNotes does not expose request-duration histogram |                                                         529 ms |
| Errors observed          |                            0% internal error ratio over the measured window |                            0 failing checks, 100% availability |
| Request rate / frequency |                                           0.216 req/s internal request rate |                             1-minute synthetic check frequency |
| Health evidence          |          Prometheus scrape target was up and `quicknotes_notes_total` was 7 |   Checkly received HTTP 200 from the public `/health` endpoint |

---

### Failure-mode comparison

Checkly can catch failures that Prometheus may not see because it checks the service from outside the Docker Compose network. For example, if the application is healthy internally but the public tunnel, DNS, firewall, or external network path is broken, Prometheus may still show QuickNotes as healthy while Checkly fails. Checkly is therefore useful for detecting user-facing availability problems from real external regions.

Prometheus can catch internal service and metric problems that Checkly cannot see. For example, Prometheus can show scrape target health, internal request rate, error ratio, and service-specific metrics like `quicknotes_notes_total`. Checkly only sees the external `/health` response, so it cannot explain whether an issue is caused by internal application errors, bad response-code distribution, or missing metrics. Both tools are useful together because Prometheus gives internal visibility while Checkly gives an outside-user view.
