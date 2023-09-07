package testutil

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/k1LoW/rcutil"
)

var (
	_ Cacher = &AllCache{}
)

// errCacheNotFound is returned when the cache is not found
var errCacheNotFound error = errors.New("cache not found")

// errNoCache is returned if not caching
var errNoCache error = errors.New("no cache")

type Cacher interface {
	Name() string
	Load(req *http.Request) (res *http.Response, err error)
	Store(req *http.Request, res *http.Response) error
	Hit() int
}

type AllCache struct {
	t   testing.TB
	m   map[string]string
	dir string
	hit int
	mu  sync.Mutex
}

type GetOnlyCache struct {
	t   testing.TB
	m   map[string]string
	dir string
	hit int
	mu  sync.Mutex
}

func NewAllCache(t testing.TB) *AllCache {
	t.Helper()
	return &AllCache{
		t:   t,
		m:   map[string]string{},
		dir: t.TempDir(),
	}
}

func (c *AllCache) Name() string {
	c.t.Helper()
	return "all"
}

func (c *AllCache) Load(req *http.Request) (res *http.Response, err error) {
	c.t.Helper()
	seed, err := rcutil.Seed(req, []string{})
	if err != nil {
		return nil, err
	}
	key := seedToKey(seed)
	c.mu.Lock()
	p, ok := c.m[key]
	c.mu.Unlock()
	if !ok {
		return nil, errCacheNotFound
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, errCacheNotFound
	}
	defer f.Close()
	res, err = rcutil.LoadResponse(f)
	if err != nil {
		return nil, err
	}
	res.Header.Set("X-Cache", "HIT")
	c.hit++
	return res, nil
}

func (c *AllCache) Store(req *http.Request, res *http.Response) error {
	c.t.Helper()
	seed, err := rcutil.Seed(req, []string{})
	if err != nil {
		return err
	}
	key := seedToKey(seed)
	p := filepath.Join(c.dir, key)
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := rcutil.StoreResponse(res, f); err != nil {
		return err
	}
	c.mu.Lock()
	c.m[key] = p
	c.mu.Unlock()
	return nil
}

func (c *AllCache) Hit() int {
	return c.hit
}

func seedToKey(seed string) string {
	sha1 := sha1.New()
	_, _ = io.WriteString(sha1, seed)
	return hex.EncodeToString(sha1.Sum(nil))
}