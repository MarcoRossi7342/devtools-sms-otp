package otpflow

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestCodeRetriesRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sms/otp" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Idempotency-Key") != "login-42-send" {
			t.Fatal("missing auth or idempotency header")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `{"to":"+15551234567"}` {
			t.Fatalf("unexpected body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"rate_limited","hint":"retry later"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"metadata":{}}`))
	}))
	defer server.Close()

	client, err := NewSMSClient("test-key")
	if err != nil {
		t.Fatal(err)
	}
	client.baseURL = server.URL
	client.sleep = func(context.Context, time.Duration) error { return nil }
	if err := client.RequestCode(context.Background(), "+15551234567", "login-42-send"); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("got %d attempts, want 2", attempts)
	}
}
