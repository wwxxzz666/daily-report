package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32DLL      = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32DLL.NewProc("CreateMutexW")
	procCloseHandle  = kernel32DLL.NewProc("CloseHandle")
	procOpenProcess  = kernel32DLL.NewProc("OpenProcess")
	procGetExitCode  = kernel32DLL.NewProc("GetExitCodeProcess")
)

var (
	mutexHandle uintptr
	lockFile    *os.File
)

// acquireLock 获取单实例锁（Mutex + 文件锁双重保障）
func acquireLock() bool {
	// 方法1: Windows Mutex
	name, _ := syscall.UTF16PtrFromString("DailyReportAssistant_SingleInstance")
	handle, _, lastErr := syscall.Syscall(procCreateMutexW.Addr(), 3, 0, 0, uintptr(unsafe.Pointer(name)))
	if handle != 0 && lastErr != 183 {
		mutexHandle = handle
		os.Remove(filepath.Join(os.TempDir(), "dailyreport.lock"))
		return true
	}
	if handle != 0 {
		syscall.Syscall(procCloseHandle.Addr(), 1, handle, 0, 0)
	}

	// 方法2: 文件锁（检查 PID 是否存活）
	lockPath := filepath.Join(os.TempDir(), "dailyreport.lock")
	if data, err := os.ReadFile(lockPath); err == nil {
		var pid int
		fmt.Sscanf(string(data), "%d", &pid)
		if pid > 0 && isProcessAlive(pid) {
			return false
		}
		os.Remove(lockPath)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	fmt.Fprintf(f, "%d", os.Getpid())
	lockFile = f
	return true
}

// releaseLock 释放单实例锁
func releaseLock() {
	if mutexHandle != 0 {
		syscall.Syscall(procCloseHandle.Addr(), 1, mutexHandle, 0, 0)
		mutexHandle = 0
	}
	if lockFile != nil {
		lockFile.Close()
		os.Remove(lockFile.Name())
		lockFile = nil
	}
}

// isProcessAlive 检查进程是否还在运行
func isProcessAlive(pid int) bool {
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	handle, _, _ := syscall.Syscall(procOpenProcess.Addr(), 3,
		uintptr(PROCESS_QUERY_LIMITED_INFORMATION), 0, uintptr(pid))
	if handle == 0 {
		return false
	}
	defer syscall.Syscall(procCloseHandle.Addr(), 1, handle, 0, 0)

	var exitCode uint32
	ret, _, _ := syscall.Syscall(procGetExitCode.Addr(), 2,
		handle, uintptr(unsafe.Pointer(&exitCode)), 0)
	return ret != 0 && exitCode == 259 // STILL_ACTIVE
}
