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
	dc := NewDiskCache(root, ttl, NoLimitKeyCapacity, NoLimitMaxCacheBytes)
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

func TestDiskCacheKeyCapacity(t *testing.T) {
	root := t.TempDir()
	capacity := uint64(1)
	dc := NewDiskCache(root, 24*time.Hour, capacity, NoLimitMaxCacheBytes)
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

func TestDiskCacheMaxCacheBytes(t *testing.T) {
	root := t.TempDir()
	maxCacheBytes := len(`{"status_code":200,"header":{"X-Test":["test"]},"body":"aGVsbG8="}`+"\n") + 1
	dc := NewDiskCache(root, 24*time.Hour, NoLimitKeyCapacity, maxCacheBytes)
	key := "test1"
	res := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Test": []string{"test"}},
		Body:       newBody("hello"),
	}
	if err := dc.Store(key, res); err != nil {
		t.Fatal(err)
	}
	err := dc.Store(key, res)
	if !errors.Is(err, ErrCacheFull) {
		t.Error(err)
	}
}
