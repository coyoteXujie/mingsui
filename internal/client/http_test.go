package client

import "testing"

func TestNormalizeHostPort(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		defaultPort string
		want        string
	}{
		{name: "域名没有端口", host: "example.com", defaultPort: "80", want: "example.com:80"},
		{name: "域名已有端口", host: "example.com:443", defaultPort: "80", want: "example.com:443"},
		{name: "IPv4 没有端口", host: "127.0.0.1", defaultPort: "8080", want: "127.0.0.1:8080"},
		{name: "IPv6 没有端口", host: "2001:db8::1", defaultPort: "443", want: "[2001:db8::1]:443"},
		{name: "带方括号 IPv6 没有端口", host: "[2001:db8::1]", defaultPort: "443", want: "[2001:db8::1]:443"},
		{name: "带方括号 IPv6 已有端口", host: "[2001:db8::1]:443", defaultPort: "80", want: "[2001:db8::1]:443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHostPort(tt.host, tt.defaultPort)
			if err != nil {
				t.Fatalf("normalizeHostPort() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeHostPort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeHostPortRejectsEmptyHost(t *testing.T) {
	if _, err := normalizeHostPort("", "80"); err == nil {
		t.Fatal("normalizeHostPort() error = nil, want empty host error")
	}
}

func TestNormalizeHostPortRejectsEmptyPort(t *testing.T) {
	if _, err := normalizeHostPort("example.com:", "80"); err == nil {
		t.Fatal("normalizeHostPort() error = nil, want empty port error")
	}
}
