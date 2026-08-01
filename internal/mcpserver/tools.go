package mcpserver

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/collector"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolGetSystemSnapshot = "get_system_snapshot"
	toolListProcesses     = "list_processes"
	toolResolveDNS        = "resolve_dns"
	toolCheckPort         = "check_port"
	toolPingHost          = "ping_host"
	toolLookupIP          = "lookup_ip"
	toolGetVersion        = "get_version"

	defaultProcessLimit = 20
	maxProcessLimit     = 200
	maxProcessFilterLen = 256
	maxTargetLen        = 255
	maxPingCount        = 10
	maxPingTimeout      = 3 * time.Second
	maxPortTimeout      = 10 * time.Second
)

var (
	localAnnotations   = &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(false)}
	networkAnnotations = &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(true)}
)

type emptyInput struct{}

type listProcessesInput struct {
	Filter         string `json:"filter,omitempty" jsonschema:"case-insensitive filter for PID, name, user, or state; also searches command when include_command is true"`
	SortBy         string `json:"sort_by,omitempty" jsonschema:"sort order: cpu, mem, pid, or name; defaults to cpu"`
	Limit          int    `json:"limit,omitempty" jsonschema:"maximum results to return; defaults to 20 and must not exceed 200"`
	IncludeCommand bool   `json:"include_command,omitempty" jsonschema:"include process command lines, which may contain sensitive arguments"`
}

type listProcessesOutput struct {
	Total           int                  `json:"total"`
	Matched         int                  `json:"matched"`
	Returned        int                  `json:"returned"`
	SortBy          string               `json:"sort_by"`
	CommandIncluded bool                 `json:"command_included"`
	Processes       []vminfo.ProcessInfo `json:"processes"`
}

type resolveDNSInput struct {
	Domain string `json:"domain" jsonschema:"domain name to resolve"`
	Server string `json:"server,omitempty" jsonschema:"optional DNS resolver as host or host:port"`
}

type checkPortInput struct {
	Host      string `json:"host" jsonschema:"host name or IP address to connect to"`
	Port      int    `json:"port" jsonschema:"TCP port from 1 to 65535"`
	TimeoutMS int    `json:"timeout_ms,omitempty" jsonschema:"connection timeout in milliseconds; defaults to 2000 and must not exceed 10000"`
}

type pingHostInput struct {
	Host      string `json:"host" jsonschema:"host name or IP address to probe"`
	Mode      string `json:"mode,omitempty" jsonschema:"probe mode: tcp or icmp; defaults to tcp"`
	Count     int    `json:"count,omitempty" jsonschema:"number of probes; defaults to 4 and must not exceed 10"`
	TimeoutMS int    `json:"timeout_ms,omitempty" jsonschema:"per-probe timeout in milliseconds; defaults to 1000 and must not exceed 3000"`
	TCPPort   int    `json:"tcp_port,omitempty" jsonschema:"TCP target port; defaults to 80"`
}

type lookupIPInput struct {
	IP string `json:"ip,omitempty" jsonschema:"optional IPv4 or IPv6 address; empty returns the caller public IP"`
}

func registerTools(server *mcp.Server, backend Backend) {
	tools := &toolHandlers{backend: backend}
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolGetSystemSnapshot,
		Title:       "Get system snapshot",
		Description: "Collect a current read-only snapshot of local system, CPU, memory, disk, network, load, process count, and health data.",
		Annotations: localAnnotations,
	}, tools.getSystemSnapshot)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolListProcesses,
		Title:       "List processes",
		Description: "List, filter, and sort local processes on Linux. Command lines are omitted unless explicitly requested.",
		Annotations: localAnnotations,
	}, tools.listProcesses)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolResolveDNS,
		Title:       "Resolve DNS",
		Description: "Resolve a domain with the system resolver or an explicitly selected DNS server.",
		Annotations: networkAnnotations,
	}, tools.resolveDNS)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolCheckPort,
		Title:       "Check TCP port",
		Description: "Test TCP connectivity and connection latency to a host and port.",
		Annotations: networkAnnotations,
	}, tools.checkPort)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolPingHost,
		Title:       "Ping host",
		Description: "Measure host reachability and latency with TCP or ICMP probes. ICMP may require host privileges.",
		Annotations: networkAnnotations,
	}, tools.pingHost)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolLookupIP,
		Title:       "Look up IP information",
		Description: "Look up public IP, ASN, location, and risk data through ip.bestcheapvps.org. An empty IP inspects the caller public IP.",
		Annotations: networkAnnotations,
	}, tools.lookupIP)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolGetVersion,
		Title:       "Get vminfo version",
		Description: "Return vminfo version, build, repository, and schema metadata.",
		Annotations: localAnnotations,
	}, tools.getVersion)
}

