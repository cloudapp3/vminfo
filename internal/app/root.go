package app

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/collector"
	"github.com/cloudapp3/vminfo/internal/i18n"
	"github.com/cloudapp3/vminfo/internal/tui"
	"github.com/cloudapp3/vminfo/internal/updater"
	"github.com/cloudapp3/vminfo/internal/web"
)

var ErrUsage = errors.New("usage")

type watchSnapshot struct {
	CollectedAt time.Time           `json:"collected_at"`
	Static      vminfo.StaticInfo   `json:"static"`
	Stats       vminfo.RuntimeStats `json:"stats"`
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	stdout = defaultWriter(stdout)
	stderr = defaultWriter(stderr)

	// Pre-scan global flags (--lang, --web, --port, --bind, --token, --tui, --interval, --silent, --no-update-check)
	langFlag := ""
	webMode := false
	webPort := 20021
	webBind := "127.0.0.1"
	webTokenFlag := ""
	tuiMode := false
	silent := false
	noUpdateCheck := os.Getenv("VMINFO_NO_UPDATE_CHECK") != ""
	webInterval := 3 * time.Second
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--lang="):
			langFlag = strings.TrimPrefix(args[i], "--lang=")
		case args[i] == "--lang" && i+1 < len(args):
			langFlag = args[i+1]
			i++
		case args[i] == "--web":
			webMode = true
		case strings.HasPrefix(args[i], "--port="):
			if p, err := strconv.Atoi(strings.TrimPrefix(args[i], "--port=")); err == nil && p > 0 && p < 65536 {
				webPort = p
			}
		case args[i] == "--port" && i+1 < len(args):
			if p, err := strconv.Atoi(args[i+1]); err == nil && p > 0 && p < 65536 {
				webPort = p
			}
			i++
		case strings.HasPrefix(args[i], "--bind="):
			webBind = strings.TrimPrefix(args[i], "--bind=")
		case args[i] == "--bind" && i+1 < len(args):
			webBind = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--token="):
			webTokenFlag = strings.TrimPrefix(args[i], "--token=")
		case args[i] == "--token":
			// --token without a value means auto-generate
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				webTokenFlag = args[i+1]
				i++
			} else {
				webTokenFlag = "" // bare --token, will auto-generate via resolveWebToken
			}
		case args[i] == "--tui":
			tuiMode = true
		case args[i] == "--silent", args[i] == "-s":
			silent = true
		case webMode && strings.HasPrefix(args[i], "--interval="):
			if d, err := time.ParseDuration(strings.TrimPrefix(args[i], "--interval=")); err == nil {
				webInterval = d
			}
		case webMode && args[i] == "--interval" && i+1 < len(args):
			if d, err := time.ParseDuration(args[i+1]); err == nil {
				webInterval = d
			}
			i++
		case args[i] == "--no-update-check":
			noUpdateCheck = true
		default:
			filtered = append(filtered, args[i])
		}
	}
	args = filtered

	// Detect locale: --lang > VMINFO_LANG > LANG/LC_ALL > default "en"
	locale := i18n.Detect()
	if langFlag != "" {
		locale = strings.ToLower(strings.TrimSpace(langFlag))
	}
	tr := i18n.New(locale)

	// Background update check (non-blocking)
	if !noUpdateCheck && !silent && vminfo.Version != "dev" {
		go func() {
			cfg := updater.Config{
				Repo:        "cloudapp3/vminfo",
				CurrentVer:  vminfo.Version,
				GitHubToken: updateTokenFromEnv(),
			}
			u := updater.New(cfg)
			result, err := u.CheckForUpdate(context.Background())
			if err != nil {
				return
			}
			if result != nil && result.UpdateAvailable {
				msg := fmt.Sprintf(tr.T("A new version of vminfo is available: %s (current: %s). Run 'vminfo update' to upgrade.")+"\n",
					formatReleaseTag(result.LatestVersion), formatReleaseTag(result.CurrentVersion))
				_, _ = fmt.Fprint(stderr, msg)
			}
		}()
	}

	// Handle web mode
	if webMode {
		addr := fmt.Sprintf("%s:%d", webBind, webPort)
		webToken, webTokenGenerated, err := resolveWebToken(webTokenFlag, webTokenFlag == "")
		if err != nil {
			return err
		}
		return runWeb(ctx, stdout, stderr, tr, addr, webInterval, tuiMode, silent, webToken, webTokenGenerated)
	}

	if len(args) == 0 {
		return runInfo(ctx, stdout, tr)
	}

	cmd := strings.ToLower(strings.TrimSpace(args[0]))
	if isHelpAlias(cmd) {
		_, _ = io.WriteString(stdout, helpText(tr))
		return nil
	}

	switch cmd {
	case "version", "--version":
		meta := vminfo.Metadata()
		lines := []string{fmt.Sprintf("%s %s", meta.Name, meta.Version)}
		if meta.Commit != "" {
			lines = append(lines, fmt.Sprintf(tr.T("commit:")+" %s", meta.Commit))
		}
		if meta.BuildTime != "" {
			lines = append(lines, fmt.Sprintf(tr.T("built:")+"  %s", meta.BuildTime))
		}
		if meta.Channel != "" {
			lines = append(lines, fmt.Sprintf(tr.T("channel:")+" %s", meta.Channel))
		}
		if meta.SchemaVersion != "" {
			lines = append(lines, fmt.Sprintf(tr.T("schema:")+" %s", meta.SchemaVersion))
		}
		_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		return err
	case "info":
		return runInfo(ctx, stdout, tr)
	case "summary":
		return runSummary(ctx, stdout, stderr, args[1:], tr)
	case "watch":
		return runWatch(ctx, stdout, stderr, args[1:], tr)
	case "ps":
		return runPS(ctx, stdout, stderr, args[1:], tr)
	case "kill":
		return runKill(ctx, stdout, args[1:], tr)
	case "update":
		return runUpdate(ctx, stdout, stderr, args[1:], tr)
	default:
		_, _ = fmt.Fprintf(stderr, tr.T("unknown command: %s")+"\n\n", cmd)
		_, _ = io.WriteString(stderr, helpText(tr))
		return fmt.Errorf("%w: unknown command %q", ErrUsage, cmd)
	}
}

