package rcutil

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/2manymws/rc"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/sync/errgroup"
)

func TestDiskCacheTTL(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		t.Fatal(err)
	}
	ttl := 100 * time.Millisecond
	dc, err := NewDiskCache(cacheRoot, ttl)
	if err != nil {
		t.Fatal(err)
	}
	key := "test"
	want := "hello"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
		StatusCode: http.StatusOK,
		Header: http.Header{
			"X-Test":         []string{"test"},
			"Content-Length": []string{fmt.Sprintf("%d", len([]byte(want)))},
		},
		Body:          newBody([]byte(want)),
		ContentLength: int64(len([]byte(want))),
	}
	if err := dc.Store(key, req, res); err != nil {
		t.Fatal(err)
	}
	_, got, err := dc.Load(key)
	if err != nil {
		t.Error(err)
	}
	opts := []cmp.Option{
		cmpopts.IgnoreFields(http.Response{}, "Body"),
	}
	if diff := cmp.Diff(res, got, opts...); diff != "" {
		t.Error(diff)
	}
	{
		got := readBody(got.Body)
		if diff := cmp.Diff(got, want); diff != "" {
			t.Error(diff)
		}
	}
	time.Sleep(ttl)
	{
		_, _, err := dc.Load(key)
		if !errors.Is(err, rc.ErrCacheNotFound) {
			t.Error(err)
		}
	}
}

func TestDiskCacheMaxKeys(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		t.Fatal(err)
	}
	maxKeys := uint64(1)
	dc, err := NewDiskCache(cacheRoot, 24*time.Hour, MaxKeys(maxKeys))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		key := fmt.Sprintf("test%d", i)
		body := "hello"
		req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
		res := &http.Response{
			Status:     http.StatusText(http.StatusOK),
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Test": []string{"test"}},
			Body:       newBody([]byte(body)),
		}
		if err := dc.Store(key, req, res); err != nil {
			t.Fatal(err)
		}
	}

	{
		key := "test1"
		_, _, err := dc.Load(key)
		if err != nil {
			t.Error(err)
		}
	}

	{
		key := "test0"
		_, _, err := dc.Load(key)
		if !errors.Is(err, rc.ErrCacheNotFound) {
			t.Error(err)
		}
	}
}

func TestDiskCacheMaxTotalBytes(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		t.Fatal(err)
	}
	maxTotalBytes := uint64(209)
	dc, err := NewDiskCache(cacheRoot, 24*time.Hour, MaxTotalBytes(maxTotalBytes), DisableWarmUp())
	if err != nil {
		t.Fatal(err)
	}

	key := "test1"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody([]byte("hello")),
	}
	if err := dc.Store(key, req, res); err != nil {
		t.Fatal(err)
	}
	if err := dc.Store(key, req, res); !errors.Is(err, ErrCacheFull) {
		t.Error(err)
	}
}

func TestAutoAdjust(t *testing.T) {
	tests := []struct {
		name             string
		enableAutoAdjust bool
		stopAdjust       bool
		wantErr          bool
	}{
		{"diable auto adjust", false, false, true},
		{"enable auto adjust", true, false, false},
		{"enable auto adjust and call StopAdjust", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			maxTotalBytes := uint64(1000)
			opts := []DiskCacheOption{
				MaxTotalBytes(maxTotalBytes),
			}
			if tt.enableAutoAdjust {
				opts = append(opts, EnableAutoAdjust())
			}
			dc, err := NewDiskCache(root, 24*time.Hour, opts...)
			if err != nil {
				t.Fatal(err)
			}
			if tt.stopAdjust {
				dc.StopAdjust()
			}
			eg := new(errgroup.Group)
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("test%d", i)
				eg.Go(func() error {
					req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
					res := &http.Response{
						Status:     http.StatusText(http.StatusOK),
						StatusCode: http.StatusOK,
						Header:     http.Header{"X-Test": []string{"test"}},
						Body:       newBody([]byte("hello")),
					}
					return dc.Store(key, req, res)
				})
			}
			if err := eg.Wait(); (err != nil) != tt.wantErr {
				t.Error(err)
			}
			if dc.maxTotalBytes > maxTotalBytes {
				t.Errorf("maxTotalBytes: got %d, want %d", dc.maxTotalBytes, maxTotalBytes)
			}
		})
	}
}

func TestDiskCacheWarmUp(t *testing.T) {
	root := t.TempDir()
	key := "test"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
		StatusCode: http.StatusOK,
		Header: http.Header{
			"X-Test":         []string{"test"},
			"Content-Length": []string{fmt.Sprintf("%d", len([]byte("hello")))},
		},
		Body:          newBody([]byte("hello")),
		ContentLength: int64(len([]byte("hello"))),
	}

	dc0, err := NewDiskCache(root, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := dc0.Store(key, req, res); err != nil {
		t.Fatal(err)
	}
	if _, _, err := dc0.Load(key); err != nil {
		t.Fatal(err)
	}

	t.Run("Warm up is enabled, so cache can be loaded", func(t *testing.T) {
		dc1, err := NewDiskCache(root, 24*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		// wait warm up
		for i := 0; i < 100; i++ {
			dc1.mu.Lock()
			if dc1.totalBytes > 0 {
				dc1.mu.Unlock()
				break
			}
			dc1.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}

		dc1.mu.Lock()
		if dc1.totalBytes == 0 {
			dc1.mu.Unlock()
			t.Error("totalBytes > 0")
			return
		}
		dc1.mu.Unlock()
		_, got, err := dc1.Load(key)
		if err != nil {
			t.Error(err)
			return
		}
		defer got.Body.Close()
		opts := []cmp.Option{
			cmpopts.IgnoreFields(http.Response{}, "Body"),
		}
		if diff := cmp.Diff(res, got, opts...); diff != "" {
			t.Error(diff)
		}
		gotb := readBody(got.Body)
		if diff := cmp.Diff(gotb, "hello"); diff != "" {
			t.Error(diff)
		}
	})

	t.Run("Warm up is disabled, so cache cannot be loaded", func(t *testing.T) {
		dc2, err := NewDiskCache(root, 24*time.Hour, DisableWarmUp())
		if err != nil {
			t.Fatal(err)
		}

		if dc2.totalBytes > 0 {
			t.Error("totalBytes == 0")
		}
		if _, _, err := dc2.Load(key); err == nil {
			t.Error("load should fail")
		}
	})
}

func TestDiskCacheStopAll(t *testing.T) {
	root := t.TempDir()
	cacheRoot := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		t.Fatal(err)
	}
	dc, err := NewDiskCache(cacheRoot, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	dc.StopAll()
}
