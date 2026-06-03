package privacy

import (
	"net/url"
	"os"
	"regexp"
	"strings"
)

var (
	// 匹配 Windows 路径：C:\Users\xxx\、D:\project\
	reFilePath = regexp.MustCompile(`[A-Za-z]:\\[^\s]*\\`)
	// 匹配 Unix 路径：/home/xxx/
	reUnixPath = regexp.MustCompile(`/home/[^\s]*/`)
	// 匹配用户目录路径中的用户名
	reUserPath = regexp.MustCompile(`(?i)(?:Users|用户)[\\/][^\\/]+`)
	// 匹配邮箱格式
	reEmail = regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`)
	// 匹配连续数字（可能是手机号、身份证等）
	reLongNumber = regexp.MustCompile(`\b\d{8,}\b`)
)

// SanitizeTitle 对窗口标题进行脱敏处理
// 保留有意义的应用信息，去除路径、用户名等敏感内容
func SanitizeTitle(title string) string {
	if title == "" || title == "(无标题)" {
		return title
	}

	result := title

	// 去除文件路径
	result = reFilePath.ReplaceAllString(result, "[路径]")

	// 去除 Unix 路径
	result = reUnixPath.ReplaceAllString(result, "[路径]")

	// 去除用户名
	result = reUserPath.ReplaceAllString(result, "[用户]/")

	// 去除邮箱
	result = reEmail.ReplaceAllString(result, "[邮箱]")

	// 去除长数字串（手机号、身份证等）
	result = reLongNumber.ReplaceAllString(result, "[数字]")

	return result
}

// SanitizeURL 清除 URL 中的查询参数和 fragment
// 只保留 scheme://host/path，去掉可能包含 token、搜索词等信息
func SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// 解析失败，做基础清理
		if idx := strings.Index(rawURL, "?"); idx > 0 {
			return rawURL[:idx]
		}
		return rawURL
	}

	// 清除 query 和 fragment
	u.RawQuery = ""
	u.Fragment = ""

	return u.String()
}

// IsSensitiveTitle 检查标题是否包含敏感关键词
func IsSensitiveTitle(title string, sensitiveWords []string) bool {
	if len(sensitiveWords) == 0 {
		return false
	}

	lowerTitle := strings.ToLower(title)
	for _, word := range sensitiveWords {
		if word == "" {
			continue
		}
		if strings.Contains(lowerTitle, strings.ToLower(word)) {
			return true
		}
	}

	return false
}

// ShouldIgnoreProcess 检查进程是否应该被完全忽略（隐私角度）
func ShouldIgnoreProcess(processName string) bool {
	// 密码管理器
	privacySensitive := []string{
		"1password", "lastpass", "bitwarden", "keepass",
		"dashlane", "roboform",
	}

	lower := strings.ToLower(processName)
	for _, p := range privacySensitive {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// GetCurrentUsername 获取当前系统用户名（用于脱敏）
func GetCurrentUsername() string {
	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}
	return username
}