func runWeb(ctx context.Context, stdout, stderr io.Writer, tr *i18n.Translator, addr string, interval time.Duration, withTUI bool, silent bool, authToken string, tokenGenerated bool) error {
	col := collector.New(interval)
	go col.Start(ctx)
	defer col.Stop()

	// Wait briefly for the first snapshot so startup output can show useful
	// interface addresses instead of only the wildcard bind address.
	snap := waitForCollectorSnapshot(ctx, col, 2*time.Second)

	// Start web server
	srv := web.NewServer(addr, col, web.Options{AuthToken: authToken})
	go func() {
		if err := srv.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			fmt.Fprintf(stderr, "web server error: %v\n", err)
		}
	}()

	if !silent || tokenGenerated {
		for _, line := range webDashboardListenLines(addr, snap, authToken) {
			fmt.Fprintln(stdout, line)
		}
	}

	if withTUI {
		if !silent {
			fmt.Fprintf(stdout, "Starting TUI alongside web dashboard...\n")
		}
		return tui.Run(ctx, stdout, tr)
	}

	// Web-only mode: block until signal
	if !silent {
		fmt.Fprintf(stdout, "Press Ctrl+C to stop\n")
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	if !silent {
		fmt.Fprintf(stdout, "\nShutting down...\n")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func waitForCollectorSnapshot(ctx context.Context, col *collector.Collector, timeout time.Duration) *collector.Snapshot {
	if snap := col.Latest(); snap != nil {
		return snap
	}
	if timeout <= 0 {
		return nil
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return col.Latest()
		case <-time.After(50 * time.Millisecond):
			if snap := col.Latest(); snap != nil {
				return snap
			}
		}
	}
	return col.Latest()
}

func webDashboardListenLines(addr string, snap *collector.Snapshot, authToken string) []string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return []string{fmt.Sprintf("Web dashboard: %s", webDashboardURL(addr, authToken))}
	}
	if !isWildcardBind(host) {
		return []string{fmt.Sprintf("Web dashboard: %s", webDashboardURL(net.JoinHostPort(host, port), authToken))}
	}

	lines := []string{"Web dashboard:"}
	lines = append(lines, fmt.Sprintf("  %-6s %s", "Local", webDashboardURL(net.JoinHostPort("127.0.0.1", port), authToken)))

	if snap == nil {
		return lines
	}

	publicIP := bestInterfaceIPv4(snap.Network.Interfaces, true)
	if publicIP != "" {
		lines = append(lines, fmt.Sprintf("  %-6s %s", "Public", webDashboardURL(net.JoinHostPort(publicIP, port), authToken)))
	}

	lanIP := bestInterfaceIPv4(snap.Network.Interfaces, false)
	if lanIP != "" && lanIP != publicIP {
		lines = append(lines, fmt.Sprintf("  %-6s %s", "LAN", webDashboardURL(net.JoinHostPort(lanIP, port), authToken)))
	}

	return lines
}

