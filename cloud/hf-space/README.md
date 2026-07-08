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
