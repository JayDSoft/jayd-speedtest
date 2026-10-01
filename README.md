# JayD Speedtest Web — ARMv7 / Sonoff iHost

Минимальная веб-оболочка для официального Ookla Speedtest CLI.

- ARMv7 (`linux/arm/v7`) — подходит для Sonoff iHost.
- Один ручной тест по кнопке.
- Ping / Download / Upload + Jitter / Packet Loss / ISP / сервер.
- Нет БД, истории, cron, Node.js, Python или PHP.
- Web-сервер — один статически скомпилированный Go-бинарник.
- Базовый образ: `lferrarotti74/speedtest-ookla:latest`, содержащий официальный Ookla CLI.

## Самый простой build

В архиве уже лежит `speedtest-web-armv7`, поэтому Dockerfile ничего не компилирует и не выполняет ARM-код во время сборки.

```powershell
docker buildx build --platform linux/arm/v7 -t YOUR_DOCKERHUB/jayd-speedtest:latest --push .
```

После этого на iHost/CUBE ищем:

```text
YOUR_DOCKERHUB/jayd-speedtest:latest
```

Параметры Run в CUBE:

```text
Network: bridge
Host Port: 8766
Add-on / Container Port: 8080
Restart: unless-stopped / always
```

Открыть:

```text
http://IP_iHOST:8766
```

Никакие volumes и environment variables не нужны.

## Локальная проверка на ПК

ARMv7-контейнер на x64 Windows обычно требует Docker Desktop/buildx с эмуляцией:

```powershell
docker buildx build --platform linux/arm/v7 -t jayd-speedtest-web:armv7 --load .
docker run --rm -p 8766:8080 jayd-speedtest-web:armv7
```

## Сборка Go-бинарника заново

Для воспроизводимой сборки есть `Dockerfile.build`:

```powershell
docker buildx build --platform linux/arm/v7 -f Dockerfile.build -t YOUR_DOCKERHUB/jayd-speedtest:latest --push .
```

Или локально при установленном Go:

```powershell
$env:CGO_ENABLED="0"
$env:GOOS="linux"
$env:GOARCH="arm"
$env:GOARM="7"
go build -trimpath -ldflags="-s -w" -o speedtest-web-armv7 main.go
```

## Порт

Внутри контейнера по умолчанию `8080`. Снаружи можно использовать любой свободный, например `8766`, `8767`, `9080` и т.д.

## Лицензия Ookla

Speedtest CLI — собственность Ookla и работает по условиям лицензии Ookla. Веб-оболочка запускает CLI с `--accept-license --accept-gdpr` при нажатии кнопки.
