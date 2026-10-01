$ErrorActionPreference = "Stop"
docker buildx build --platform linux/arm/v7 -t jayd-speedtest-web:armv7 --load .
Write-Host "Built: jayd-speedtest-web:armv7"
Write-Host "Run: docker run -d --name jayd-speedtest -p 8766:8080 jayd-speedtest-web:armv7"