func webDashboardURL(hostport, authToken string) string {
	if strings.TrimSpace(authToken) == "" {
		return "http://" + hostport
	}
	u := &url.URL{
		Scheme: "http",
		Host:   hostport,
		Path:   "/",
	}
	query := u.Query()
	query.Set("token", authToken)
	u.RawQuery = query.Encode()
	return u.String()
}

func isWildcardBind(host string) bool {
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::", "[::]":
		return true
	default:
		return false
	}
}

func resolveWebToken(raw string, autoGenerate bool) (token string, generated bool, err error) {
	value := strings.TrimSpace(raw)
	if value != "" {
		return value, false, nil
	}
	if autoGenerate {
		token, err := generateWebToken()
		if err != nil {
			return "", false, fmt.Errorf("generate web token: %w", err)
		}
		return token, true, nil
	}
	return "", false, nil
}

func generateWebToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func bestInterfaceIPv4(ifaces []collector.NetInterface, wantPublic bool) string {
	if len(ifaces) == 0 {
		return ""
	}

	type candidate struct {
		name string
		ip   string
		rx   uint64
		tx   uint64
	}

	items := make([]candidate, 0, len(ifaces))
	for _, iface := range ifaces {
		ip := strings.TrimSpace(iface.IPv4)
		if ip == "" || isLoopbackIP(ip) {
			continue
		}
		isPublic := !isPrivateIP(ip)
		if wantPublic != isPublic {
			continue
		}
		if !wantPublic && isVirtualIface(iface.Name) {
			continue
		}
		items = append(items, candidate{
			name: iface.Name,
			ip:   ip,
			rx:   iface.DownloadSec,
			tx:   iface.UploadSec,
		})
	}

	if len(items) == 0 {
		return ""
	}

	slices.SortFunc(items, func(a, b candidate) int {
		aActive := a.rx > 0 || a.tx > 0
		bActive := b.rx > 0 || b.tx > 0
		if aActive != bActive {
			if aActive {
				return -1
			}
			return 1
		}
		aPri := ifaceDisplayPriority(a.name)
		bPri := ifaceDisplayPriority(b.name)
		if aPri != bPri {
			return cmp.Compare(aPri, bPri)
		}
		return cmp.Compare(a.name, b.name)
	})

	return items[0].ip
}

func ifaceDisplayPriority(name string) int {
	switch {
	case strings.HasPrefix(name, "eth"), strings.HasPrefix(name, "en"):
		return 0
	case strings.HasPrefix(name, "wl"):
		return 1
	case strings.HasPrefix(name, "bond"):
		return 2
	case strings.HasPrefix(name, "br"):
		return 4
	case strings.HasPrefix(name, "docker"):
		return 5
	case strings.HasPrefix(name, "veth"), strings.HasPrefix(name, "tun"), strings.HasPrefix(name, "tap"):
		return 6
	case strings.HasPrefix(name, "lo"):
		return 7
	default:
		return 3
	}
}

