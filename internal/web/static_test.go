package web

import (
	"strings"
	"testing"
)

func TestProcessTableUsesDOMTextProperties(t *testing.T) {
	data, err := staticFS.ReadFile("static/js/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	source := string(data)

	if strings.Contains(source, "dom.procTbody.innerHTML") {
		t.Fatal("process table still renders untrusted process data through innerHTML")
	}
	for _, required := range []string{
		"document.createElement('tr')",
		"document.createElement('td')",
		"cell.textContent = String(value)",
		"cell.title = String(title)",
		"dom.procTbody.appendChild(fragment)",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("app.js is missing safe process rendering construct %q", required)
		}
	}
	if !strings.Contains(source, "escapeHtml(String(d.device || ''))") {
		t.Fatal("app.js does not escape disk device names before using innerHTML")
	}
}