type toolHandlers struct {
	backend Backend
}

func (t *toolHandlers) getSystemSnapshot(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, collector.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	snapshot, err := t.backend.SystemSnapshot(ctx)
	if err != nil {
		return nil, collector.Snapshot{}, fmt.Errorf("collect system snapshot: %w", err)
	}
	normalizeSnapshotCollections(&snapshot)
	return nil, snapshot, nil
}

func (t *toolHandlers) listProcesses(ctx context.Context, _ *mcp.CallToolRequest, input listProcessesInput) (*mcp.CallToolResult, listProcessesOutput, error) {
	filter := strings.TrimSpace(input.Filter)
	if len(filter) > maxProcessFilterLen {
		return nil, listProcessesOutput{}, fmt.Errorf("filter must not exceed %d bytes", maxProcessFilterLen)
	}
	sortBy := strings.ToLower(strings.TrimSpace(input.SortBy))
	if sortBy == "" {
		sortBy = "cpu"
	}
	if sortBy != "cpu" && sortBy != "mem" && sortBy != "pid" && sortBy != "name" {
		return nil, listProcessesOutput{}, fmt.Errorf("sort_by must be one of: cpu, mem, pid, name")
	}
	limit := input.Limit
	if limit == 0 {
		limit = defaultProcessLimit
	}
	if limit < 1 || limit > maxProcessLimit {
		return nil, listProcessesOutput{}, fmt.Errorf("limit must be between 1 and %d", maxProcessLimit)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, err := t.backend.ListProcesses(ctx)
	if err != nil {
		return nil, listProcessesOutput{}, fmt.Errorf("list processes: %w", err)
	}
	total := len(items)
	filtered := make([]vminfo.ProcessInfo, 0, total)
	for _, item := range items {
		if processMatchesFilter(item, filter, input.IncludeCommand) {
			filtered = append(filtered, item)
		}
	}
	sortProcessList(filtered, sortBy)
	matched := len(filtered)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	if !input.IncludeCommand {
		for i := range filtered {
			filtered[i].Command = ""
		}
	}
	return nil, listProcessesOutput{
		Total:           total,
		Matched:         matched,
		Returned:        len(filtered),
		SortBy:          sortBy,
		CommandIncluded: input.IncludeCommand,
		Processes:       filtered,
	}, nil
}

func (t *toolHandlers) resolveDNS(ctx context.Context, _ *mcp.CallToolRequest, input resolveDNSInput) (*mcp.CallToolResult, vminfo.DNSResult, error) {
	domain, err := validateTarget("domain", input.Domain)
	if err != nil {
		return nil, vminfo.DNSResult{}, err
	}
	server := strings.TrimSpace(input.Server)
	if len(server) > maxTargetLen || strings.ContainsAny(server, " \t\r\n") {
		return nil, vminfo.DNSResult{}, fmt.Errorf("server must be a valid host or host:port")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return nil, t.backend.ResolveDNS(ctx, domain, server), nil
}

func (t *toolHandlers) checkPort(ctx context.Context, _ *mcp.CallToolRequest, input checkPortInput) (*mcp.CallToolResult, vminfo.PortResult, error) {
	host, err := validateTarget("host", input.Host)
	if err != nil {
		return nil, vminfo.PortResult{}, err
	}
	if input.Port < 1 || input.Port > 65535 {
		return nil, vminfo.PortResult{}, fmt.Errorf("port must be between 1 and 65535")
	}
	timeout, err := timeoutFromMilliseconds(input.TimeoutMS, 2*time.Second, maxPortTimeout)
	if err != nil {
		return nil, vminfo.PortResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return nil, t.backend.CheckPort(ctx, host, input.Port, timeout), nil
}

func (t *toolHandlers) pingHost(ctx context.Context, _ *mcp.CallToolRequest, input pingHostInput) (*mcp.CallToolResult, vminfo.PingResult, error) {
	host, err := validateTarget("host", input.Host)
	if err != nil {
		return nil, vminfo.PingResult{}, err
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = "tcp"
	}
	if mode != "tcp" && mode != "icmp" {
		return nil, vminfo.PingResult{}, fmt.Errorf("mode must be tcp or icmp")
	}
	count := input.Count
	if count == 0 {
		count = 4
	}
	if count < 1 || count > maxPingCount {
		return nil, vminfo.PingResult{}, fmt.Errorf("count must be between 1 and %d", maxPingCount)
	}
	timeout, err := timeoutFromMilliseconds(input.TimeoutMS, time.Second, maxPingTimeout)
	if err != nil {
		return nil, vminfo.PingResult{}, err
	}
	port := input.TCPPort
	if port == 0 {
		port = 80
	}
	if port < 1 || port > 65535 {
		return nil, vminfo.PingResult{}, fmt.Errorf("tcp_port must be between 1 and 65535")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(count)*timeout+time.Second)
	defer cancel()
	return nil, t.backend.Ping(ctx, host, vminfo.PingOptions{
		Mode: mode, Count: count, Timeout: timeout, Port: port,
	}), nil
}

func (t *toolHandlers) lookupIP(ctx context.Context, _ *mcp.CallToolRequest, input lookupIPInput) (*mcp.CallToolResult, vminfo.IPInfo, error) {
	ip := strings.TrimSpace(input.IP)
	if ip != "" && net.ParseIP(ip) == nil {
		return nil, vminfo.IPInfo{}, fmt.Errorf("ip must be a valid IPv4 or IPv6 address")
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	return nil, t.backend.LookupIP(ctx, ip), nil
}

func (t *toolHandlers) getVersion(context.Context, *mcp.CallToolRequest, emptyInput) (*mcp.CallToolResult, vminfo.AppMetadata, error) {
	return nil, t.backend.Metadata(), nil
}

func validateTarget(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if len(value) > maxTargetLen || strings.ContainsAny(value, " \t\r\n") {
		return "", fmt.Errorf("%s must be a valid host name or IP address", name)
	}
	return value, nil
}

func timeoutFromMilliseconds(value int, fallback, maximum time.Duration) (time.Duration, error) {
	if value == 0 {
		return fallback, nil
	}
	if value < 1 {
		return 0, fmt.Errorf("timeout_ms must be positive")
	}
	timeout := time.Duration(value) * time.Millisecond
	if timeout > maximum {
		return 0, fmt.Errorf("timeout_ms must not exceed %d", maximum/time.Millisecond)
	}
	return timeout, nil
}

func processMatchesFilter(item vminfo.ProcessInfo, filter string, includeCommand bool) bool {
	if filter == "" {
		return true
	}
	query := strings.ToLower(filter)
	fields := []string{
		strconv.FormatInt(int64(item.PID), 10),
		strconv.FormatInt(int64(item.PPID), 10),
		item.Name,
		item.User,
		item.State,
	}
	if includeCommand {
		fields = append(fields, item.Command)
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func sortProcessList(items []vminfo.ProcessInfo, sortBy string) {
	slices.SortFunc(items, func(a, b vminfo.ProcessInfo) int {
		switch sortBy {
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

func normalizeSnapshotCollections(snapshot *collector.Snapshot) {
	if snapshot.CPU.PerCore == nil {
		snapshot.CPU.PerCore = []float64{}
	}
	if snapshot.CPU.History == nil {
		snapshot.CPU.History = []float64{}
	}
	if snapshot.Disk.Filesystems == nil {
		snapshot.Disk.Filesystems = []collector.Filesystem{}
	}
	if snapshot.Disk.IO == nil {
		snapshot.Disk.IO = []collector.DiskIO{}
	}
	if snapshot.Network.Interfaces == nil {
		snapshot.Network.Interfaces = []collector.NetInterface{}
	}
	if snapshot.Processes.List == nil {
		snapshot.Processes.List = []collector.ProcessEntry{}
	}
	if snapshot.Health.Warnings == nil {
		snapshot.Health.Warnings = []collector.HealthWarning{}
	}
}

func boolPtr(value bool) *bool { return &value }
