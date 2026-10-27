// Command spidomtr load tests a single URL using the spidomtr library.
//
// Usage:
//
//	spidomtr [flags] <url>
//
// Run "spidomtr -h" for the list of available flags.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/spider-pigs/spidomtr"
	"github.com/spider-pigs/spidomtr/pkg/handlers"
	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The first Ctrl+C cancels in-flight requests so the summary is still
	// printed; a second one terminates immediately.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		signal.Stop(sig)
		cancel()
	}()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// headerFlags collects repeatable "Key: Value" header flags.
type headerFlags http.Header

func (h headerFlags) String() string {
	parts := make([]string, 0, len(h))
	for k, vs := range h {
		for _, v := range vs {
			parts = append(parts, k+": "+v)
		}
	}
	return strings.Join(parts, ", ")
}

func (h headerFlags) Set(s string) error {
	i := strings.Index(s, ":")
	if i <= 0 {
		return fmt.Errorf("invalid header %q, expected \"Key: Value\"", s)
	}
	http.Header(h).Add(strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]))
	return nil
}

// intListFlag parses a comma separated list of integers.
type intListFlag []int

func (l *intListFlag) String() string {
	if l == nil {
		return ""
	}
	parts := make([]string, len(*l))
	for i, v := range *l {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func (l *intListFlag) Set(s string) error {
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("invalid integer %q", p)
		}
		out = append(out, v)
	}
	*l = out
	return nil
}

type config struct {
	url              string
	method           string
	headers          http.Header
	body             []byte
	iterations       int
	users            int
	timeout          time.Duration
	expectStatus     []int
	insecure         bool
	disableKeepAlive bool
	noRedirects      bool
	id               string
	description      string
	buckets          int
	percentiles      []int
	progress         bool
	verbose          bool
	showLogo         bool
	failOnError      bool
}

// errReported marks errors that have already been printed to the user.
var errReported = errors.New("reported")

func parseFlags(args []string, stderr io.Writer) (*config, error) {
	cfg := &config{headers: http.Header{}}
	expect := intListFlag{}
	percentiles := intListFlag(append([]int(nil), spidomtr.DefaultPercentiles...))
	var body, bodyFile string
	var noLogo bool

	fs := flag.NewFlagSet("spidomtr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage:\n  spidomtr [flags] <url>   load test a URL\n  spidomtr version         print version information\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprint(stderr, "\nExample:\n  spidomtr -n 100 -c 10 -m POST -H 'Content-Type: application/json' -d '{\"a\":1}' https://example.com/api\n")
	}

	fs.IntVar(&cfg.iterations, "n", 10, "number of requests per user")
	fs.IntVar(&cfg.users, "c", 1, "number of concurrent users")
	fs.DurationVar(&cfg.timeout, "t", 10*time.Second, "timeout per request")
	fs.StringVar(&cfg.method, "m", http.MethodGet, "HTTP method")
	fs.Var(headerFlags(cfg.headers), "H", "request header \"Key: Value\" (repeatable)")
	fs.StringVar(&body, "d", "", "request body")
	fs.StringVar(&bodyFile, "D", "", "read request body from file")
	fs.Var(&expect, "expect", "comma separated list of accepted status codes (default: any status < 400)")
	fs.BoolVar(&cfg.insecure, "k", false, "skip TLS certificate verification")
	fs.BoolVar(&cfg.disableKeepAlive, "disable-keepalive", false, "disable HTTP keep-alive")
	fs.BoolVar(&cfg.noRedirects, "no-redirects", false, "do not follow redirects")
	fs.StringVar(&cfg.id, "id", "", "id of the test run, shown with -v (default: the URL)")
	fs.StringVar(&cfg.description, "desc", "", "description of the test run, shown with -v")
	fs.IntVar(&cfg.buckets, "buckets", spidomtr.DefaultHistogramBuckets, "number of histogram buckets")
	fs.Var(&percentiles, "percentiles", "comma separated list of latency percentiles")
	fs.BoolVar(&cfg.progress, "progress", true, "show progress bar")
	fs.BoolVar(&cfg.verbose, "v", false, "print every request result after the run")
	fs.BoolVar(&noLogo, "no-logo", false, "hide the ascii logo")
	fs.BoolVar(&cfg.failOnError, "fail-on-error", false, "exit with status 1 if any request failed")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, err
		}
		// The flag package has already printed the error and usage.
		return nil, fmt.Errorf("%w: %v", errReported, err)
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return nil, errors.New("exactly one url is required")
	}
	cfg.url = fs.Arg(0)
	u, err := url.Parse(cfg.url)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid url %q: must be an absolute http(s) url", cfg.url)
	}

	if cfg.iterations < 1 {
		return nil, errors.New("-n must be at least 1")
	}
	if cfg.users < 1 {
		return nil, errors.New("-c must be at least 1")
	}
	if cfg.timeout <= 0 {
		return nil, errors.New("-t must be positive")
	}
	if cfg.buckets < 1 {
		return nil, errors.New("-buckets must be at least 1")
	}
	for _, p := range percentiles {
		if p < 1 || p > 100 {
			return nil, fmt.Errorf("invalid percentile %d: must be between 1 and 100", p)
		}
	}
	for _, s := range expect {
		if s < 100 || s > 599 {
			return nil, fmt.Errorf("invalid status code %d", s)
		}
	}
	if body != "" && bodyFile != "" {
		return nil, errors.New("-d and -D are mutually exclusive")
	}

	cfg.method = strings.ToUpper(cfg.method)
	if _, err := http.NewRequest(cfg.method, cfg.url, nil); err != nil {
		return nil, err
	}
	cfg.expectStatus = expect
	cfg.percentiles = percentiles
	cfg.showLogo = !noLogo
	if cfg.id == "" {
		cfg.id = cfg.url
	}
	// Identify the tool unless the user sets User-Agent, even to an empty
	// value (which makes net/http omit the header).
	if _, ok := cfg.headers["User-Agent"]; !ok {
		cfg.headers.Set("User-Agent", userAgent())
	}

	switch {
	case bodyFile != "":
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		cfg.body = b
	case body != "":
		cfg.body = []byte(body)
	}

	return cfg, nil
}

