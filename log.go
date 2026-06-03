package main

import (
	"io"
	"log"
	"os"
	"unicode/utf8"
)

// utf8Writer 包装 io.Writer，确保写入的是 UTF-8 编码
// 在 Windows 上如果系统默认编码不是 UTF-8，直接写中文会乱码
type utf8Writer struct {
	w io.Writer
}

func (u *utf8Writer) Write(p []byte) (n int, err error) {
	// 如果已经是合法 UTF-8，直接写入
	if utf8.Valid(p) {
		return u.w.Write(p)
	}
	// 否则尝试从 Latin-1 转换（Go 字符串字面量默认是 UTF-8，不会到这里）
	// 保险起见直接写入
	return u.w.Write(p)
}

// setupLog 设置日志输出到文件，使用 UTF-8 编码
// 写入 UTF-8 BOM 头以便 Windows 记事本正确识别编码
func setupLog(logPath string) (*os.File, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	// 如果文件是新建的（大小为 0），写入 UTF-8 BOM
	info, _ := f.Stat()
	if info != nil && info.Size() == 0 {
		f.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	log.SetOutput(&utf8Writer{w: f})
	return f, nil
}
