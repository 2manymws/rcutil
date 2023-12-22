package rcutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEncodeAndDecode(t *testing.T) {
	tests := []struct {
		req     *http.Request
		res     *http.Response
		wantReq *http.Request
		wantRes *http.Response
	}{
		{
			req:     &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/foo"}, Body: newBody("req")},
			res:     &http.Response{Body: newBody("")},
			wantReq: &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/foo"}, Body: newBody("req")},
			wantRes: &http.Response{Body: newBody("")},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := new(bytes.Buffer)
			if err := EncodeReqRes(tt.req, tt.res, buf); err != nil {
				t.Fatal(err)
			}
			gotReq, gotRes, err := DecodeReqRes(buf)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				gotReq.Body.Close()
				gotRes.Body.Close()
				tt.wantReq.Body.Close()
				tt.wantRes.Body.Close()
			})
			opts := []cmp.Option{
				cmpopts.IgnoreFields(http.Response{}, "Body"),
				cmpopts.IgnoreFields(http.Request{}, "Body"),
				cmpopts.IgnoreFields(http.Request{}, "ctx"),
			}
			if diff := cmp.Diff(tt.wantReq, gotReq, opts...); diff != "" {
				t.Error(diff)
			}
			gotb := readBody(gotReq.Body)
			wantb := readBody(tt.wantReq.Body)
			if diff := cmp.Diff(wantb, gotb); diff != "" {
				t.Error(diff)
			}

			{
				if diff := cmp.Diff(tt.wantRes, gotRes, opts...); diff != "" {
					t.Error(diff)
				}
				gotb := readBody(gotRes.Body)
				wantb := readBody(tt.wantRes.Body)
				if diff := cmp.Diff(wantb, gotb); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}

func TestKeyToPath(t *testing.T) {
	tests := []struct {
		key  string
		n    int
		want string
	}{
		{"ab", 2, "ab"},
		{"abcd", 2, "ab/cd"},
		{"abcde", 2, "ab/cd/e"},
		{"abcdef", 2, "ab/cd/ef"},
		{"abcdefg", 2, "ab/cd/ef/g"},
		{"abcdefg", 3, "abc/def/g"},
		{"abcdefg", 10, "abcdefg"},
		{"abcdefg", 0, "abcdefg"},
		{"abcdefg", -1, "abcdefg"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := KeyToPath(tt.key, tt.n)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func newBody(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func readBody(r io.ReadCloser) string {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(b)
}
