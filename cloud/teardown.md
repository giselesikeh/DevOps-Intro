# Lab 10 Teardown Notes

## Local QuickNotes container

Stop and remove the local Cloudflare bonus container:

    docker rm -f quicknotes-cloudflare

## Cloudflare quick tunnel

The Cloudflare quick tunnel is ephemeral and stops when the cloudflared tunnel process is stopped.

To stop it, press Ctrl + C in the terminal running cloudflared.

## Hugging Face Space

The Hugging Face Space can be deleted from:

    https://huggingface.co/spaces/Gisele/quicknotes-lab10/settings

The Space costs nothing on the free tier, so I left it available for grading.
