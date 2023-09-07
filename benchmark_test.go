package rcutil_test

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/k1LoW/rc"
	"github.com/k1LoW/rcutil/testutil"
	"github.com/k1LoW/rp"
	testur "github.com/k1LoW/rp/testutil"
)

func BenchmarkNGINXCache(b *testing.B) {
	hostname := "a.example.com"
	_ = testutil.NewUpstreamEchoNGINXServer(b, hostname)
	var upstreams = map[string]string{
		"a.example.com": fmt.Sprintf("http://%s:80", hostname),
	}
	proxy := testutil.NewReverseProxyNGINXServer(b, "r.example.com", upstreams)

	// Make cache
	const (
		concurrency = 100
		cacherange  = 10000
	)
	limitCh := make(chan struct{}, concurrency)
	wg := &sync.WaitGroup{}
	for i := 0; i < cacherange; i++ {
		i := i
		limitCh <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/%d", proxy, i), nil)
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
		}()
	}

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := rand.Intn(cacherange)
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/%d", proxy, i), nil)
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
			got := res.StatusCode
			want := http.StatusOK
			if res.StatusCode != http.StatusOK {
				b.Errorf("got %v want %v", got, want)
			}
			if res.Header.Get("X-Nginx-Cache") != "HIT" {
				b.Errorf("got %v want %v", got, want)
			}
		}
	})
}

func BenchmarkRC(b *testing.B) {
	hostname := "a.example.com"
	urlstr := testutil.NewUpstreamEchoNGINXServer(b, hostname)
	var upstreams = map[string]string{
		"a.example.com": urlstr,
	}
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
		concurrency = 100
		cacherange  = 10000
	)
	limitCh := make(chan struct{}, concurrency)
	wg := &sync.WaitGroup{}
	for i := 0; i < cacherange; i++ {
		i := i
		limitCh <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/%d", proxy.URL, i), nil)
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
		}()
	}

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := rand.Intn(cacherange)
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/%d", proxy.URL, i), nil)
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
			got := res.StatusCode
			want := http.StatusOK
			if res.StatusCode != http.StatusOK {
				b.Errorf("got %v want %v", got, want)
			}
			if res.Header.Get("X-Cache") != "HIT" {
				b.Errorf("got %v want %v", got, want)
			}
		}
	})
}
