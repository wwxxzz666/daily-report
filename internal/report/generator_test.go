package report

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncate_CJK 覆盖中文截断。
// 旧实现用 len(s) / s[:maxLen-3] 按字节切，中文 3 字节/字，
// 切点很容易落在字符中间，产出非法 UTF-8，写进 DOCX 就是乱码。
func TestTruncate_CJK(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLen    int
		wantRunes int
	}{
		{"中文短串不截断", "写代码", 80, 3},
		{"中文恰好等于上限", strings.Repeat("中", 20), 20, 20},
		{"中文超长截断", strings.Repeat("中", 40), 20, 20},
		{"混合中英超长截断", strings.Repeat("代码code", 20), 11, 11},
		{"emoji 超长截断", strings.Repeat("🎉", 40), 10, 10},
		{"maxLen 过小返回空串", "中文", 3, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)

			if !utf8.ValidString(got) {
				t.Fatalf("truncate 产出了非法 UTF-8: %q", got)
			}
			if n := utf8.RuneCountInString(got); n != tt.wantRunes {
				t.Errorf("truncate(%d 字符的输入, %d) 得到 %d 个字符 (%q)，期望 %d",
					utf8.RuneCountInString(tt.input), tt.maxLen, n, got, tt.wantRunes)
			}
			if tt.wantRunes > 0 && tt.maxLen >= 4 && utf8.RuneCountInString(tt.input) > tt.maxLen {
				if !strings.HasSuffix(got, "...") {
					t.Errorf("被截断的字符串应以 '...' 结尾，得到 %q", got)
				}
			}
			// 前缀必须是原文的合法前缀（不能凭空多出字符）
			if tt.wantRunes > 0 {
				prefix := strings.TrimSuffix(got, "...")
				if !strings.HasPrefix(tt.input, prefix) {
					t.Errorf("截断结果 %q 不是原文的前缀", got)
				}
			}
		})
	}
}

// TestSummarizeTitles_CJK 确认长中文标题汇总后仍是合法 UTF-8。
func TestSummarizeTitles_CJK(t *testing.T) {
	g := &Generator{}

	titles := []string{
		strings.Repeat("这是一个很长很长的中文窗口标题", 4),
		strings.Repeat("另一个同样冗长的中文标题内容", 4),
	}
	got := g.summarizeTitles(titles, 3)

	if !utf8.ValidString(got) {
		t.Fatalf("summarizeTitles 产出了非法 UTF-8: %q", got)
	}
	if n := utf8.RuneCountInString(got); n > 80 {
		t.Errorf("汇总结果 %d 个字符，超过 80 上限: %q", n, got)
	}
}

// TestSummarizeTitles_ByteVsRune 记录一个行为变化：
// 30 个中文字符 = 90 字节，旧实现会误判为"超长"并截断，
// 新实现按字符数判断，30 ≤ 80 应原样返回。
func TestSummarizeTitles_ByteVsRune(t *testing.T) {
	g := &Generator{}
	title := strings.Repeat("中", 30) // 90 字节, 30 字符

	got := g.summarizeTitles([]string{title}, 3)

	if got != title {
		t.Errorf("30 个中文字符不应被截断\n得到: %q\n期望: %q", got, title)
	}
}
