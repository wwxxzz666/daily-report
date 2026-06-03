package monitor

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ActiveWindow 当前活动窗口信息
type ActiveWindow struct {
	ProcessName string
	WindowTitle string
}

var (
	user32                = windows.NewLazyDLL("user32.dll")
	kernel32              = windows.NewLazyDLL("kernel32.dll")
	psapi                 = windows.NewLazyDLL("psapi.dll")
	procGetForegroundWin  = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowThreadID = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess       = kernel32.NewProc("OpenProcess")
	procQueryFullProcName = kernel32.NewProc("QueryFullProcessImageNameW")
	procGetModuleFileName = psapi.NewProc("GetModuleFileNameExW")
	procCloseHandle       = kernel32.NewProc("CloseHandle")
)

const (
	PROCESS_QUERY_INFORMATION       = 0x0400
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	PROCESS_VM_READ                  = 0x0010
	PROCESS_ALL_ACCESS               = 0x1F0FFF
)

// GetActiveWindow 获取当前前台窗口的进程名和标题
func GetActiveWindow() (*ActiveWindow, error) {
	hwnd, _, _ := procGetForegroundWin.Call()
	if hwnd == 0 {
		return nil, fmt.Errorf("无法获取前台窗口")
	}

	// 获取窗口标题
	title := getWindowText(windows.HWND(hwnd))
	if title == "" {
		title = "(无标题)"
	}

	// 获取进程 ID
	var pid uint32
	procGetWindowThreadID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return &ActiveWindow{ProcessName: "unknown", WindowTitle: title}, nil
	}

	// 获取进程名
	procName := getProcessName(pid)

	return &ActiveWindow{
		ProcessName: procName,
		WindowTitle: title,
	}, nil
}

func getWindowText(hwnd windows.HWND) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		256,
	)
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func getProcessName(pid uint32) string {
	// 尝试用不同权限打开进程
	accessRights := []uintptr{
		PROCESS_QUERY_INFORMATION | PROCESS_VM_READ,
		PROCESS_QUERY_LIMITED_INFORMATION,
		PROCESS_ALL_ACCESS,
	}

	var handle uintptr
	for _, access := range accessRights {
		handle, _, _ = procOpenProcess.Call(access, 0, uintptr(pid))
		if handle != 0 {
			break
		}
	}

	if handle == 0 {
		return fmt.Sprintf("pid-%d", pid)
	}
	defer procCloseHandle.Call(handle)

	// 方法1: QueryFullProcessImageNameW (Windows Vista+)
	var size uint32 = 1024
	buf := make([]uint16, size)
	ret, _, _ := procQueryFullProcName.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if ret != 0 {
		exePath := syscall.UTF16ToString(buf)
		name := filepath.Base(exePath)
		return strings.TrimSuffix(strings.ToLower(name), ".exe")
	}

	// 方法2: GetModuleFileNameExW (备选方案)
	buf2 := make([]uint16, 1024)
	ret2, _, _ := procGetModuleFileName.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf2[0])),
		1024,
	)
	if ret2 > 0 {
		exePath := syscall.UTF16ToString(buf2)
		name := filepath.Base(exePath)
		return strings.TrimSuffix(strings.ToLower(name), ".exe")
	}

	return fmt.Sprintf("pid-%d", pid)
}
