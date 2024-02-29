package rcutil

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/2manymws/rc"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDiskCacheTTL(t *testing.T) {
	root := t.TempDir()
	ttl := 100 * time.Millisecond
	dc, err := NewDiskCache(root, ttl)
	if err != nil {
		t.Fatal(err)
	}
	key := "test"
	want := "hello"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody([]byte(want)),
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
	maxKeys := uint64(1)
	dc, err := NewDiskCache(root, 24*time.Hour, MaxKeys(maxKeys))
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
	maxTotalBytes := uint64(209)
	dc, err := NewDiskCache(root, 24*time.Hour, MaxTotalBytes(maxTotalBytes))
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
	root := t.TempDir()
	maxTotalBytes := uint64(209)
	dc, err := NewDiskCache(root, 24*time.Hour, MaxTotalBytes(maxTotalBytes), EnableAutoAdjust())
	if err != nil {
		t.Fatal(err)
	}
	key1 := "test1"
	key2 := "test2"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody([]byte("hello")),
	}
	if err := dc.Store(key1, req, res); err != nil {
		t.Fatal(err)
	}
	if err := dc.Store(key2, req, res); err != nil {
		t.Error(err)
	}
}

func TestDiskCacheWarmUp(t *testing.T) {
	root := t.TempDir()
	key := "test"
	req := &http.Request{Method: http.MethodGet, Header: http.Header{}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))}
	res := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody([]byte("hello")),
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
		if dc1.totalBytes == 0 {
			t.Error("totalBytes > 0")
		}
		_, got, err := dc1.Load(key)
		if err != nil {
			t.Error(err)
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
