package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/cloudapp3/vminfo"
	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestProcessViewsRemoveTerminalControlPayloads(t *testing.T) {
	model := newModel(context.Background(), vminfo.StaticInfo{}, i18n.New("en"))
	model.width = 120
	model.height = 40
	model.ready = true
	model.showKernel = true
	model.viewport = viewport.New(116, 24)
	model.processes = []vminfo.ProcessInfo{{
		PID:     42,
		Name:    "safe\x1b]0;malicious-title\aname",
		Command: "before\x1b]8;;https://evil.invalid\aafter",
		User:    "root\x1b]0;forged-user\a",
	}}
	model.refreshProcessListState()

	for _, treeView := range []bool{false, true} {
		model.treeView = treeView
		output := model.renderProcesses()
		if strings.Contains(output, "malicious-title") || strings.Contains(output, "evil.invalid") || strings.Contains(output, "forged-user") {
			t.Fatalf("process view exposed terminal control payload (tree=%v): %q", treeView, output)
		}
	}
}

func TestTerminalTextFallsBackAfterSanitizing(t *testing.T) {
	if got := terminalText("\x1b]0;malicious-title\a", "fallback"); got != "fallback" {
		t.Fatalf("terminalText() = %q, want fallback", got)
	}
}

func TestSystemViewsRemoveTerminalControlPayloads(t *testing.T) {
	staticInfo := vminfo.StaticInfo{
		Hostname:  "safe-host\x1b]0;hostname-payload\a",
		Platform:  "linux\x1b]0;platform-payload\a",
		OSVersion: "12\x1b]0;version-payload\a",
		Kernel:    "6.1\x1b]0;kernel-payload\a",
		Arch:      "amd64\x1b]0;arch-payload\a",
		CPUModel:  "example-cpu\x1b]0;cpu-payload\a",
		CPUCores:  4,
	}
	model := newModel(context.Background(), staticInfo, i18n.New("en"))
	model.width = 120

	outputs := map[string]string{
		"header":  model.renderMain(),
		"compact": model.renderSystemOneLine(120),
		"panel":   model.renderSystemContent(),
	}
	payloads := []string{
		"hostname-payload",
		"platform-payload",
		"version-payload",
		"kernel-payload",
		"arch-payload",
		"cpu-payload",
	}
	for name, output := range outputs {
		for _, payload := range payloads {
			if strings.Contains(output, payload) {
				t.Fatalf("%s exposed terminal control payload %q: %q", name, payload, output)
			}
		}
	}

	for _, want := range []string{"safe-host", "linux 12", "6.1", "amd64", "example-cpu"} {
		if !strings.Contains(outputs["panel"], want) {
			t.Fatalf("system panel %q does not contain sanitized value %q", outputs["panel"], want)
		}
	}
}
