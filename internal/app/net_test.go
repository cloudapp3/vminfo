package app

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudapp3/vminfo/internal/i18n"
)

func TestRunNetRequiresAction(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, nil, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on no action, got %v", err)
	}
}

func TestRunNetUnknownAction(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, []string{"frob"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on unknown action, got %v", err)
	}
}

func TestRunNetPortBadPort(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, []string{"port", "example.com", "notaport"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on bad port, got %v", err)
	}
}

func TestRunNetPortMissingPort(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, []string{"port", "example.com"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on missing port, got %v", err)
	}
}

func TestRunNetDNSJSON(t *testing.T) {
	var out bytes.Buffer
	if err := runNet(context.Background(), &out, &bytes.Buffer{}, []string{"dns", "--json", "localhost"}, i18n.New("en")); err != nil {
		t.Fatalf("runNet dns returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"domain": "localhost"`) {
		t.Fatalf("expected JSON with domain field, got: %s", out.String())
	}
}

func TestRunNetPingMissingHost(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, []string{"ping"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on missing host, got %v", err)
	}
}

func TestRunNetPingTCPJSON(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	var out bytes.Buffer
	args := []string{"ping", "--mode", "tcp", "--tcp-port", strconv.Itoa(port), "--count", "2", "--json", "127.0.0.1"}
	if err := runNet(context.Background(), &out, &bytes.Buffer{}, args, i18n.New("en")); err != nil {
		t.Fatalf("runNet ping returned error: %v", err)
	}
	if !strings.Contains(out.String(), `"mode": "tcp"`) || !strings.Contains(out.String(), `"sent": 2`) {
		t.Fatalf("expected tcp ping JSON, got: %s", out.String())
	}
}

func TestRunNetIPTooManyArgs(t *testing.T) {
	err := runNet(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, []string{"ip", "1.1.1.1", "2.2.2.2"}, i18n.New("en"))
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage on too many args, got %v", err)
	}
}
