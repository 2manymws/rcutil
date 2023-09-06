package testutil

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
var conf embed.FS

func NewReverseProxyNGINXServer(t testing.TB, hostname string, upstreams map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	tb, err := conf.ReadFile("templates/nginx_reverse.conf.tmpl")
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
		b, _ := os.ReadFile(p)
		t.Logf("%s NGINX config:\n%s\n", hostname, string(b))
	}

	return createNGINXServer(t, hostname, p)
}

func NewUpstreamEchoNGINXServer(t testing.TB, hostname string) string {
	t.Helper()
	dir := t.TempDir()
	tb, err := conf.ReadFile("templates/nginx_echo.conf.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := template.Must(template.New("conf").Parse(string(tb)))
	p := filepath.Join(dir, fmt.Sprintf("%s.nginx_echo.conf", hostname))
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := tmpl.Execute(f, map[string]any{
		"Hostname": hostname,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if os.Getenv("DEBUG") != "" {
		b, _ := os.ReadFile(p)
		t.Logf("%s NGINX config:\n%s\n", hostname, string(b))
	}

	return createNGINXServer(t, hostname, p)
}

func createNGINXServer(t testing.TB, hostname, confp string) string {
	t.Helper()
	dir := t.TempDir()
	sb, err := conf.ReadFile("templates/sleep.js")
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
	opt := &dockertest.RunOptions{
		Name:       hostname,
		Hostname:   hostname,
		Repository: "nginx",
		Tag:        "latest",
		Networks:   []*dockertest.Network{testNetwork(t)},
		Mounts: []string{
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
			_ = tailLogs(t, pool, r, os.Stderr, true)
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
			_ = pool.RemoveNetwork(n)
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
