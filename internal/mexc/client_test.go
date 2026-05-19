package mexc

import (
	"context"
	goerrs "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func recomputeForTest(t *testing.T, secret, payload string) string {
	t.Helper()
	c := &Client{APIKey: "key", APISecret: secret}
	return c.signRaw(payload)
}

func TestSignReproduces(t *testing.T) {
	c := &Client{APIKey: "key", APISecret: "secret"}
	got := c.sign("1700000000", "param1=a&param2=b")
	want := recomputeForTest(t, "secret", "key"+"1700000000"+"param1=a&param2=b")
	if got != want {
		t.Errorf("sign mismatch:\n got  %s\n want %s", got, want)
	}
}

func TestAuthHeadersSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ApiKey") != "k" {
			t.Errorf("ApiKey header missing/wrong: %q", r.Header.Get("ApiKey"))
		}
		if r.Header.Get("Request-Time") == "" {
			t.Errorf("Request-Time header missing")
		}
		if r.Header.Get("Signature") == "" {
			t.Errorf("Signature header missing")
		}
		io.WriteString(w, `{"success":true,"code":0,"data":null}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "k", "s")
	var out Envelope[any]
	if err := c.do(context.Background(), "GET", "/api/v1/contract/ping", nil, nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestRetriesOn5xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(502)
			return
		}
		io.WriteString(w, `{"success":true,"code":0,"data":null}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "k", "s")
	c.sleep = func(_ int) {}
	var out Envelope[any]
	if err := c.do(context.Background(), "GET", "/api/v1/contract/ping", nil, nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if calls != 3 {
		t.Errorf("retries: got %d calls, want 3", calls)
	}
}

func TestGivesUpAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	}))
	defer srv.Close()
	c := New(srv.URL, "k", "s")
	c.sleep = func(_ int) {}
	var out Envelope[any]
	err := c.do(context.Background(), "GET", "/api/v1/contract/ping", nil, nil, &out)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "after retries") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestAuthErrorOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, `{"success":false,"code":401,"message":"unauthorized"}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "k", "s")
	c.sleep = func(_ int) {}
	var out Envelope[any]
	err := c.do(context.Background(), "GET", "/api/v1/private/account/assets", nil, nil, &out)
	if !goerrs.Is(err, ErrAuth) {
		t.Fatalf("got %v want ErrAuth", err)
	}
}

func TestRespectsRetryAfter(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(429)
			return
		}
		io.WriteString(w, `{"success":true,"code":0,"data":null}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "k", "s")
	slept := []int{}
	c.sleep = func(n int) { slept = append(slept, n) }
	var out Envelope[any]
	if err := c.do(context.Background(), "GET", "/api/v1/contract/ping", nil, nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if len(slept) != 1 || slept[0] != 2 {
		t.Errorf("slept = %v, want [2]", slept)
	}
}