// version returns the module version from the build info. Release builds
// (go install ...@vX.Y.Z) report the tag, local builds a pseudo-version and
// builds without version information (go run, -buildvcs=false) "devel".
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}

func userAgent() string {
	return "spidomtr/" + version()
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "spidomtr %s (%s %s/%s)\n", version(), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func newClient(cfg *config) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = cfg.users
	transport.DisableKeepAlives = cfg.disableKeepAlive
	if cfg.insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // opt-in via -k
	}

	client := &http.Client{Transport: transport}
	if cfg.noRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}

func statusAccepted(code int, expected []int) bool {
	if len(expected) == 0 {
		return code < 400
	}
	for _, e := range expected {
		if code == e {
			return true
		}
	}
	return false
}

func newHTTPTest(cfg *config, client *http.Client) testunit.TestUnit {
	return testunit.New(
		testunit.ID(cfg.method+" "+cfg.url),
		testunit.Test(func(ctx context.Context, _ []interface{}) ([]interface{}, error) {
			var body io.Reader
			if cfg.body != nil {
				body = bytes.NewReader(cfg.body)
			}
			req, err := http.NewRequestWithContext(ctx, cfg.method, cfg.url, body)
			if err != nil {
				return nil, err
			}
			req.Header = cfg.headers.Clone()
			// net/http ignores a Host header; it must be set on the request.
			if host := req.Header.Get("Host"); host != "" {
				req.Host = host
			}

			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			// Drain the body so the connection can be reused and the full
			// response time is measured.
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				return nil, err
			}

			if !statusAccepted(resp.StatusCode, cfg.expectStatus) {
				return nil, fmt.Errorf("unexpected status: %s", resp.Status)
			}
			return nil, nil
		}),
	)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "version" {
		if len(args) > 1 {
			fmt.Fprintln(stderr, "error: version takes no arguments")
			return 2
		}
		printVersion(stdout)
		return 0
	}

	cfg, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		if !errors.Is(err, errReported) {
			fmt.Fprintf(stderr, "error: %v\n", err)
		}
		return 2
	}

	var hs []spidomtr.RunnerHandler
	if cfg.progress {
		hs = append(hs, handlers.ProgressBar())
	}
	if cfg.verbose {
		hs = append(hs, handlers.Logger())
	}

	runner := spidomtr.NewRunner(
		spidomtr.ID(cfg.id),
		spidomtr.Description(cfg.description),
		spidomtr.Iterations(cfg.iterations),
		spidomtr.Users(cfg.users),
		spidomtr.Timeout(cfg.timeout),
		spidomtr.HistogramBuckets(cfg.buckets),
		spidomtr.Percentiles(cfg.percentiles),
		spidomtr.ShowLogo(cfg.showLogo),
		spidomtr.Handlers(hs...),
	)

	res := runner.Run(ctx, newHTTPTest(cfg, newClient(cfg)))

	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "interrupted")
		return 130
	}
	if cfg.failOnError && res.Stats.Errors > 0 {
		return 1
	}
	return 0
}
