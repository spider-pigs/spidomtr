package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseFlagsDefaults(t *testing.T) {
	cfg, err := parseFlags([]string{"http://example.com"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.method != http.MethodGet || cfg.iterations != 10 || cfg.users != 1 ||
		cfg.timeout != 10*time.Second || cfg.id != "http://example.com" || !cfg.progress || !cfg.showLogo {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestParseFlagsAll(t *testing.T) {
	dir := t.TempDir()
	bodyFile := filepath.Join(dir, "body.json")
	if err := os.WriteFile(bodyFile, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := parseFlags([]string{
		"-n", "5", "-c", "3", "-t", "2s", "-m", "post",
		"-H", "Content-Type: application/json", "-H", "X-Token: a:b",
		"-D", bodyFile, "-expect", "200,201", "-percentiles", "50,99",
		"-k", "-no-logo", "-progress=false", "-id", "run",
		"https://example.com/api",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.iterations != 5 || cfg.users != 3 || cfg.timeout != 2*time.Second || cfg.method != "POST" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
	if got := cfg.headers.Get("X-Token"); got != "a:b" {
		t.Errorf("header X-Token = %q", got)
	}
	if string(cfg.body) != `{"a":1}` {
		t.Errorf("body = %q", cfg.body)
	}
	if !reflect.DeepEqual(cfg.expectStatus, []int{200, 201}) || !reflect.DeepEqual(cfg.percentiles, []int{50, 99}) {
		t.Errorf("expect=%v percentiles=%v", cfg.expectStatus, cfg.percentiles)
	}
	if !cfg.insecure || cfg.showLogo || cfg.progress || cfg.id != "run" {
		t.Errorf("unexpected bools/id: %+v", cfg)
	}
}

func TestParseFlagsErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"a", "b"},
		{"example.com"},
		{"ftp://example.com"},
		{"-n", "0", "http://x"},
		{"-c", "0", "http://x"},
		{"-t", "0s", "http://x"},
		{"-H", "nocolon", "http://x"},
		{"-expect", "abc", "http://x"},
		{"-expect", "42", "http://x"},
		{"-percentiles", "0", "http://x"},
		{"-d", "x", "-D", "f", "http://x"},
		{"-D", "/does/not/exist", "http://x"},
		{"-m", "BAD METHOD", "http://x"},
	}
	for _, args := range cases {
		if _, err := parseFlags(args, io.Discard); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}

func TestStatusAccepted(t *testing.T) {
	if !statusAccepted(302, nil) || statusAccepted(404, nil) {
		t.Error("default status check wrong")
	}
	if !statusAccepted(404, []int{404}) || statusAccepted(200, []int{404}) {
		t.Error("explicit status check wrong")
	}
}

func TestRunAgainstServer(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("X-Test") != "yes" || string(b) != "hello" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	base := []string{"-n", "4", "-c", "3", "-m", "POST", "-H", "X-Test: yes", "-d", "hello",
		"-no-logo", "-progress=false", "-fail-on-error"}

	if code := run(context.Background(), append(base, srv.URL), io.Discard); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if h := atomic.LoadInt32(&hits); h != 12 {
		t.Errorf("hits = %d, want 12", h)
	}

	if code := run(context.Background(), append(base, "-expect", "200", srv.URL), io.Discard); code != 1 {
		t.Errorf("exit code = %d, want 1 for unexpected status", code)
	}
}

func TestRunUserAgent(t *testing.T) {
	agents := make(chan []string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agents <- r.Header.Values("User-Agent")
	}))
	defer srv.Close()

	cases := []struct {
		name  string
		flags []string
		check func([]string) bool
	}{
		{"default", nil, func(ua []string) bool { return len(ua) == 1 && strings.HasPrefix(ua[0], "spidomtr") }},
		{"override", []string{"-H", "user-agent: custom/1.0"}, func(ua []string) bool { return len(ua) == 1 && ua[0] == "custom/1.0" }},
		{"disabled", []string{"-H", "User-Agent:"}, func(ua []string) bool { return len(ua) == 0 }},
	}
	for _, c := range cases {
		args := append([]string{"-n", "1", "-no-logo", "-progress=false"}, c.flags...)
		if code := run(context.Background(), append(args, srv.URL), io.Discard); code != 0 {
			t.Fatalf("%s: exit code = %d", c.name, code)
		}
		if ua := <-agents; !c.check(ua) {
			t.Errorf("%s: User-Agent = %q", c.name, ua)
		}
	}
}

func TestRunHostHeader(t *testing.T) {
	hosts := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hosts <- r.Host
	}))
	defer srv.Close()

	args := []string{"-n", "1", "-H", "Host: custom.example", "-no-logo", "-progress=false", srv.URL}
	if code := run(context.Background(), args, io.Discard); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := <-hosts; got != "custom.example" {
		t.Errorf("Host = %q, want custom.example", got)
	}
}

// Run with -race to verify handlers are safe with concurrent users.
func TestRunVerboseConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	args := []string{"-n", "20", "-c", "4", "-v", "-no-logo", srv.URL}
	if code := run(context.Background(), args, io.Discard); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
}

func TestRunInterrupted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args := []string{"-n", "3", "-no-logo", "-progress=false", srv.URL}
	if code := run(ctx, args, io.Discard); code != 130 {
		t.Errorf("exit code = %d, want 130", code)
	}
}

func TestRunUsageErrors(t *testing.T) {
	var sb strings.Builder
	if code := run(context.Background(), []string{"-h"}, &sb); code != 0 {
		t.Errorf("-h exit code = %d", code)
	}
	if !strings.Contains(sb.String(), "Usage: spidomtr") {
		t.Errorf("usage not printed: %q", sb.String())
	}
	if code := run(context.Background(), nil, io.Discard); code != 2 {
		t.Errorf("missing url exit code = %d, want 2", code)
	}

	sb.Reset()
	if code := run(context.Background(), []string{"-n", "abc", "http://x"}, &sb); code != 2 {
		t.Errorf("bad flag exit code = %d, want 2", code)
	}
	if n := strings.Count(sb.String(), `invalid value "abc"`); n != 1 {
		t.Errorf("flag error printed %d times, want 1:\n%s", n, sb.String())
	}
}
