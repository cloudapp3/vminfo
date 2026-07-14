package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

const (
	maxNetProbeCount   = 100
	maxNetProbeTimeout = 10 * time.Second
)

// runNet dispatches `vminfo net <action>` subcommands.
func runNet(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: net requires an action: dns | port | ping | ip", ErrUsage)
	}
	action := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch action {
	case "dns":
		return runNetDNS(ctx, stdout, stderr, rest, tr)
	case "port":
		return runNetPort(ctx, stdout, stderr, rest, tr)
	case "ping":
		return runNetPing(ctx, stdout, stderr, rest, tr)
	case "ip":
		return runNetIP(ctx, stdout, stderr, rest, tr)
	default:
		return fmt.Errorf("%w: unknown net action %q (want: dns | port | ping | ip)", ErrUsage, action)
	}
}

// reorderNetFlags lets net subcommands accept flags before or after their
// positional target while still using the standard library flag package.
func reorderNetFlags(args []string, specs map[string]bool) ([]string, error) {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	hasTerminator := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			hasTerminator = true
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}

		nameValue := strings.TrimLeft(arg, "-")
		name, _, hasValue := strings.Cut(nameValue, "=")
		requiresValue, known := specs[name]
		flags = append(flags, arg)
		if !known || !requiresValue || hasValue {
			continue
		}
		if i+1 >= len(args) || args[i+1] == "--" {
			return nil, fmt.Errorf("%w: flag --%s requires a value", ErrUsage, name)
		}
		flags = append(flags, args[i+1])
		i++
	}
	if hasTerminator {
		flags = append(flags, "--")
	}
	return append(flags, positionals...), nil
}

