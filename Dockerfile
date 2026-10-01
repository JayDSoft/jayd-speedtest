# ARMv7-only image for Sonoff iHost / eWeLink CUBE.
# Base image already contains the official Ookla Speedtest CLI.
FROM --platform=linux/arm/v7 lferrarotti74/speedtest-ookla:latest

COPY --chown=speedtest:speedtest speedtest-web-armv7 /usr/local/bin/speedtest-web

EXPOSE 8080
ENV PORT=8080 \
    TEST_TIMEOUT_SECONDS=180

ENTRYPOINT ["/usr/local/bin/speedtest-web"]
