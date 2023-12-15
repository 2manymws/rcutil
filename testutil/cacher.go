package testutil

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"testing"

	"github.com/2manymws/rcutil"
)

var (
	_ Cacher = &AllCache{}
)

type Cacher interface {
	Load(req *http.Request) (cachedReq *http.Request, cachedRes *http.Response, err error)
	Store(req *http.Request, res *http.Response) error
	Hit() int
}

type AllCache struct {
	t  testing.TB
	dc *rcutil.DiskCache
}

func NewAllCache(t testing.TB) *AllCache {
	t.Helper()
	dc, err := rcutil.NewDiskCache(t.TempDir(), rcutil.NoLimitTTL)
	if err != nil {
		t.Fatal(err)
	}
	return &AllCache{
		t:  t,
		dc: dc,
	}
}

func (c *AllCache) Load(req *http.Request) (*http.Request, *http.Response, error) {
	c.t.Helper()
	seed, err := rcutil.Seed(req, []string{})
	if err != nil {
		return nil, nil, err
	}
	key := seedToKey(seed)
	req, res, err := c.dc.Load(key)
	if err != nil {
		return nil, nil, err
	}
	rcutil.SetCacheResultHeader(res, true)
	return req, res, nil
}

func (c *AllCache) Store(req *http.Request, res *http.Response) error {
	c.t.Helper()
	seed, err := rcutil.Seed(req, []string{})
	if err != nil {
		return err
	}
	key := seedToKey(seed)
	if err := c.dc.Store(key, req, res); err != nil {
		return err
	}
	return nil
}

func (c *AllCache) Hit() int {
	m := c.dc.Metrics()
	return int(m.Hits)
}

func seedToKey(seed string) string {
	sha1 := sha1.New()
	_, _ = io.WriteString(sha1, seed)
	return hex.EncodeToString(sha1.Sum(nil))
}
