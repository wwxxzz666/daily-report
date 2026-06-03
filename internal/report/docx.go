package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docx "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"
)

// SaveDocx 将 Markdown 格式的文本写入 .docx 文件
func SaveDocx(content, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	builder := docx.NewDocumentBuilder(
		docx.WithDefaultFont("微软雅黑"),
		docx.WithDefaultFontSize(22), // 11pt
	)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")

		if trimmed == "" {
			// 空行 → 空段落
			builder.AddParagraph().End()
			continue
		}

		// 标题处理
		if strings.HasPrefix(trimmed, "### ") {
			builder.AddParagraph().
				Text(strings.TrimPrefix(trimmed, "### ")).
				Bold().
				FontSize(24).
				End()
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			builder.AddParagraph().
				Text(strings.TrimPrefix(trimmed, "## ")).
				Bold().
				FontSize(28).
				End()
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			builder.AddParagraph().
				Text(strings.TrimPrefix(trimmed, "# ")).
				Bold().
				FontSize(32).
				Alignment(domain.AlignmentCenter).
				End()
			continue
		}

		// 表格分隔行
		if isTableSeparator(trimmed) {
			continue
		}

		// 表格行
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			cells := parseTableRow(trimmed)
			para := builder.AddParagraph()
			for i, cell := range cells {
				if i > 0 {
					para.Text(" | ").FontSize(20)
				}
				para.Text(strings.TrimSpace(cell)).FontSize(20)
			}
			para.End()
			continue
		}

		// 列表项
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := strings.TrimPrefix(trimmed, "- ")
			text = strings.TrimPrefix(text, "* ")
			builder.AddParagraph().
				Text("• " + text).
				End()
			continue
		}

		// 数字列表
		if isNumberedList(trimmed) {
			builder.AddParagraph().
				Text(trimmed).
				End()
			continue
		}

		// 普通段落（处理行内加粗 **text**）
		para := builder.AddParagraph()
		addFormattedText(para, trimmed)
		para.End()
	}

	doc, err := builder.Build()
	if err != nil {
		return fmt.Errorf("构建文档失败: %w", err)
	}

	if err := doc.SaveAs(filePath); err != nil {
		return fmt.Errorf("保存文档失败: %w", err)
	}

	return nil
}

func isTableSeparator(s string) bool {
	trimmed := strings.TrimSpace(s)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	// 去掉中间的分隔 | 后检查剩余字符
	for _, c := range trimmed {
		if c != '-' && c != ' ' && c != ':' && c != '|' {
			return false
		}
	}
	return len(trimmed) > 0
}

func parseTableRow(s string) []string {
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	return strings.Split(s, "|")
}

func isNumberedList(s string) bool {
	if len(s) < 3 {
		return false
	}
	for i, c := range s {
		if c == '.' && i > 0 {
			return s[i+1] == ' '
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return false
}

// addFormattedText 处理行内 Markdown 格式（**bold**）
func addFormattedText(para *docx.ParagraphBuilder, text string) {
	parts := strings.Split(text, "**")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i%2 == 1 {
			// 奇数索引 = 加粗
			para.Text(part).Bold()
		} else {
			para.Text(part)
		}
	}
}
