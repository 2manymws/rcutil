package testutil

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	networkName = "rcutil-test-network"
)

//go:embed templates/*
var templates embed.FS

func NewReverseProxyNGINXServer(t testing.TB, hostname string, upstreams map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	tb, err := templates.ReadFile("templates/nginx_reverse.conf.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := template.Must(template.New("conf").Parse(string(tb)))
	p := filepath.Join(dir, fmt.Sprintf("%s.nginx_reverse.conf", hostname))
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := tmpl.Execute(f, map[string]any{
		"Hostname":  hostname,
		"Upstreams": upstreams,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if os.Getenv("DEBUG") != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s NGINX config:\n%s\n", hostname, string(b))
	}

	return createNGINXServer(t, hostname, p, p)
}

func NewUpstreamEchoNGINXServer(t testing.TB, hostname string, bodySize int) string {
	t.Helper()
	dir := t.TempDir()
	cb, err := templates.ReadFile("templates/nginx_echo.conf.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	confp := filepath.Join(dir, fmt.Sprintf("%s.nginx_echo.conf", hostname))
	if err := os.WriteFile(confp, cb, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	tb, err := templates.ReadFile("templates/index.html.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	indexp := filepath.Join(dir, "index.html")
	tmpl := template.Must(template.New("index").Parse(string(tb)))
	f, err := os.Create(indexp)
	if err != nil {
		t.Fatal(err)
	}
	var s strings.Builder
	s.Grow(bodySize)
	for i := 0; i < bodySize; i++ {
		if err := s.WriteByte(0); err != nil {
			t.Fatal(err)
		}
	}
	if err := tmpl.Execute(f, map[string]any{
		"Hostname": hostname,
		"PlusBody": s.String(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DEBUG") != "" {
		b, err := os.ReadFile(confp)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s NGINX config:\n%s\n", hostname, string(b))
	}

	return createNGINXServer(t, hostname, confp, indexp)
}

func createNGINXServer(t testing.TB, hostname, confp, indexp string) string {
	t.Helper()
	dir := t.TempDir()
	sb, err := templates.ReadFile("templates/sleep.js")
	if err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "sleep.js")
	if err := os.WriteFile(sp, sb, 0644); err != nil {
		t.Fatal(err)
	}
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}
	now := time.Now()
	opt := &dockertest.RunOptions{
		Name:       fmt.Sprintf("%s-%s", hostname, now.Format("20060102150405")),
		Hostname:   hostname,
		Repository: "nginx",
		Tag:        "latest",
		Networks:   []*dockertest.Network{testNetwork(t)},
		Mounts: []string{
			fmt.Sprintf("%s:/etc/nginx/index.html:ro", indexp),
			fmt.Sprintf("%s:/etc/nginx/nginx.conf:ro", confp),
			fmt.Sprintf("%s:/etc/nginx/njs/sleep.js:ro", sp),
		},
	}
	r, err := pool.RunWithOptions(opt)
	if err != nil {
		t.Fatalf("Could not start resource: %s", err)
	}

	if os.Getenv("DEBUG") != "" {
		go func() {
			if err := tailLogs(t, pool, r, os.Stderr, true); err != nil {
				t.Error(err)
			}
		}()
	}

	t.Cleanup(func() {
		if err := pool.Purge(r); err != nil {
			t.Fatalf("Could not purge resource: %s", err)
		}
	})

	var urlstr string
	if err := pool.Retry(func() error {
		urlstr = fmt.Sprintf("http://127.0.0.1:%s", r.GetPort("80/tcp"))
		if _, err := http.Get(urlstr); err != nil {
			time.Sleep(1 * time.Second)
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("Could not connect to NGINX server: %s", err)
	}
	return urlstr
}

func testNetwork(t testing.TB) *dockertest.Network {
	t.Helper()
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}
	ns, err := pool.NetworksByName(networkName)
	if err != nil {
		t.Fatalf("Could not connect to docker: %s", err)
	}
	switch len(ns) {
	case 0:
		n, err := pool.CreateNetwork(networkName)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := pool.RemoveNetwork(n); err != nil {
				t.Error(err)
			}
		})
		return n
	case 1:
		// deletion of network is left to the function that created it.
		return &ns[0]
	default:
		t.Fatalf("Could not connect to docker: %s", err)
	}
	return nil
}

func tailLogs(t testing.TB, pool *dockertest.Pool, r *dockertest.Resource, wr io.Writer, follow bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	opts := docker.LogsOptions{
		Context:     ctx,
		Stderr:      true,
		Stdout:      true,
		Follow:      follow,
		Timestamps:  true,
		RawTerminal: false,

		Container: r.Container.ID,

		OutputStream: wr,
		ErrorStream:  wr,
	}
	t.Cleanup(func() {
		cancel()
	})
	return pool.Client.Logs(opts)
}
