package textsafe

import "testing"

func TestTerminal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain unicode", input: "vminfo process", want: "vminfo process"},
		{name: "CSI color", input: "before\x1b[31mafter\x1b[0m", want: "beforeafter"},
		{name: "C1 CSI color", input: "before\u009b31mafter", want: "beforeafter"},
		{name: "OSC with bell", input: "before\x1b]0;malicious-title\aafter", want: "beforeafter"},
		{name: "OSC with ST", input: "before\x1b]8;;https://example.invalid\x1b\\after", want: "beforeafter"},
		{name: "C1 OSC", input: "before\u009dmalicious-title\u009cafter", want: "beforeafter"},
		{name: "C0 and C1", input: "a\n\tb\u007fc\u0085d", want: "abcd"},
		{name: "generic escape", input: "before\x1b7after", want: "beforeafter"},
		{name: "unterminated OSC", input: "before\x1b]0;malicious-title", want: "before"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Terminal(tt.input); got != tt.want {
				t.Fatalf("Terminal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
