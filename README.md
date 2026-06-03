# 日报助手

自动记录工作时间的应用使用情况，结合 AI 生成日报/周报的 Windows 桌面工具。

## 功能特性

- **窗口活动监控** — 每 5 秒采样前台窗口，记录应用名和窗口标题
- **浏览器历史同步** — 自动导入 Chrome/Edge 浏览记录
- **AI 报告生成** — 接入 LLM 根据活动数据自动生成日报/周报（DOCX 格式）
- **定时调度** — 工作日下班自动生成日报，周五生成周报
- **隐私保护** — 敏感词过滤（密码、薪资等）、标题脱敏、URL 清洗、密码管理器自动跳过
- **系统托盘** — 关闭窗口后最小化到托盘继续运行

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.26, Wails v2, SQLite |
| 前端 | Vue 3, TypeScript, Vite |
| 报告 | docxgo (DOCX 输出) |
| 系统 | Windows API (user32, kernel32, psapi) |

## 快速开始

### 前置条件

- Windows 10/11
- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 Runtime（Win10/11 通常已自带）

### 构建运行

```bash
# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式（热重载）
wails dev

# 构建生产版本
wails build
```

构建产物在 `build/bin/日报助手.exe`。

### 配置 LLM

首次启动后，编辑 `%APPDATA%/日报助手/config.yaml`，填入你的 API Key：

```yaml
llm:
  provider: "deepseek"    # 推荐国内用户使用 DeepSeek
  api_key: "sk-你的key"
```

支持的 LLM 提供商：

| 提供商 | 说明 | 需要 API Key |
|--------|------|:------------:|
| deepseek | DeepSeek，性价比高 | 是 |
| openai | GPT-4o-mini | 是 |
| qwen | 通义千问（阿里云） | 是 |
| moonshot | Moonshot (Kimi) | 是 |
| zhipu | 智谱清言 (GLM-4) | 是 |
| ollama | 本地模型，数据不出本机 | 否 |
| lmstudio | LM Studio 本地模型 | 否 |

## 项目结构

```
日报助手/
├── main.go                 # 入口，Wails 应用配置
├── app.go                  # Wails binding 层，前端调用的 API
├── log.go                  # 日志配置（UTF-8 编码）
├── internal/
│   ├── config/             # 配置加载与保存
│   ├── monitor/            # 窗口采样器（Windows API 调用）
│   ├── browser/            # 浏览器历史同步（Chrome/Edge）
│   ├── report/             # 报告生成 + LLM 调用
│   ├── scheduler/          # 定时任务调度
│   ├── storage/            # SQLite 数据库操作
│   └── privacy/            # 隐私保护（脱敏、过滤）
├── frontend/
│   └── src/
│       ├── App.vue         # 主界面（Tab 导航）
│       └── views/
│           ├── Dashboard   # 仪表盘（状态、今日活动、快捷操作）
│           ├── Reports     # 报告历史列表
│           └── Settings    # 设置页（LLM、工作时间）
└── config.example.yaml     # 配置文件示例
```

## 配置说明

配置文件位置：`%APPDATA%/日报助手/config.yaml`

```yaml
work_time:
  start: "09:00"            # 工作开始时间
  end: "18:00"              # 下班时间（日报自动生成时间）
  weekdays: [1, 2, 3, 4, 5] # 工作日（1=周一, 7=周日）

sample_interval: "5s"       # 窗口采样间隔

monitored_apps: ["*"]       # 监控所有应用
ignored_apps:               # 忽略列表（优先级高于 monitored_apps）
  - "explorer"
  - "Taskmgr"

sensitive_words:            # 窗口标题含这些词则跳过记录
  - "密码"
  - "password"

browser:
  enabled: true
  browsers: ["chrome", "edge"]

report:
  output_dir: "%APPDATA%/日报助手/reports"
  weekly_day: 5             # 周五生成周报
```

## 数据存储

| 文件 | 说明 |
|------|------|
| `%APPDATA%/日报助手/data.db` | SQLite 数据库 |
| `%APPDATA%/日报助手/config.yaml` | 配置文件 |
| `%APPDATA%/日报助手/app.log` | 运行日志 |
| `%APPDATA%/日报助手/reports/` | 生成的 DOCX 报告 |

## 许可证

MIT
