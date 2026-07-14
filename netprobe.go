package vminfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// DNSResult is the outcome of a DNS lookup performed by ResolveDNS.
type DNSResult struct {
	Domain    string   `json:"domain"`
	Addrs     []string `json:"addrs,omitempty"`
	Server    string   `json:"server,omitempty"`
	ElapsedMs float64  `json:"elapsed_ms"`
	Err       string   `json:"error,omitempty"`
}

// PortResult is the outcome of a TCP connectivity probe performed by CheckPort.
type PortResult struct {
	Host      string  `json:"host"`
	Port      int     `json:"port"`
	Open      bool    `json:"open"`
	ElapsedMs float64 `json:"elapsed_ms"`
	Err       string  `json:"error,omitempty"`
}

// ResolveDNS looks up domain's host addresses. If server is empty it uses the
// system default resolver; otherwise it queries the given DNS server
// (accepts "1.1.1.1" or "1.1.1.1:53"; a bare host defaults to port 53).
func ResolveDNS(ctx context.Context, domain, server string) DNSResult {
	start := time.Now()
	res := DNSResult{Domain: domain, Server: server}

	resolver := net.DefaultResolver
	if server != "" {
		target := server
		if _, _, err := net.SplitHostPort(server); err != nil {
			target = net.JoinHostPort(server, "53")
		}
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, network, target)
			},
		}
	}

	addrs, err := resolver.LookupHost(ctx, domain)
	res.ElapsedMs = float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Addrs = addrs
	return res
}

// CheckPort tests TCP connectivity to host:port. It honors both ctx
// cancellation and timeout (fallback 2s when timeout <= 0).
func CheckPort(ctx context.Context, host string, port int, timeout time.Duration) PortResult {
	start := time.Now()
	res := PortResult{Host: host, Port: port}

	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	res.ElapsedMs = float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		res.Err = err.Error()
		return res
	}
	conn.Close()
	res.Open = true
	return res
}

// PingOptions controls a Ping probe sequence.
type PingOptions struct {
	Mode    string        // "tcp" (default) or "icmp"
	Count   int           // number of probes (default 4, maximum 100)
	Timeout time.Duration // per-probe timeout (default 1s, maximum 10s)
	Port    int           // tcp mode target port (default 80, range 1..65535)
}

const (
	defaultPingCount   = 4
	maxPingCount       = 100
	defaultPingTimeout = time.Second
	maxPingTimeout     = 10 * time.Second
	defaultPingPort    = 80
)

// PingResult is the outcome of a Ping probe sequence.
type PingResult struct {
	Host        string    `json:"host"`
	Mode        string    `json:"mode"`
	Port        int       `json:"port,omitempty"`
	Sent        int       `json:"sent"`
	Lost        int       `json:"lost"`
	LossPercent float64   `json:"loss_percent"`
	RTTs        []float64 `json:"rtts_ms,omitempty"`
	MinMs       float64   `json:"min_ms,omitempty"`
	AvgMs       float64   `json:"avg_ms,omitempty"`
	MaxMs       float64   `json:"max_ms,omitempty"`
	Err         string    `json:"error,omitempty"`
}

// Ping probes host Count times. Mode "tcp" (default) does TCP-dial RTTs and is
// cross-platform / unprivileged; Mode "icmp" sends ICMP Echo via golang.org/x/net
// (unprivileged udp4: needs net.ipv4.ping_group_range on Linux, unsupported on Windows).
func Ping(ctx context.Context, host string, opts PingOptions) PingResult {
	normalized, err := normalizePingOptions(opts)
	res := PingResult{Host: host, Mode: normalized.Mode}
	if normalized.Mode == "tcp" {
		res.Port = normalized.Port
	}
	if err != nil {
		res.Err = err.Error()
		return res
	}

	if normalized.Mode == "icmp" {
		rtts, lost, err := pingICMP(ctx, host, normalized.Count, normalized.Timeout)
		if err != nil {
			res.Err = err.Error()
			return res
		}
		fillPingStats(&res, normalized.Count, lost, rtts)
		return res
	}

	rtts, lost, err := pingTCP(
		ctx,
		host,
		normalized.Port,
		normalized.Count,
		normalized.Timeout,
	)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	fillPingStats(&res, normalized.Count, lost, rtts)
	return res
}

func normalizePingOptions(opts PingOptions) (PingOptions, error) {
	opts.Mode = strings.ToLower(strings.TrimSpace(opts.Mode))
	if opts.Mode == "" {
		opts.Mode = "tcp"
	}
	if opts.Mode != "tcp" && opts.Mode != "icmp" {
		return opts, fmt.Errorf("unsupported ping mode %q", opts.Mode)
	}

	if opts.Count < 0 {
		return opts, fmt.Errorf("ping count must not be negative")
	}
	if opts.Count == 0 {
		opts.Count = defaultPingCount
	}
	if opts.Count > maxPingCount {
		return opts, fmt.Errorf("ping count must not exceed %d", maxPingCount)
	}

	if opts.Timeout < 0 {
		return opts, fmt.Errorf("ping timeout must not be negative")
	}
	if opts.Timeout == 0 {
		opts.Timeout = defaultPingTimeout
	}
	if opts.Timeout > maxPingTimeout {
		return opts, fmt.Errorf("ping timeout must not exceed %s", maxPingTimeout)
	}

	if opts.Port < 0 || opts.Port > 65535 {
		return opts, fmt.Errorf("ping port must be between 1 and 65535")
	}
	if opts.Mode == "tcp" && opts.Port == 0 {
		opts.Port = defaultPingPort
	}
	return opts, nil
}

