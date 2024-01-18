package rcutil_test

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2manymws/rc"
	"github.com/2manymws/rcutil"
	"github.com/2manymws/rcutil/testutil"
	"github.com/2manymws/rp"
	testur "github.com/2manymws/rp/testutil"
)

func BenchmarkNGINXCache1MBBody(b *testing.B) {
	const (
		hostname = "a.example.com"
		bodySize = 1024 * 1024 // 'OK from {{ .Hostname }}!!' + 1MB
	)
	_ = testutil.NewUpstreamEchoNGINXServer(b, hostname, bodySize)
	upstreams := map[string]string{}
	upstreams[hostname] = fmt.Sprintf("http://%s:80", hostname)
	proxy := testutil.NewReverseProxyNGINXServer(b, "r.example.com", upstreams)

	// Make cache
	const (
		concurrency = 1
		cacherange  = 1000
	)
	testutil.WarmUpToCreateCache(b, proxy, hostname, concurrency, cacherange)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := rand.Intn(cacherange)
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/cache/%d", proxy, i), nil)
			if err != nil {
				b.Error(err)
				return
			}
			req.Host = hostname
			req.Close = true
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				b.Error(err)
				return
			}
			res.Body.Close()
			if res.Header.Get("X-Nginx-Cache") != "HIT" {
				b.Errorf("cache miss: %v %s", req.Header, req.URL.String())
			}
		}
	})
}

func BenchmarkDiscCache1MBBody(b *testing.B) {
	const (
		hostname = "a.example.com"
		bodySize = 1024 * 1024 // 'OK from {{ .Hostname }}!!' + 1MB
	)
	urlstr := testutil.NewUpstreamEchoNGINXServer(b, hostname, bodySize)
	upstreams := map[string]string{}
	upstreams[hostname] = urlstr

	c := testutil.NewAllCache(b)
	m := rc.New(c)
	rl := testur.NewRelayer(upstreams)
	r := rp.NewRouter(rl)
	proxy := httptest.NewServer(m(r))
	b.Cleanup(func() {
		proxy.Close()
	})

	// Make cache
	const (
		concurrency = 1
		cacherange  = 1000
	)
	testutil.WarmUpToCreateCache(b, proxy.URL, hostname, concurrency, cacherange)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := rand.Intn(cacherange)
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/cache/%d", proxy.URL, i), nil)
			if err != nil {
				b.Error(err)
				return
			}
			req.Host = hostname
			req.Close = true
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				b.Error(err)
				return
			}
			res.Body.Close()
			if res.Header.Get(rcutil.CacheResultHeader) != rcutil.CacheHit {
				b.Errorf("cache miss: %s", req.URL.String())
			}
		}
	})
}

func BenchmarkEncodeDecode1MBBody(b *testing.B) {
	const bodySize = 1024 * 1024 // 1MB
	var sb strings.Builder
	sb.Grow(bodySize)
	for i := 0; i < bodySize; i++ {
		sb.WriteByte(0)
	}
	dir := b.TempDir()
	p := filepath.Join(dir, "cachefile")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(sb.String())),
		}
		c, err := os.Create(p)
		if err != nil {
			b.Error(err)
			return
		}
		if err := rcutil.EncodeReqRes(req, res, c); err != nil {
			b.Error(err)
			return
		}
		if err := c.Close(); err != nil {
			b.Error(err)
			return
		}
		cc, err := os.Open(p)
		if err != nil {
			b.Error(err)
			return
		}
		if _, _, err := rcutil.DecodeReqRes(cc); err != nil {
			b.Error(err)
			return
		}
		if err := cc.Close(); err != nil {
			b.Error(err)
			return
		}
	}
}
