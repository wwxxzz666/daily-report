package browser

import (
	"fmt"
	"os"
	"path/filepath"
)

// BrowserPath 返回浏览器 History 数据库路径
func BrowserPath(browser string) ([]string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, fmt.Errorf("LOCALAPPDATA 环境变量未设置")
	}

	switch browser {
	case "chrome":
		base := filepath.Join(localAppData, "Google", "Chrome", "User Data")
		return findProfiles(base)
	case "edge":
		base := filepath.Join(localAppData, "Microsoft", "Edge", "User Data")
		return findProfiles(base)
	default:
		return nil, fmt.Errorf("不支持的浏览器: %s", browser)
	}
}

// findProfiles 查找浏览器所有 profile 的 History 文件
func findProfiles(baseDir string) ([]string, error) {
	var paths []string

	// Default profile
	defaultPath := filepath.Join(baseDir, "Default", "History")
	if _, err := os.Stat(defaultPath); err == nil {
		paths = append(paths, defaultPath)
	}

	// Profile 1, Profile 2, ...
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if len(paths) > 0 {
			return paths, nil
		}
		return nil, fmt.Errorf("读取浏览器目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && (len(entry.Name()) >= 7 && entry.Name()[:7] == "Profile") {
			p := filepath.Join(baseDir, entry.Name(), "History")
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("未找到浏览器历史文件")
	}

	return paths, nil
}
