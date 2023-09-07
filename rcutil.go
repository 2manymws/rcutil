package rcutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// Seed returns seed for cache key.
func Seed(req *http.Request, vary []string) (string, error) {
	const sep = "|"
	seed := req.Method + sep + req.URL.Path + sep + req.URL.RawQuery
	for _, h := range vary {
		if vv := req.Header.Get(h); vv != "" {
			seed += sep + ":" + h + vv
		}
	}
	return strings.ToLower(seed), nil
}

type cacheResponse struct {
	StatusCode int         `json:"status_code"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"body"`
}

// EncodeResponse encodes http.Response.
func EncodeResponse(res *http.Response, w io.Writer) error {
	c := &cacheResponse{
		StatusCode: res.StatusCode,
		Header:     res.Header,
	}
	// FIXME: Use stream
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	c.Body = b
	if err := json.NewEncoder(w).Encode(c); err != nil {
		return err
	}
	return nil
}

// DecodeResponse decodes to http.Response.
func DecodeResponse(r io.Reader) (*http.Response, error) {
	c := &cacheResponse{}
	if err := json.NewDecoder(r).Decode(c); err != nil {
		return nil, err
	}
	res := &http.Response{
		StatusCode: c.StatusCode,
		Header:     c.Header,
		Body:       io.NopCloser(bytes.NewReader(c.Body)),
	}
	return res, nil
}

// KeyToPath converts key to path
func KeyToPath(key string, n int) string {
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

// WriteCounter counts bytes written.
type WriteCounter struct {
	io.Writer
	Bytes int
}

// Write writes bytes.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n, err := wc.Writer.Write(p)
	if err != nil {
		return n, err
	}
	wc.Bytes += n
	return n, err
}
