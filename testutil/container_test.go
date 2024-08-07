package testutil

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"testing"
	"time"
)

func TestContainer(t *testing.T) {
	upstream := "a.example.com"
	urlstr := NewUpstreamEchoNGINXServer(t, upstream, 1024)
	res, err := http.DefaultClient.Get(urlstr)
	if err != nil {
		t.Fatal(err)
	}
	b, err := httputil.DumpResponse(res, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(os.Stderr, "echo NGINX server:\n%s\n", (string(b))); err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	upstreams := map[string]string{}
	upstreams[upstream] = fmt.Sprintf("http://%s:80", "a.example.com")

	proxy := NewReverseProxyNGINXServer(t, "r.example.com", upstreams)
	{
		req, err := http.NewRequest("GET", proxy, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = upstream
		res, err := http.DefaultClient.Get(proxy)
		if err != nil {
			t.Fatal(err)
		}
		b, err := httputil.DumpResponse(res, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(os.Stderr, "reverse proxy NGINX server:\n%s\n", (string(b))); err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
	}

	{
		now := time.Now()
		req, err := http.NewRequest("GET", proxy+"/sleep", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = upstream
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		after := time.Now()
		if after.Sub(now) < 1*time.Second {
			t.Fatal("sleep.js is not working")
		}
	}

	{
		req, err := http.NewRequest("GET", proxy+"/cache/hello", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = upstream
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
	}

	{
		req, err := http.NewRequest("GET", proxy+"/cache/hello", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = upstream
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.Header.Get("X-Nginx-Cache") != "HIT" {
			b, err := httputil.DumpResponse(res, true)
			if err != nil {
				t.Fatal(err)
			}
			t.Fatalf("NGINX cache is not working:\n%s", string(b))
		}
	}
}