func runNetDNS(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	var err error
	args, err = reorderNetFlags(args, map[string]bool{"server": true, "json": false})
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("net dns", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		server string
		asJSON bool
	)
	fs.StringVar(&server, "server", "", tr.T("DNS server (host or host:port); empty = system default"))
	fs.BoolVar(&asJSON, "json", false, tr.T("write result as JSON"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: net dns requires exactly one domain", ErrUsage)
	}

	res := vminfo.ResolveDNS(ctx, fs.Arg(0), server)
	if asJSON {
		return encodeNetJSON(stdout, res)
	}
	return writeDNS(stdout, res, tr)
}

func runNetPort(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	var err error
	args, err = reorderNetFlags(args, map[string]bool{"timeout": true, "json": false})
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("net port", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		timeout time.Duration
		asJSON  bool
	)
	fs.DurationVar(&timeout, "timeout", 2*time.Second, tr.T("dial timeout"))
	fs.BoolVar(&asJSON, "json", false, tr.T("write result as JSON"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("%w: net port requires <host> <port>", ErrUsage)
	}
	port, err := strconv.Atoi(strings.TrimSpace(fs.Arg(1)))
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid port %q", ErrUsage, fs.Arg(1))
	}
	if timeout <= 0 || timeout > maxNetProbeTimeout {
		return fmt.Errorf("%w: timeout must be > 0 and <= %s", ErrUsage, maxNetProbeTimeout)
	}

	res := vminfo.CheckPort(ctx, fs.Arg(0), port, timeout)
	if asJSON {
		return encodeNetJSON(stdout, res)
	}
	return writePort(stdout, res, tr)
}

func encodeNetJSON(stdout io.Writer, v any) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeDNS(w io.Writer, res vminfo.DNSResult, tr *i18n.Translator) error {
	if res.Err != "" {
		_, err := fmt.Fprintf(w, tr.T("dns lookup failed: %s")+"\n", res.Err)
		return err
	}
	server := res.Server
	if server == "" {
		server = tr.T("system")
	}
	addrs := strings.Join(res.Addrs, ", ")
	if addrs == "" {
		addrs = tr.T("(no addresses)")
	}
	_, err := fmt.Fprintf(w, "%s → %s  (%s, %.1fms)\n", res.Domain, addrs, server, res.ElapsedMs)
	return err
}

func writePort(w io.Writer, res vminfo.PortResult, tr *i18n.Translator) error {
	if res.Err != "" {
		_, err := fmt.Fprintf(w, "%s:%d  %s  (%.1fms)\n", res.Host, res.Port, tr.T("error")+": "+res.Err, res.ElapsedMs)
		return err
	}
	state := tr.T("closed")
	if res.Open {
		state = tr.T("open")
	}
	_, err := fmt.Fprintf(w, "%s:%d  %s  (%.1fms)\n", res.Host, res.Port, state, res.ElapsedMs)
	return err
}

func runNetPing(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	var err error
	args, err = reorderNetFlags(args, map[string]bool{
		"mode": true, "count": true, "timeout": true, "tcp-port": true, "json": false,
	})
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("net ping", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		mode    string
		count   int
		timeout time.Duration
		port    int
		asJSON  bool
	)
	fs.StringVar(&mode, "mode", "tcp", tr.T("probe mode: tcp | icmp"))
	fs.IntVar(&count, "count", 4, tr.T("number of probes"))
	fs.DurationVar(&timeout, "timeout", time.Second, tr.T("per-probe timeout"))
	fs.IntVar(&port, "tcp-port", 80, tr.T("tcp target port"))
	fs.BoolVar(&asJSON, "json", false, tr.T("write result as JSON"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: net ping requires exactly one host", ErrUsage)
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "tcp" && mode != "icmp" {
		return fmt.Errorf("%w: invalid ping mode %q (want tcp or icmp)", ErrUsage, mode)
	}
	if count < 1 || count > maxNetProbeCount {
		return fmt.Errorf("%w: ping count must be between 1 and %d", ErrUsage, maxNetProbeCount)
	}
	if timeout <= 0 || timeout > maxNetProbeTimeout {
		return fmt.Errorf("%w: ping timeout must be > 0 and <= %s", ErrUsage, maxNetProbeTimeout)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%w: invalid tcp port %d (want 1-65535)", ErrUsage, port)
	}

	res := vminfo.Ping(ctx, fs.Arg(0), vminfo.PingOptions{Mode: mode, Count: count, Timeout: timeout, Port: port})
	if asJSON {
		return encodeNetJSON(stdout, res)
	}
	return writePing(stdout, res, tr)
}

func writePing(w io.Writer, res vminfo.PingResult, tr *i18n.Translator) error {
	if res.Err != "" {
		_, err := fmt.Fprintf(w, tr.T("ping failed: %s")+"\n", res.Err)
		return err
	}
	target := res.Host
	if res.Mode == "tcp" && res.Port > 0 {
		target = fmt.Sprintf("%s:%d", res.Host, res.Port)
	}
	if len(res.RTTs) == 0 {
		_, err := fmt.Fprintf(w, "%s  %s  sent=%d lost=%d loss=%.0f%%  (%s)\n", target, res.Mode, res.Sent, res.Lost, res.LossPercent, tr.T("no replies"))
		return err
	}
	_, err := fmt.Fprintf(w, "%s  %s  sent=%d lost=%d loss=%.0f%%  rtt %.1f/%.1f/%.1f ms (min/avg/max)\n", target, res.Mode, res.Sent, res.Lost, res.LossPercent, res.MinMs, res.AvgMs, res.MaxMs)
	return err
}

func runNetIP(ctx context.Context, stdout, stderr io.Writer, args []string, tr *i18n.Translator) error {
	var err error
	args, err = reorderNetFlags(args, map[string]bool{"server": true, "json": false})
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("net ip", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		server string
		asJSON bool
	)
	fs.StringVar(&server, "server", vminfo.DefaultIPLookupServer, tr.T("IP lookup service base URL"))
	fs.BoolVar(&asJSON, "json", false, tr.T("write result as JSON"))
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("%w: net ip accepts at most one IP", ErrUsage)
	}
	ip := ""
	if fs.NArg() == 1 {
		ip = fs.Arg(0)
	}

	res := vminfo.LookupIP(ctx, ip, server)
	if asJSON {
		return encodeNetJSON(stdout, res)
	}
	return writeIP(stdout, res, tr)
}

func writeIP(w io.Writer, res vminfo.IPInfo, tr *i18n.Translator) error {
	if res.Err != "" {
		_, err := fmt.Fprintf(w, tr.T("ip lookup failed: %s")+"\n", res.Err)
		return err
	}
	line := res.IP
	if res.Country != "" {
		line += "  " + res.Country
		if res.CountryCode != "" {
			line += " (" + res.CountryCode + ")"
		}
	}
	if res.City != "" {
		line += "  " + res.City
	}
	if res.ASN != "" {
		line += "  " + res.ASN
		if res.Org != "" {
			line += " " + res.Org
		}
	}
	var flags []string
	if res.IsDatacenter {
		flags = append(flags, "datacenter")
	}
	if res.IsVPN {
		flags = append(flags, "vpn")
	}
	if res.IsProxy {
		flags = append(flags, "proxy")
	}
	if res.IsTor {
		flags = append(flags, "tor")
	}
	if len(flags) > 0 {
		line += "  [" + strings.Join(flags, ",") + "]"
	}
	// "via ip.bestcheapvps.org" discloses the third-party lookup (privacy).
	_, err := fmt.Fprintf(w, "%s  (via ip.bestcheapvps.org, %.1fms)\n", line, res.ElapsedMs)
	return err
}
