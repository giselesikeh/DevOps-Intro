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
