package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "sync"
    "time"
)

type speedtestJSON struct {
    Type      string `json:"type"`
    Timestamp string `json:"timestamp"`
    Ping      struct {
        Jitter  float64 `json:"jitter"`
        Latency float64 `json:"latency"`
        Low     float64 `json:"low"`
        High    float64 `json:"high"`
    } `json:"ping"`
    Download struct {
        Bandwidth int64 `json:"bandwidth"`
        Bytes     int64 `json:"bytes"`
        Elapsed   int64 `json:"elapsed"`
        Latency   struct {
            IQM float64 `json:"iqm"`
        } `json:"latency"`
    } `json:"download"`
    Upload struct {
        Bandwidth int64 `json:"bandwidth"`
        Bytes     int64 `json:"bytes"`
        Elapsed   int64 `json:"elapsed"`
        Latency   struct {
            IQM float64 `json:"iqm"`
        } `json:"latency"`
    } `json:"upload"`
    PacketLoss float64 `json:"packetLoss"`
    ISP        string  `json:"isp"`
    Interface  struct {
        ExternalIP string `json:"externalIp"`
    } `json:"interface"`
    Server struct {
        ID       int    `json:"id"`
        Host     string `json:"host"`
        Port     int    `json:"port"`
        Name     string `json:"name"`
        Location string `json:"location"`
        Country  string `json:"country"`
    } `json:"server"`
    Result struct {
        ID  string `json:"id"`
        URL string `json:"url"`
    } `json:"result"`
}

type response struct {
    OK          bool    `json:"ok"`
    Error       string  `json:"error,omitempty"`
    PingMS      float64 `json:"ping_ms,omitempty"`
    JitterMS    float64 `json:"jitter_ms,omitempty"`
    DownloadMbps float64 `json:"download_mbps,omitempty"`
    UploadMbps   float64 `json:"upload_mbps,omitempty"`
    PacketLoss   float64 `json:"packet_loss,omitempty"`
    ISP          string  `json:"isp,omitempty"`
    ExternalIP   string  `json:"external_ip,omitempty"`
    Server       string  `json:"server,omitempty"`
    ResultURL    string  `json:"result_url,omitempty"`
    Timestamp    string  `json:"timestamp,omitempty"`
}

var runMu sync.Mutex
var running bool

func main() {
    port := getenv("PORT", "8080")

    mux := http.NewServeMux()
    mux.HandleFunc("/", indexHandler)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        _, _ = w.Write([]byte("ok\n"))
    })
    mux.HandleFunc("/api/run", runHandler)

    srv := &http.Server{
        Addr:              ":" + port,
        Handler:           securityHeaders(mux),
        ReadHeaderTimeout: 5 * time.Second,
        IdleTimeout:       60 * time.Second,
    }

    log.Printf("JayD Speedtest Web listening on :%s", port)
    log.Fatal(srv.ListenAndServe())
}

func getenv(key, fallback string) string {
    if v := strings.TrimSpace(os.Getenv(key)); v != "" {
        return v
    }
    return fallback
}

func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Cache-Control", "no-store")
        next.ServeHTTP(w, r)
    })
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(indexHTML))
}

func runHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.Header().Set("Allow", http.MethodPost)
        http.Error(w, "POST required", http.StatusMethodNotAllowed)
        return
    }

    runMu.Lock()
    if running {
        runMu.Unlock()
        writeJSON(w, http.StatusConflict, response{OK: false, Error: "Speedtest is already running"})
        return
    }
    running = true
    runMu.Unlock()
    defer func() {
        runMu.Lock()
        running = false
        runMu.Unlock()
    }()

    timeoutSec := 180
    if s := strings.TrimSpace(os.Getenv("TEST_TIMEOUT_SECONDS")); s != "" {
        if n, err := strconv.Atoi(s); err == nil && n >= 30 && n <= 600 {
            timeoutSec = n
        }
    }

    ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSec)*time.Second)
    defer cancel()

    speedtestPath, err := exec.LookPath("speedtest")
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "speedtest binary not found in container"})
        return
    }

    args := []string{"--accept-license", "--accept-gdpr", "--format=json", "--progress=no"}
    cmd := exec.CommandContext(ctx, speedtestPath, args...)
    out, err := cmd.CombinedOutput()
    if err != nil {
        if errors.Is(ctx.Err(), context.DeadlineExceeded) {
            writeJSON(w, http.StatusGatewayTimeout, response{OK: false, Error: "Speedtest timed out"})
            return
        }
        msg := strings.TrimSpace(string(out))
        if msg == "" {
            msg = err.Error()
        }
        writeJSON(w, http.StatusBadGateway, response{OK: false, Error: msg})
        return
    }

    var s speedtestJSON
    if err := json.Unmarshal(out, &s); err != nil {
        writeJSON(w, http.StatusBadGateway, response{OK: false, Error: fmt.Sprintf("Cannot parse Speedtest JSON: %v", err)})
        return
    }

    server := strings.TrimSpace(strings.Join(nonEmpty([]string{s.Server.Name, s.Server.Location, s.Server.Country}), ", "))
    writeJSON(w, http.StatusOK, response{
        OK:           true,
        PingMS:       s.Ping.Latency,
        JitterMS:     s.Ping.Jitter,
        DownloadMbps: float64(s.Download.Bandwidth) * 8 / 1_000_000,
        UploadMbps:   float64(s.Upload.Bandwidth) * 8 / 1_000_000,
        PacketLoss:   s.PacketLoss,
        ISP:          s.ISP,
        ExternalIP:   s.Interface.ExternalIP,
        Server:       server,
        ResultURL:    s.Result.URL,
        Timestamp:    s.Timestamp,
    })
}

