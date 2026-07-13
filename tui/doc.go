// Package tui exposes the interactive terminal UI used by the vminfo CLI so it
// can be embedded in other Go programs.
//
// [Run] starts the same full-screen, keyboard-driven dashboard the vminfo binary
// shows: live CPU, memory, network, and disk metrics, TCP and conntrack state,
// a process list, and host metadata. It requires a real TTY on the provided
// Options.Stdin and Options.Stdout; in a non-interactive context Run returns an
// error.
//
// The UI language is selected via Options.Lang (for example "en" or "zh"); when
// empty it is auto-detected from the VMINFO_LANG, LC_ALL, and LANG environment
// variables.
//
// Example:
//
//	err := tui.Run(ctx, tui.Options{Lang: "en"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Host metric collection lives in the root package at
// [github.com/cloudapp3/vminfo].
package tui
