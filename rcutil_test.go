package rcutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSeed(t *testing.T) {
	tests := []struct {
		req     *http.Request
		vary    []string
		want    string
		wantErr bool
	}{
		{nil, nil, "", true},
		{
			&http.Request{
				Method: http.MethodGet,
				Host:   "example.com",
			},
			nil,
			"",
			true,
		},
		{
			&http.Request{
				Method: http.MethodGet,
				Host:   "example.com",
				URL: &url.URL{
					Host: "example.com",
					Path: "/foo",
				},
			},
			nil,
			"get|example.com|/foo|",
			false,
		},
		{
			&http.Request{
				Method: http.MethodPost,
				Host:   "example.com",
				URL: &url.URL{
					Host: "127.0.0.1",
					Path: "/foo",
				},
			},
			nil,
			"post|example.com|/foo|",
			false,
		},
		{
			&http.Request{
				Method: http.MethodGet,
				Host:   "example.com",
				URL: &url.URL{
					Host:     "127.0.0.1",
					Path:     "/foo",
					RawQuery: "a=1&b=2",
				},
			},
			nil,
			"get|example.com|/foo|a=1&b=2",
			false,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := Seed(tt.req, tt.vary)
			if (err != nil) != tt.wantErr {
				t.Errorf("got Seed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got Seed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeAndDecode(t *testing.T) {
	image, err := os.ReadFile("testdata/2manymws.png")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		req     *http.Request
		res     *http.Response
		wantReq *http.Request
		wantRes *http.Response
	}{
		{
			req: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, "https://example.com/foo", newBody([]byte("req")))
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Content-Type", "text/plain")
				return req
			}(),
			res: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
				StatusCode: http.StatusOK,
				Header: http.Header{
					"X-Cache":        []string{"MISS"},
					"Content-Length": []string{"0"},
				},
				Body:          newBody(nil),
				ContentLength: 0,
			},
			wantReq: &http.Request{
				Method: http.MethodGet,
				Header: http.Header{
					"Content-Type": []string{"text/plain"},
					"User-Agent":   {"Go-http-client/1.1"},
				},
				Host:       "example.com",
				URL:        &url.URL{Path: "/foo"},
				Body:       newBody([]byte("req")),
				RequestURI: "/foo",
			},
			wantRes: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
				StatusCode: http.StatusOK,
				Header: http.Header{
					"X-Cache":        []string{"MISS"},
					"Content-Length": []string{"0"},
				},
				Body:          newBody(nil),
				ContentLength: 0,
			},
		},
		{
			req: &http.Request{
				Method: http.MethodGet,
				Host:   "example.com",
				URL:    &url.URL{Path: "/foo"},
				Body:   newBody([]byte("req")),
			},
			res: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Length": []string{fmt.Sprintf("%d", len(image))},
				},
				Body:          newBody(image),
				ContentLength: int64(len(image)),
			},
			wantReq: &http.Request{
				Method: http.MethodGet,
				Host:   "example.com",
				URL:    &url.URL{Path: "/foo"},
				Header: http.Header{
					"User-Agent": {"Go-http-client/1.1"},
				},
				Body:       newBody([]byte("req")),
				RequestURI: "/foo",
			},
			wantRes: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Length": []string{fmt.Sprintf("%d", len(image))},
				},
				Body:          newBody(image),
				ContentLength: int64(len(image)),
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			reqb := new(bytes.Buffer)
			resb := new(bytes.Buffer)
			if err := EncodeReq(tt.req, reqb); err != nil {
				t.Fatal(err)
			}
			if err := EncodeRes(tt.res, resb); err != nil {
				t.Fatal(err)
			}
			gotReq, err := DecodeReq(reqb)
			if err != nil {
				t.Fatal(err)
			}
			gotRes, err := DecodeRes(resb)
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
				cmpopts.IgnoreFields(http.Request{}, "Body", "TransferEncoding", "Proto", "ProtoMajor", "ProtoMinor", "ContentLength", "ctx", "pat", "matches", "otherValues"),
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

func TestDeprecatedEncodeAndDecode(t *testing.T) {
	image, err := os.ReadFile("testdata/2manymws.png")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		req     *http.Request
		res     *http.Response
		wantReq *http.Request
		wantRes *http.Response
	}{
		{
			req:     &http.Request{Method: http.MethodGet, Header: http.Header{"Content-Type": []string{"text/plain"}}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))},
			res:     &http.Response{StatusCode: http.StatusOK, Header: http.Header{"X-Cache": []string{"MISS"}}, Body: newBody(nil)},
			wantReq: &http.Request{Method: http.MethodGet, Header: http.Header{"Content-Type": []string{"text/plain"}}, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))},
			wantRes: &http.Response{Status: http.StatusText(http.StatusOK), StatusCode: http.StatusOK, Header: http.Header{"X-Cache": []string{"MISS"}}, Body: newBody(nil)},
		},
		{
			req:     &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))},
			res:     &http.Response{Body: newBody(image)},
			wantReq: &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/foo"}, Body: newBody([]byte("req"))},
			wantRes: &http.Response{Body: newBody(image)},
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
				cmpopts.IgnoreFields(http.Request{}, "Body", "ctx", "pat", "matches", "otherValues"),
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

func newBody(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(b))
}

func readBody(r io.ReadCloser) string {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err) //nostyle:dontpanic
	}
	return string(b)
}