func nonEmpty(in []string) []string {
    out := make([]string, 0, len(in))
    for _, s := range in {
        if strings.TrimSpace(s) != "" {
            out = append(out, strings.TrimSpace(s))
        }
    }
    return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}

const indexHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>JayD Speedtest</title>
<style>
:root{color-scheme:dark;--bg:#101418;--card:#171d23;--muted:#8c9aa8;--text:#f5f7fa;--line:#28323c;--accent:#4ea1ff;--bad:#ff6b6b}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:15px/1.45 system-ui,-apple-system,Segoe UI,Roboto,sans-serif;min-height:100vh;display:grid;place-items:center;padding:18px}.wrap{width:min(720px,100%)}h1{font-size:24px;margin:0 0 14px;text-align:center}.panel{background:var(--card);border:1px solid var(--line);border-radius:16px;padding:18px;box-shadow:0 10px 40px #0004}.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin:16px 0}.metric{background:#11171c;border:1px solid var(--line);border-radius:12px;padding:14px;text-align:center}.label{color:var(--muted);font-size:12px;text-transform:uppercase;letter-spacing:.08em}.value{font-size:28px;font-weight:700;margin-top:5px}.unit{font-size:13px;color:var(--muted);margin-left:3px}button{width:100%;border:0;border-radius:11px;padding:13px 16px;font-weight:700;font-size:16px;background:var(--accent);color:#07111b;cursor:pointer}button:disabled{opacity:.55;cursor:wait}.details{border-top:1px solid var(--line);padding-top:13px;margin-top:15px;display:grid;grid-template-columns:auto 1fr;gap:6px 12px}.details span:nth-child(odd){color:var(--muted)}#status{text-align:center;color:var(--muted);min-height:22px;margin-top:10px}.error{color:var(--bad)!important}.small{font-size:12px;color:var(--muted);text-align:center;margin-top:10px}a{color:var(--accent)}@media(max-width:560px){.grid{grid-template-columns:1fr}.value{font-size:25px}}
</style>
</head>
<body>
<div class="wrap">
<h1>JayD Speedtest</h1>
<div class="panel">
<button id="run">Запустить Speedtest</button>
<div id="status">Готов к тесту</div>
<div class="grid">
<div class="metric"><div class="label">Ping</div><div class="value"><span id="ping">—</span><span class="unit">ms</span></div></div>
<div class="metric"><div class="label">Download</div><div class="value"><span id="down">—</span><span class="unit">Mbps</span></div></div>
<div class="metric"><div class="label">Upload</div><div class="value"><span id="up">—</span><span class="unit">Mbps</span></div></div>
</div>
<div class="details">
<span>Jitter</span><span id="jitter">—</span>
<span>Packet loss</span><span id="loss">—</span>
<span>ISP</span><span id="isp">—</span>
<span>External IP</span><span id="ip">—</span>
<span>Server</span><span id="server">—</span>
<span>Result</span><span id="result">—</span>
</div>
<div class="small">Без истории и фоновых тестов. Замер запускается только кнопкой.</div>
</div>
</div>
<script>
const $=id=>document.getElementById(id);const run=$('run'),status=$('status');
function n(v,d=2){return Number.isFinite(Number(v))?Number(v).toFixed(d):'—'}
run.onclick=async()=>{run.disabled=true;status.className='';status.textContent='Измерение… обычно 20–60 секунд';
try{const r=await fetch('/api/run',{method:'POST'});const x=await r.json();if(!r.ok||!x.ok)throw new Error(x.error||('HTTP '+r.status));
$('ping').textContent=n(x.ping_ms);$('down').textContent=n(x.download_mbps);$('up').textContent=n(x.upload_mbps);$('jitter').textContent=n(x.jitter_ms)+' ms';$('loss').textContent=n(x.packet_loss)+' %';$('isp').textContent=x.isp||'—';$('ip').textContent=x.external_ip||'—';$('server').textContent=x.server||'—';
$('result').innerHTML=x.result_url?'<a href="'+x.result_url.replace(/"/g,'&quot;')+'" target="_blank" rel="noopener">Ookla result</a>':'—';status.textContent='Готово';
}catch(e){status.className='error';status.textContent='Ошибка: '+e.message}finally{run.disabled=false}};
</script>
</body></html>`
