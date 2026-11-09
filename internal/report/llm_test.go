package report

import "testing"

// TestIsLocalhost 覆盖 H3（强制 HTTPS）的白名单判断。
// 关键点：不能用字符串包含判断，否则 evil.example 只要带上 ?x=localhost
// 就能让明文 HTTP 端点通过校验，把 API Key 暴露在链路上。
func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     bool
	}{
		{"localhost", "http://localhost:11434/v1/chat/completions", true},
		{"localhost 大写", "http://LOCALHOST:11434/v1/chat/completions", true},
		{"ipv4 回环", "http://127.0.0.1:1234/v1/chat/completions", true},
		{"ipv6 回环", "http://[::1]:8080/v1", true},

		{"远程 https", "https://api.deepseek.com/v1/chat/completions", false},
		{"远程明文 http", "http://evil.example/v1/chat/completions", false},

		// 以下三个是旧实现（strings.Contains）会误判为本地、从而绕过 HTTPS 强制的输入
		{"query 里带 localhost", "http://evil.example/v1?host=localhost", false},
		{"path 里带 localhost", "http://evil.example/localhost/v1", false},
		{"query 里带 127.0.0.1", "http://evil.example/v1?ip=127.0.0.1", false},

		{"空字符串", "", false},
		{"非法 URL", "://bad", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalhost(tt.endpoint); got != tt.want {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}
