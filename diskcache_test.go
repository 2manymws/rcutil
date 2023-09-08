package rcutil

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

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
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody(want),
	}
	if err := dc.Store(key, res); err != nil {
		t.Fatal(err)
	}
	got, err := dc.Load(key)
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
		_, err := dc.Load(key)
		if !errors.Is(err, ErrCacheNotFound) {
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
		res := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Test": []string{"test"}},
			Body:       newBody(body),
		}
		if err := dc.Store(key, res); err != nil {
			t.Fatal(err)
		}
	}

	{
		key := "test1"
		_, err := dc.Load(key)
		if err != nil {
			t.Error(err)
		}
	}

	{
		key := "test0"
		_, err := dc.Load(key)
		if !errors.Is(err, ErrCacheNotFound) {
			t.Error(err)
		}
	}
}

func TestDiskCacheMaxTotalBytes(t *testing.T) {
	root := t.TempDir()
	maxTotalBytes := uint64(len(`{"status_code":200,"header":{"X-Test":["test"]},"body":"aGVsbG8="}`+"\n") + 1)
	dc, err := NewDiskCache(root, 24*time.Hour, MaxTotalBytes(maxTotalBytes))
	if err != nil {
		t.Fatal(err)
	}
	key := "test1"
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody("hello"),
	}
	if err := dc.Store(key, res); err != nil {
		t.Fatal(err)
	}
	if err := dc.Store(key, res); !errors.Is(err, ErrCacheFull) {
		t.Error(err)
	}
}

func TestDiskCacheWarmUp(t *testing.T) {
	root := t.TempDir()
	key := "test"
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody("hello"),
	}

	dc0, err := NewDiskCache(root, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := dc0.Store(key, res); err != nil {
		t.Fatal(err)
	}
	if _, err := dc0.Load(key); err != nil {
		t.Fatal(err)
	}

	t.Run("Warm up is enabled, so cache can be loaded", func(t *testing.T) {
		dc1, err := NewDiskCache(root, 24*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		got, err := dc1.Load(key)
		if err != nil {
			t.Fatal(err)
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
		if _, err := dc2.Load(key); err == nil {
			t.Error("load should fail")
		}
	})
}