func isVirtualIface(name string) bool {
	return ifaceDisplayPriority(name) >= 4
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	privateNets := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	for _, cidr := range privateNets {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func isLoopbackIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

func runInfo(ctx context.Context, w io.Writer, tr *i18n.Translator) error {
	return tui.Run(ctx, w, tr)
}

func runSummary(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	fs := flag.NewFlagSet("summary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var interval time.Duration
	fs.BoolVar(&asJSON, "json", false, tr.T("write summary as JSON"))
	fs.DurationVar(&interval, "interval", vminfo.DefaultSampleInterval, tr.T("sampling interval"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("%w: summary does not accept positional args", ErrUsage)
	}

	staticInfo, stats, err := vminfo.CollectAll(ctx, vminfo.Options{SampleInterval: interval})
	if err != nil {
		return fmt.Errorf("failed to collect host info: %w", err)
	}
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(vminfo.Snapshot{Static: staticInfo, Stats: stats})
	}
	return writeSummary(stdout, staticInfo, stats, tr)
}

func runWatch(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var interval time.Duration
	var count int
	fs.BoolVar(&asJSON, "json", false, tr.T("write newline-delimited JSON snapshots"))
	fs.DurationVar(&interval, "interval", vminfo.DefaultSampleInterval, tr.T("sampling interval per snapshot"))
	fs.IntVar(&count, "count", 0, tr.T("number of snapshots to emit (0 means infinite)"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("%w: watch does not accept positional args", ErrUsage)
	}
	if count < 0 {
		return fmt.Errorf("%w: watch count must be >= 0", ErrUsage)
	}

	encoder := json.NewEncoder(stdout)
	for emitted := 0; count == 0 || emitted < count; emitted++ {
		staticInfo, stats, err := vminfo.CollectAll(ctx, vminfo.Options{SampleInterval: interval})
		if err != nil {
			return fmt.Errorf("failed to collect host info: %w", err)
		}
		collectedAt := time.Now()
		if asJSON {
			if err := encoder.Encode(watchSnapshot{
				CollectedAt: collectedAt,
				Static:      staticInfo,
				Stats:       stats,
			}); err != nil {
				return err
			}
			continue
		}
		if emitted > 0 {
			if _, err := fmt.Fprintln(stdout); err != nil {
				return err
			}
		}
		if err := writeWatchSnapshot(stdout, collectedAt, staticInfo, stats, tr); err != nil {
			return err
		}
	}
	return nil
}

func runPS(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	fs := flag.NewFlagSet("ps", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var sortKey string
	fs.BoolVar(&asJSON, "json", false, tr.T("write process list as JSON"))
	fs.StringVar(&sortKey, "sort", "cpu", tr.T("sort key: cpu|mem|pid|name"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("%w: ps does not accept positional args", ErrUsage)
	}

	items, err := vminfo.ListProcesses(ctx)
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}
	sortProcesses(items, sortKey)
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	}
	return writeProcesses(stdout, items, tr)
}

func runKill(ctx context.Context, stdout io.Writer, args []string, tr *i18n.Translator) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: kill requires exactly one pid", ErrUsage)
	}
	pidValue, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return fmt.Errorf("%w: invalid pid %q", ErrUsage, args[0])
	}
	if err := vminfo.TerminateProcess(ctx, int32(pidValue)); err != nil {
		return fmt.Errorf("failed to terminate PID %d: %w", pidValue, err)
	}
	_, err = fmt.Fprintf(stdout, tr.T("sent SIGTERM to PID %d")+"\n", pidValue)
	return err
}

func writeSummary(w io.Writer, staticInfo vminfo.StaticInfo, stats vminfo.RuntimeStats, tr *i18n.Translator) error {
	lines := []string{
		tr.T("Host Summary"),
		"============",
		fmt.Sprintf(tr.T("Host     :")+" %s", firstNonEmpty(staticInfo.Hostname, "-")),
		fmt.Sprintf(tr.T("OS       :")+" %s", strings.TrimSpace(strings.Join([]string{firstNonEmpty(staticInfo.Platform, staticInfo.OS, "-"), strings.TrimSpace(staticInfo.OSVersion)}, " "))),
		fmt.Sprintf(tr.T("Kernel   :")+" %s", firstNonEmpty(staticInfo.Kernel, "-")),
		fmt.Sprintf(tr.T("Arch     :")+" %s", firstNonEmpty(staticInfo.Arch, "-")),
		fmt.Sprintf(tr.T("CPU      :")+" %s ("+tr.T("%d cores")+")", firstNonEmpty(staticInfo.CPUModel, "-"), staticInfo.CPUCores),
		fmt.Sprintf(tr.T("Memory   :")+" %s"+tr.T(" used / ")+"%s"+tr.T(" total"), formatBytes(stats.MemUsed), formatBytes(staticInfo.MemTotal)),
		fmt.Sprintf(tr.T("Swap     :")+" %s"+tr.T(" used / ")+"%s"+tr.T(" total"), formatBytes(stats.SwapUsed), formatBytes(staticInfo.SwapTotal)),
		fmt.Sprintf(tr.T("Disk     :")+" %s"+tr.T(" used / ")+"%s"+tr.T(" total"), formatBytes(stats.DiskUsed), formatBytes(staticInfo.DiskTotal)),
		fmt.Sprintf(tr.T("CPU      :")+" %s", formatPercent(stats.CPU)),
		fmt.Sprintf(tr.T("Load     :")+" %.2f %.2f %.2f", stats.Load1, stats.Load5, stats.Load15),
		fmt.Sprintf(tr.T("Network  :")+" ↓ %s/s ↑ %s/s", formatBytes(stats.NetInSpeed), formatBytes(stats.NetOutSpeed)),
		fmt.Sprintf(tr.T("Conn     :")+" tcp %d / udp %d / proc %d", stats.TCPCount, stats.UDPCount, stats.ProcessCount),
		fmt.Sprintf(tr.T("Uptime   :")+" %s", formatUptime(stats.Uptime)),
	}
	_, err := fmt.Fprintln(w, strings.Join(lines, "\n"))
	return err
}

func writeWatchSnapshot(w io.Writer, collectedAt time.Time, staticInfo vminfo.StaticInfo, stats vminfo.RuntimeStats, tr *i18n.Translator) error {
	osText := strings.TrimSpace(strings.Join([]string{
		firstNonEmpty(staticInfo.Platform, staticInfo.OS, "-"),
		strings.TrimSpace(staticInfo.OSVersion),
	}, " "))

	lines := []string{
		fmt.Sprintf("[%s] host=%s os=%s", collectedAt.Format(time.RFC3339), firstNonEmpty(staticInfo.Hostname, "-"), osText),
		fmt.Sprintf(
			"cpu=%s mem=%s/%s swap=%s/%s disk=%s/%s",
			formatPercent(stats.CPU),
			formatBytes(stats.MemUsed),
			formatBytes(staticInfo.MemTotal),
			formatBytes(stats.SwapUsed),
			formatBytes(staticInfo.SwapTotal),
			formatBytes(stats.DiskUsed),
			formatBytes(staticInfo.DiskTotal),
		),
		fmt.Sprintf(
			"load=%.2f %.2f %.2f net=↓ %s/s ↑ %s/s conn=tcp %d udp %d proc %d uptime=%s",
			stats.Load1,
			stats.Load5,
			stats.Load15,
			formatBytes(stats.NetInSpeed),
			formatBytes(stats.NetOutSpeed),
			stats.TCPCount,
			stats.UDPCount,
			stats.ProcessCount,
			formatUptime(stats.Uptime),
		),
	}
	_, err := fmt.Fprintln(w, strings.Join(lines, "\n"))
	return err
}

func writeProcesses(w io.Writer, items []vminfo.ProcessInfo, tr *i18n.Translator) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, tr.T("PID")+"\t"+tr.T("PPID")+"\t"+tr.T("CPU%")+"\t"+tr.T("MEM%")+"\t"+tr.T("RSS")+"\t"+tr.T("USER")+"\t"+tr.T("STATE")+"\t"+tr.T("NAME")); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(
			tw,
			"%d\t%d\t%.1f\t%.1f\t%s\t%s\t%s\t%s\n",
			item.PID,
			item.PPID,
			item.CPUPercent,
			item.MemoryPercent,
			formatBytes(item.RSSBytes),
			firstNonEmpty(item.User, "-"),
			firstNonEmpty(item.State, "-"),
			firstNonEmpty(item.Name, "-"),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func sortProcesses(items []vminfo.ProcessInfo, sortKey string) {
	sortKey = strings.ToLower(strings.TrimSpace(sortKey))
	slices.SortFunc(items, func(a, b vminfo.ProcessInfo) int {
		switch sortKey {
		case "mem":
			if a.MemoryPercent != b.MemoryPercent {
				return cmp.Compare(b.MemoryPercent, a.MemoryPercent)
			}
		case "pid":
			if a.PID != b.PID {
				return cmp.Compare(a.PID, b.PID)
			}
		case "name":
			aName := strings.ToLower(strings.TrimSpace(a.Name))
			bName := strings.ToLower(strings.TrimSpace(b.Name))
			if aName != bName {
				return cmp.Compare(aName, bName)
			}
		default:
			if a.CPUPercent != b.CPUPercent {
				return cmp.Compare(b.CPUPercent, a.CPUPercent)
			}
		}
		return cmp.Compare(a.PID, b.PID)
	})
}

func helpText(tr *i18n.Translator) string {
	return strings.Join([]string{
		tr.T("Usage:"),
		"  vminfo                 " + tr.T("start TUI"),
		"  vminfo info            " + tr.T("start TUI (alias)"),
		"  vminfo --web           " + tr.T("start web dashboard"),
		"  vminfo --web --tui     " + tr.T("start web + TUI"),
		"  vminfo --web --port N  " + tr.T("web dashboard on port N (default 20021)"),
		"  vminfo version         " + tr.T("show app version"),
		"  vminfo summary         " + tr.T("collect one snapshot"),
		"  vminfo summary --json  " + tr.T("collect one snapshot as JSON"),
		"  vminfo watch           " + tr.T("stream runtime snapshots"),
		"  vminfo watch --json    " + tr.T("stream snapshots as JSON lines"),
		"  vminfo ps              " + tr.T("list local processes"),
		"  vminfo ps --json       " + tr.T("list local processes as JSON"),
		"  vminfo kill <pid>      " + tr.T("send SIGTERM on Linux"),
		"  vminfo update          " + tr.T("check for and install updates"),
		"  vminfo update --check  " + tr.T("check for updates without installing"),
		"  vminfo --version       " + tr.T("show app version"),
		"  vminfo --help          " + tr.T("show help"),
		"",
		tr.T("Global options:"),
		"  --lang <code>          " + tr.T("force language: en|zh|de|es|fr|ja|ko|pt|ru"),
		"  --web                  " + tr.T("enable web dashboard"),
		"  --port <N>             " + tr.T("web dashboard port (default 20021)"),
		"  --bind <addr>          " + tr.T("bind address (default 127.0.0.1, use 0.0.0.0 for all)"),
		"  --token [value]        " + tr.T("protect --web with a token; bare --token generates one"),
		"  --tui                  " + tr.T("start TUI alongside --web"),
		"  --silent, -s           " + tr.T("suppress informational output"),
		"  --interval <duration>  " + tr.T("refresh interval (default 3s)"),
		"  --no-update-check      " + tr.T("skip background update check"),
		"",
		tr.T("Status:"),
		"  " + tr.T("TUI, web, summary, watch, ps, kill, update, and version are implemented."),
	}, "\n") + "\n"
}

func isHelpAlias(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func defaultWriter(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func formatPercent(value float64) string {
	if value < 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", value)
}

func formatBytes(bytes uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	value := float64(bytes)
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}
	if unitIndex == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unitIndex])
	}
	return fmt.Sprintf("%.1f %s", value, units[unitIndex])
}

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	if d <= 0 {
		return "0s"
	}
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
