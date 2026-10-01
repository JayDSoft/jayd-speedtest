# JayD Speedtest

Lightweight ARMv7 web interface for running the official Ookla Speedtest CLI on demand.

Designed for Sonoff iHost / eWeLink CUBE and other `linux/arm/v7` Docker hosts.

## Features

- Manual speed test only — no history, database, cron, or background tests
- Lightweight single-binary Go web UI
- Official Ookla Speedtest CLI
- Progress bar and current test stage
- English and Russian interface

## Ports

The container listens on port `8080`.

Example for eWeLink CUBE:

- Host port: `8765`
- Add-on/container port: `8080`
- Network: `bridge`

Open:

`http://<iHost-IP>:8765`

## Language

The interface language is selected with the `LANG` environment variable when the container is started.

- English: `LANG=en`
- Russian: `LANG=ru`
- If `LANG` is not set, English is used by default.

In eWeLink CUBE, add the environment variable under the container's **Environment variables** settings before starting it.

Examples:

```text
LANG=en
```

or

```text
LANG=ru
```

Common locale-style Russian values such as `ru_RU.UTF-8` are also recognized as Russian. Any other value falls back to English.

## Other environment variables

- `PORT=8080` — internal HTTP port
- `TEST_TIMEOUT_SECONDS=180` — maximum duration of one speed test
