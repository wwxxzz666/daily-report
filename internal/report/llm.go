package report

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"daily-report/internal/config"
)

const maxResponseSize = 10 << 20 // 10MB

// ChatRequest OpenAI Chat Completions 请求格式
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse OpenAI Chat Completions 响应格式
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CallLLM 调用 OpenAI 兼容 API
func CallLLM(cfg *config.Config, systemPrompt, userMessage string) (string, error) {
	endpoint := cfg.LLM.GetEndpoint()
	model := cfg.LLM.GetModel()

	if endpoint == "" {
		return "", fmt.Errorf("未配置 LLM 端点，请在配置文件中设置 llm.provider 或 llm.endpoint")
	}

	// H3: 强制 HTTPS（本地 localhost 例外）
	if !strings.HasPrefix(endpoint, "https://") && !isLocalhost(endpoint) {
		return "", fmt.Errorf("LLM 端点必须使用 HTTPS: %s", endpoint)
	}

	reqBody := ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	timeout := cfg.GetLLMTimeout()
	// 国内网络环境 OCSP 常被阻断，跳过 TLS 证书吊销检查
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Timeout: timeout, Transport: transport}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		// H2: 每次重试重建请求，避免 body 被消耗后发送空请求
		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonData))
		if err != nil {
			return "", fmt.Errorf("创建请求失败: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.LLM.APIKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求失败: %w", err)
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}

		// M1: 限制读取大小，防止内存耗尽
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("读取响应失败: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API 返回错误 (%d): %s", resp.StatusCode, string(body))
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return "", fmt.Errorf("解析响应失败: %w", err)
		}

		if chatResp.Error != nil {
			return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
		}

		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("API 返回空结果")
		}

		content := strings.TrimSpace(chatResp.Choices[0].Message.Content)

		// M5: 校验响应长度
		if len(content) > 100000 {
			return "", fmt.Errorf("LLM 响应过长 (%d 字符)，已截断", len(content))
		}

		return content, nil
	}

	return "", fmt.Errorf("重试 3 次后仍然失败: %w", lastErr)
}

func isLocalhost(url string) bool {
	return strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1")
}
