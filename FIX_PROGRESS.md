# 日报助手 Bug 修复进度

## 修复完成 ✅

### 1. 浏览器历史时间戳溢出

**文件**: `internal/browser/history.go:22-29`

**问题**: `time.Duration(webkitTimestamp) * time.Microsecond` 溢出，导致时间显示为 1441 年

**原因**: WebKit 时间戳（从 1601 年起的微秒数）约 1.3×10^16，`time.Duration(seconds) * time.Second` 会超出 int64 范围

**修复**: 使用 `time.Unix` 直接计算，避免 Duration 溢出
```go
func webKitToTime(webkitTimestamp int64) time.Time {
    sec := webkitTimestamp / 1_000_000
    usec := webkitTimestamp % 1_000_000
    const unixToWebKit int64 = 11644473600
    return time.Unix(sec-unixToWebKit, usec*1000)
}
```

### 2. 进程名显示为 pid-xxx

**文件**: `internal/monitor/window.go:81-132`

**问题**: `QueryFullProcessImageNameW` 调用失败，返回 "The data area passed to a system call is too small"

**修复**:
- 添加 `psapi.dll` 的 `GetModuleFileNameExW` 作为备选方案
- 使用多种权限尝试打开进程（`PROCESS_QUERY_INFORMATION|PROCESS_VM_READ`, `PROCESS_QUERY_LIMITED_INFORMATION`, `PROCESS_ALL_ACCESS`）
- 增大缓冲区到 1024 字符
- 先尝试 `QueryFullProcessImageNameW`，失败后回退到 `GetModuleFileNameExW`

## 验证结果

- 浏览器时间戳: 2025-2026 年 ✅
- 进程名解析: 全部正确（`windowsterminal`, `日报助手`, `douyin` 等）✅

## 构建命令

```powershell
$env:PATH = "D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go\bin;C:\Users\37119\go\bin;$env:PATH"
$env:GOROOT = "D:\anaconda3\pkgs\go-1.26.3-h5588389_0\go"
$env:GOPROXY = "https://goproxy.cn,direct"
cd D:\study\日报助手
wails build
```
