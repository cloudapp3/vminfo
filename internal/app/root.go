package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/VPSMarket/vminfo/internal/tui"
	"github.com/VPSMarket/vminfo/pkg/vminfo"
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

	if len(args) == 0 {
		return runInfo(ctx, stdout)
	}

	cmd := strings.ToLower(strings.TrimSpace(args[0]))
	if isHelpAlias(cmd) {
		_, _ = io.WriteString(stdout, helpText())
		return nil
	}

	switch cmd {
	case "version", "--version":
		return runVersion(stdout, stderr, args[1:])
	case "info":
		return runInfo(ctx, stdout)
	case "summary":
		return runSummary(ctx, stdout, stderr, args[1:])
	case "watch":
		return runWatch(ctx, stdout, stderr, args[1:])
	case "ps":
		return runPS(ctx, stdout, stderr, args[1:])
	case "kill":
		return runKill(ctx, stdout, args[1:])
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n\n", cmd)
		_, _ = io.WriteString(stderr, helpText())
		return fmt.Errorf("%w: unknown command %q", ErrUsage, cmd)
	}
}

func runInfo(ctx context.Context, w io.Writer) error {
	return tui.Run(ctx, w)
}

func runSummary(ctx context.Context, stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("summary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var interval time.Duration
	fs.BoolVar(&asJSON, "json", false, "write summary as JSON")
	fs.DurationVar(&interval, "interval", vminfo.DefaultSampleInterval, "sampling interval")
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
		return err
	}
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(vminfo.Snapshot{Static: staticInfo, Stats: stats})
	}
	return writeSummary(stdout, staticInfo, stats)
}

func runWatch(ctx context.Context, stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var interval time.Duration
	var count int
	fs.BoolVar(&asJSON, "json", false, "write newline-delimited JSON snapshots")
	fs.DurationVar(&interval, "interval", vminfo.DefaultSampleInterval, "sampling interval per snapshot")
	fs.IntVar(&count, "count", 0, "number of snapshots to emit (0 means infinite)")
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
			return err
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
		if err := writeWatchSnapshot(stdout, collectedAt, staticInfo, stats); err != nil {
			return err
		}
	}
	return nil
}

func runPS(ctx context.Context, stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("ps", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	var sortKey string
	fs.BoolVar(&asJSON, "json", false, "write process list as JSON")
	fs.StringVar(&sortKey, "sort", "cpu", "sort key: cpu|mem|pid|name")
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
		return err
	}
	sortProcesses(items, sortKey)
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	}
	return writeProcesses(stdout, items)
}

func runKill(ctx context.Context, stdout io.Writer, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: kill requires exactly one pid", ErrUsage)
	}
	pidValue, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil {
		return fmt.Errorf("%w: invalid pid %q", ErrUsage, args[0])
	}
	if err := vminfo.TerminateProcess(ctx, int32(pidValue)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "sent SIGTERM to PID %d\n", pidValue)
	return err
}

func runVersion(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var asJSON bool
	fs.BoolVar(&asJSON, "json", false, "write version metadata as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("%w: version does not accept positional args", ErrUsage)
	}

	meta := vminfo.Metadata()
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(meta)
	}

	lines := []string{
		fmt.Sprintf("%s %s", meta.Name, meta.Version),
	}
	if meta.Commit != "" {
		lines = append(lines, fmt.Sprintf("commit: %s", meta.Commit))
	}
	if meta.BuildTime != "" {
		lines = append(lines, fmt.Sprintf("built:  %s", meta.BuildTime))
	}
	if meta.Channel != "" {
		lines = append(lines, fmt.Sprintf("channel: %s", meta.Channel))
	}
	if meta.SchemaVersion != "" {
		lines = append(lines, fmt.Sprintf("schema: %s", meta.SchemaVersion))
	}
	_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return err
}

