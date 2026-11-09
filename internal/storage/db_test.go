package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInit_CreatesParentDir 确认 Init 会创建任意文件名的父目录。
func TestInit_CreatesParentDir(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "a", "b", "custom.sqlite3")

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init 失败: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Errorf("父目录未被创建: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("数据库文件未创建: %v", err)
	}
}

// TestInit_ShortPath 回归测试。
// 旧实现 dir := dbPath[:len(dbPath)-len("/data.db")]，
// 路径短于 "/data.db" 时会得到负索引直接 panic。
func TestInit_ShortPath(t *testing.T) {
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Init 在短路径上 panic 了: %v", r)
		}
	}()

	db, err := Init("db")
	if err != nil {
		t.Fatalf("Init(\"db\") 失败: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat("db"); err != nil {
		t.Errorf("数据库文件未创建: %v", err)
	}
}

// TestInit_RelativePathNoDir 确认无目录部分的相对路径不会报错。
func TestInit_RelativePathNoDir(t *testing.T) {
	root := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	db, err := Init("data.db")
	if err != nil {
		t.Fatalf("Init 失败: %v", err)
	}
	defer db.Close()
}