func fillPingStats(res *PingResult, sent, lost int, rtts []float64) {
	res.Sent = sent
	res.Lost = lost
	if sent > 0 {
		res.LossPercent = float64(lost) / float64(sent) * 100
	}
	res.RTTs = rtts
	if len(rtts) == 0 {
		return
	}
	mn, mx := rtts[0], rtts[0]
	sum := 0.0
	for _, r := range rtts {
		mn = min(mn, r)
		mx = max(mx, r)
		sum += r
	}
	res.MinMs = mn
	res.MaxMs = mx
	res.AvgMs = sum / float64(len(rtts))
}

func pingTCP(ctx context.Context, host string, port, count int, timeout time.Duration) ([]float64, int, error) {
	rtts := make([]float64, 0, count)
	lost := 0
	dialer := net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	for range count {
		if err := ctx.Err(); err != nil {
			return rtts, lost, err
		}
		start := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		elapsed := time.Since(start)
		if err != nil {
			lost++
			continue
		}
		conn.Close()
		rtts = append(rtts, float64(elapsed.Microseconds())/1000.0)
	}
	return rtts, lost, nil
}

func pingICMP(ctx context.Context, host string, count int, timeout time.Duration) ([]float64, int, error) {
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, 0, fmt.Errorf("icmp unavailable (try --mode tcp): %w", err)
	}
	defer c.Close()
	stopClose := context.AfterFunc(ctx, func() {
		_ = c.Close()
	})
	defer stopClose()

	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, 0, err
	}
	if len(addresses) == 0 {
		return nil, 0, fmt.Errorf("no IPv4 address found for %q", host)
	}
	dst := &net.IPAddr{IP: addresses[0]}

	id := os.Getpid() & 0xffff
	rtts := make([]float64, 0, count)
	lost := 0
	for i := range count {
		if err := ctx.Err(); err != nil {
			return rtts, lost, err
		}
		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{ID: id, Seq: i, Data: []byte("vminfo-ping")},
		}
		wb, err := msg.Marshal(nil)
		if err != nil {
			lost++
			continue
		}
		start := time.Now()
		if _, err := c.WriteTo(wb, dst); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return rtts, lost, ctxErr
			}
			lost++
			continue
		}
		deadline := nextProbeDeadline(ctx, time.Now(), timeout)
		if err := c.SetReadDeadline(deadline); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return rtts, lost, ctxErr
			}
			lost++
			continue
		}
		rb := make([]byte, 1500)
		n, _, err := c.ReadFrom(rb)
		elapsed := time.Since(start)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return rtts, lost, ctxErr
			}
			lost++
			continue
		}
		rm, err := icmp.ParseMessage(1, rb[:n]) // 1 = ProtocolICMP (IPv4)
		if err != nil {
			lost++
			continue
		}
		if rm.Type != ipv4.ICMPTypeEchoReply {
			lost++
			continue
		}
		rtts = append(rtts, float64(elapsed.Microseconds())/1000.0)
	}
	return rtts, lost, nil
}

func nextProbeDeadline(ctx context.Context, now time.Time, timeout time.Duration) time.Time {
	deadline := now.Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		return contextDeadline
	}
	return deadline
}

// DefaultIPLookupServer is the default IP geo/ASN lookup service.
const DefaultIPLookupServer = "https://ip.bestcheapvps.org"

// IPInfo holds geo/ASN/risk info for an IP, as returned by the lookup service.
type IPInfo struct {
	IP           string  `json:"ip"`
	Country      string  `json:"country,omitempty"`
	CountryCode  string  `json:"country_code,omitempty"`
	Region       string  `json:"region,omitempty"`
	City         string  `json:"city,omitempty"`
	Postal       string  `json:"postal,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	Timezone     string  `json:"timezone,omitempty"`
	ASN          string  `json:"asn,omitempty"`
	Org          string  `json:"org,omitempty"`
	ISP          string  `json:"isp,omitempty"`
	Prefix       string  `json:"prefix,omitempty"`
	IsTor        bool    `json:"is_tor,omitempty"`
	IsProxy      bool    `json:"is_proxy,omitempty"`
	IsVPN        bool    `json:"is_vpn,omitempty"`
	IsDatacenter bool    `json:"is_datacenter,omitempty"`
	ThreatScore  int     `json:"threat_score,omitempty"`
	ElapsedMs    float64 `json:"elapsed_ms,omitempty"`
	Err          string  `json:"error,omitempty"`
}

// LookupIP queries the IP lookup service at server (default
// DefaultIPLookupServer). If ip is empty the service returns the caller's own
// public IP info; otherwise it returns info for ip. This is an explicit,
// user-triggered outbound request (privacy: disclosed in --help and output).
func LookupIP(ctx context.Context, ip, server string) IPInfo {
	if server == "" {
		server = DefaultIPLookupServer
	}
	start := time.Now()
	base := strings.TrimRight(server, "/")
	target := base + "/api/v1/myip"
	if strings.TrimSpace(ip) != "" {
		target = base + "/api/v1/ip/" + url.PathEscape(strings.TrimSpace(ip))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return IPInfo{Err: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return IPInfo{Err: err.Error()}
	}
	defer resp.Body.Close()

	var env struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data *IPInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return IPInfo{Err: err.Error()}
	}
	if env.Code != 200 || env.Data == nil {
		msg := env.Msg
		if msg == "" {
			msg = fmt.Sprintf("lookup returned code %d", env.Code)
		}
		return IPInfo{Err: msg, ElapsedMs: float64(time.Since(start).Microseconds()) / 1000.0}
	}
	info := *env.Data
	info.ElapsedMs = float64(time.Since(start).Microseconds()) / 1000.0
	return info
}