func writeSummary(w io.Writer, staticInfo vminfo.StaticInfo, stats vminfo.RuntimeStats) error {
	lines := []string{
		"Host Summary",
		"============",
		fmt.Sprintf("Host     : %s", firstNonEmpty(staticInfo.Hostname, "-")),
		fmt.Sprintf("OS       : %s", strings.TrimSpace(strings.Join([]string{firstNonEmpty(staticInfo.Platform, staticInfo.OS, "-"), strings.TrimSpace(staticInfo.OSVersion)}, " "))),
		fmt.Sprintf("Kernel   : %s", firstNonEmpty(staticInfo.Kernel, "-")),
		fmt.Sprintf("Arch     : %s", firstNonEmpty(staticInfo.Arch, "-")),
		fmt.Sprintf("CPU      : %s (%d cores)", firstNonEmpty(staticInfo.CPUModel, "-"), staticInfo.CPUCores),
		fmt.Sprintf("Memory   : %s used / %s total", formatBytes(stats.MemUsed), formatBytes(staticInfo.MemTotal)),
		fmt.Sprintf("Swap     : %s used / %s total", formatBytes(stats.SwapUsed), formatBytes(staticInfo.SwapTotal)),
		fmt.Sprintf("Disk     : %s used / %s total", formatBytes(stats.DiskUsed), formatBytes(staticInfo.DiskTotal)),
		fmt.Sprintf("CPU      : %s", formatPercent(stats.CPU)),
		fmt.Sprintf("Load     : %.2f %.2f %.2f", stats.Load1, stats.Load5, stats.Load15),
		fmt.Sprintf("Network  : ↓ %s/s ↑ %s/s", formatBytes(stats.NetInSpeed), formatBytes(stats.NetOutSpeed)),
		fmt.Sprintf("Conn     : tcp %d / udp %d / proc %d", stats.TCPCount, stats.UDPCount, stats.ProcessCount),
		fmt.Sprintf("Uptime   : %s", formatUptime(stats.Uptime)),
	}
	_, err := fmt.Fprintln(w, strings.Join(lines, "\n"))
	return err
}

func writeWatchSnapshot(w io.Writer, collectedAt time.Time, staticInfo vminfo.StaticInfo, stats vminfo.RuntimeStats) error {
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

func writeProcesses(w io.Writer, items []vminfo.ProcessInfo) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "PID\tPPID\tCPU%\tMEM%\tRSS\tUSER\tSTATE\tNAME"); err != nil {
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
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		switch sortKey {
		case "mem":
			if left.MemoryPercent != right.MemoryPercent {
				return left.MemoryPercent > right.MemoryPercent
			}
		case "pid":
			if left.PID != right.PID {
				return left.PID < right.PID
			}
		case "name":
			leftName := strings.ToLower(strings.TrimSpace(left.Name))
			rightName := strings.ToLower(strings.TrimSpace(right.Name))
			if leftName != rightName {
				return leftName < rightName
			}
		default:
			if left.CPUPercent != right.CPUPercent {
				return left.CPUPercent > right.CPUPercent
			}
		}
		return left.PID < right.PID
	})
}

func helpText() string {
	return strings.Join([]string{
		"Usage:",
		"  vminfo                 start TUI",
		"  vminfo info            start TUI (alias)",
		"  vminfo version         show app version",
		"  vminfo version --json  show app metadata as JSON",
		"  vminfo summary         collect one snapshot",
		"  vminfo summary --json  collect one snapshot as JSON",
		"  vminfo watch           stream runtime snapshots",
		"  vminfo watch --json    stream snapshots as JSON lines",
		"  vminfo ps              list local processes",
		"  vminfo ps --json       list local processes as JSON",
		"  vminfo kill <pid>      send SIGTERM on Linux",
		"  vminfo --version       show app version",
		"  vminfo --help          show help",
		"",
		"Status:",
		"  TUI, summary, watch, ps, kill, and version are implemented.",
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
