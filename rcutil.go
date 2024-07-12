package rcutil

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	CacheResultHeader = "X-Cache"
	CacheHit          = "HIT"
	CacheMiss         = "MISS"
)

var ErrNoRequest = errors.New("no request")
var ErrInvalidRequest = errors.New("invalid request")

// Seed returns seed for cache key.
// The return value seed is NOT path-safe.
func Seed(req *http.Request, vary []string) (string, error) {
	if req == nil {
		return "", ErrNoRequest
	}
	if req.URL == nil {
		return "", ErrInvalidRequest
	}
	if req.Host == "" {
		return "", ErrInvalidRequest
	}
	const sep = "|"
	// Use req.Host ( does not use req.URL.Host )
	// See https://httpwg.org/specs/rfc9110.html#rfc.section.7.1 and https://httpwg.org/specs/rfc9110.html#rfc.section.7.2
	seed := req.Method + sep + req.Host + sep + req.URL.Path + sep + req.URL.RawQuery
	for _, h := range vary {
		if vv := req.Header.Get(h); vv != "" {
			seed += sep + h + ":" + vv
		}
	}
	return strings.ToLower(seed), nil
}

// EncodeReq encodes http.Request.
func EncodeReq(req *http.Request, w io.Writer) error {
	return req.Write(w)
}

// EncodeRes encodes http.Response.
func EncodeRes(res *http.Response, w io.Writer) error {
	return res.Write(w)
}

// DecodeReq decodes to http.Request
func DecodeReq(r io.Reader) (*http.Request, error) {
	return http.ReadRequest(bufio.NewReader(r))
}

// DecodeRes decodes to http.Response
func DecodeRes(r io.Reader) (*http.Response, error) {
	return http.ReadResponse(bufio.NewReader(r), nil)
}

type cachedReqRes struct {
	Method    string      `json:"method"`
	Host      string      `json:"host"`
	URL       string      `json:"url"`
	ReqHeader http.Header `json:"req_header"`
	ReqBody   []byte      `json:"req_body"`

	StatusCode int         `json:"status_code"`
	ResHeader  http.Header `json:"res_header"`
	ResBody    []byte      `json:"res_body"`
}

// EncodeReqRes encodes http.Request and http.Response.
func EncodeReqRes(req *http.Request, res *http.Response, w io.Writer) error {
	c := &cachedReqRes{
		Method:    req.Method,
		Host:      req.Host,
		URL:       req.URL.String(),
		ReqHeader: req.Header,

		StatusCode: res.StatusCode,
		ResHeader:  res.Header,
	}
	{
		// FIXME: Use stream
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		defer req.Body.Close()
		c.ReqBody = b
	}

	{
		// FIXME: Use stream
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		c.ResBody = b
	}

	if err := gob.NewEncoder(w).Encode(c); err != nil {
		return err
	}
	return nil
}

// DecodeReqRes decodes to http.Request and http.Response.
func DecodeReqRes(r io.Reader) (*http.Request, *http.Response, error) {
	c := &cachedReqRes{}
	if err := gob.NewDecoder(r).Decode(c); err != nil {
		return nil, nil, err
	}
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, nil, err
	}
	req := &http.Request{
		Method: c.Method,
		Host:   c.Host,
		URL:    u,
		Header: c.ReqHeader,
		Body:   io.NopCloser(bytes.NewReader(c.ReqBody)),
	}
	res := &http.Response{
		Status:     http.StatusText(c.StatusCode),
		StatusCode: c.StatusCode,
		Header:     c.ResHeader,
		Body:       io.NopCloser(bytes.NewReader(c.ResBody)),
	}
	return req, res, nil
}

// KeyToPath converts key to path
// It is the responsibility of the user to pass path-safe keys
func KeyToPath(key string, n int) string {
	if n <= 0 {
		return key
	}
	var result strings.Builder
	l := len(key)
	for i, char := range key {
		if i > 0 && i%n == 0 && l-i > 0 {
			result.WriteRune(filepath.Separator)
		}
		result.WriteRune(char)
	}

	return result.String()
}

// PathToKey converts path to key
func PathToKey(path string) string {
	return strings.ReplaceAll(path, string(filepath.Separator), "")
}

// WriteCounter counts bytes written.
type WriteCounter struct {
	io.Writer
	Bytes uint64
}

// Write writes bytes.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n, err := wc.Writer.Write(p)
	if err != nil {
		return n, err
	}
	wc.Bytes += uint64(n)
	return n, err
}

// SetCacheResultHeader sets cache header.
func SetCacheResultHeader(res *http.Response, hit bool) {
	if hit {
		res.Header.Set(CacheResultHeader, CacheHit)
	} else {
		res.Header.Set(CacheResultHeader, CacheMiss)
	}
}
