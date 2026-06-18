package request

import (
	"net/http"
	"testing"
)

func TestClientIP_XForwardedFor(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

	got := ClientIP(r)
	if got != "203.0.113.50" {
		t.Errorf("ClientIP() = %q, want %q", got, "203.0.113.50")
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "192.168.1.1")

	got := ClientIP(r)
	if got != "192.168.1.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"

	got := ClientIP(r)
	if got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1"

	got := ClientIP(r)
	if got != "10.0.0.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "10.0.0.1")
	}
}

func TestClientIP_XForwardedForTakesPrecedence(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	r.Header.Set("X-Real-IP", "192.168.1.1")
	r.RemoteAddr = "10.0.0.1:12345"

	got := ClientIP(r)
	if got != "203.0.113.50" {
		t.Errorf("ClientIP() = %q, want %q", got, "203.0.113.50")
	}
}

func TestClientIP_InvalidXFF_FallbackToXRealIP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	r.Header.Set("X-Real-IP", "192.168.1.1")

	got := ClientIP(r)
	if got != "192.168.1.1" {
		t.Errorf("ClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

func TestClientIP_AllInvalid(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	r.Header.Set("X-Real-IP", "also-not-ip")
	r.RemoteAddr = "bad-addr"

	got := ClientIP(r)
	if got != "" {
		t.Errorf("ClientIP() = %q, want empty string", got)
	}
}

func TestClientIP_EmptyHeaders(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)

	got := ClientIP(r)
	if got != "" {
		t.Errorf("ClientIP() = %q, want empty string", got)
	}
}

func TestClientIP_IPv6RemoteAddr(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:8080"

	got := ClientIP(r)
	if got != "::1" {
		t.Errorf("ClientIP() = %q, want %q", got, "::1")
	}
}

func TestClientIP_IPv6XForwardedFor(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "2001:db8::1, 2001:db8::2")

	got := ClientIP(r)
	if got != "2001:db8::1" {
		t.Errorf("ClientIP() = %q, want %q", got, "2001:db8::1")
	}
}

func TestIPResolver_TrustedProxy(t *testing.T) {
	resolver, err := NewIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("NewIPResolver() error = %v", err)
	}

	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")

	got := resolver.Resolve(r)
	if got != "203.0.113.50" {
		t.Errorf("Resolve() = %q, want %q", got, "203.0.113.50")
	}
}

func TestIPResolver_UntrustedProxy(t *testing.T) {
	resolver, err := NewIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("NewIPResolver() error = %v", err)
	}

	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")

	got := resolver.Resolve(r)
	if got != "192.168.1.1" {
		t.Errorf("Resolve() = %q, want %q", got, "192.168.1.1")
	}
}

func TestIPResolver_LastNonTrustedIP(t *testing.T) {
	resolver, err := NewIPResolver([]string{"10.0.0.0/8", "172.16.0.0/12"})
	if err != nil {
		t.Fatalf("NewIPResolver() error = %v", err)
	}

	r, _ := http.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 172.16.0.1, 10.0.0.2")

	got := resolver.Resolve(r)
	if got != "203.0.113.50" {
		t.Errorf("Resolve() = %q, want %q", got, "203.0.113.50")
	}
}
