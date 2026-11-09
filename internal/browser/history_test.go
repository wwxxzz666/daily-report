package browser

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCopyToTempFile_Content 确认流式拷贝内容一致。
func TestCopyToTempFile_Content(t *testing.T) {
	src := filepath.Join(t.TempDir(), "History")
	// 造一个比内存页大的内容，确保走的是流式路径而不是偶然的小文件
	payload := bytes.Repeat([]byte("sqlite history payload\n"), 100000)
	if err := os.WriteFile(src, payload, 0600); err != nil {
		t.Fatal(err)
	}

	tmp, err := copyToTempFile(src, "dailyreport_test_*.db")
	if err != nil {
		t.Fatalf("拷贝失败: %v", err)
	}
	defer os.Remove(tmp)

	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("拷贝内容不一致: %d vs %d 字节", len(got), len(payload))
	}
}

// TestCopyToTempFile_Permission 临时文件必须是仅所有者可读写。
func TestCopyToTempFile_Permission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不使用 POSIX 权限位")
	}

	src := filepath.Join(t.TempDir(), "History")
	if err := os.WriteFile(src, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	tmp, err := copyToTempFile(src, "dailyreport_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp)

	info, err := os.Stat(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("临时文件权限应为 0600，实际 %o", perm)
	}
}

// TestCopyToTempFile_MissingSource 源文件不存在时不留下临时文件。
func TestCopyToTempFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	before, _ := os.ReadDir(os.TempDir())

	_, err := copyToTempFile(filepath.Join(dir, "nope"), "dailyreport_missing_*.db")
	if err == nil {
		t.Fatal("期望报错")
	}

	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before) {
		// 只断言没有新增 dailyreport_missing_* 前缀的文件
		for _, e := range after {
			if len(e.Name()) > 20 && e.Name()[:20] == "dailyreport_missing_" {
				t.Errorf("失败后残留临时文件: %s", e.Name())
			}
		}
	}
}

// TestCopyToTempFile_EmptyFile 空文件也能正常拷贝（0 字节不报错）。
func TestCopyToTempFile_EmptyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "History")
	if err := os.WriteFile(src, nil, 0600); err != nil {
		t.Fatal(err)
	}

	tmp, err := copyToTempFile(src, "dailyreport_empty_*.db")
	if err != nil {
		t.Fatalf("空文件不应报错: %v", err)
	}
	defer os.Remove(tmp)

	info, err := os.Stat(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("期望 0 字节，实际 %d", info.Size())
	}
}
