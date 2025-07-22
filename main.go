//go:generate goversioninfo

package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const vanguardPath = `C:\Program Files\Riot Vanguard`
const tokenElevation = 20 // TOKEN_INFORMATION_CLASS: TokenElevation

func isAdmin() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	var elevation struct {
		TokenIsElevated uint32
	}
	var outLen uint32
	err := windows.GetTokenInformation(token, tokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &outLen)
	return err == nil && elevation.TokenIsElevated != 0
}

func elevateIfNeeded() {
	if isAdmin() {
		return
	}

	fmt.Println("Requesting administrative privileges...")

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	argsPtr, _ := syscall.UTF16PtrFromString("")
	cwdPtr, _ := syscall.UTF16PtrFromString("")
	showCmd := 1 // SW_NORMAL

	r, _, err := shellExecute.Call(0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		uintptr(showCmd),
	)

	if r <= 32 {
		log.Fatalf("Failed to elevate process: %v", err)
	}

	os.Exit(0)
}

func checkVanguardInstalled() bool {
	info, err := os.Stat(vanguardPath)
	return err == nil && info.IsDir()
}

func prompt(msg string) bool {
	fmt.Print(msg + " [Y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToUpper(input))
	return input == "Y"
}

func controlService(name string, action string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()

	switch action {
	case "disable":
		return s.UpdateConfig(mgr.Config{StartType: mgr.StartDisabled})
	case "demand":
		return s.UpdateConfig(mgr.Config{StartType: mgr.StartManual})
	case "system", "automatic":
		return s.UpdateConfig(mgr.Config{StartType: mgr.StartAutomatic})
	case "stop":
		_, err := s.Control(svc.Stop)
		return err
	default:
		return errors.New("unknown service action: " + action)
	}
}

func killProcessByName(name string) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err = windows.Process32First(snapshot, &entry); err != nil {
		return err
	}

	for {
		exeName := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(exeName, name) {
			handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, entry.ProcessID)
			if err == nil {
				defer windows.CloseHandle(handle)
				windows.TerminateProcess(handle, 1)
			}
			return nil
		}
		if err = windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return nil
}

func disableVanguard() {
	fmt.Println("Disabling Vanguard...")

	_ = controlService("vgc", "disable")
	_ = controlService("vgk", "disable")
	_ = controlService("vgc", "stop")
	_ = controlService("vgk", "stop")

	_ = killProcessByName("vgtray.exe")

	files, _ := os.ReadDir(vanguardPath)
	for _, file := range files {
		oldPath := filepath.Join(vanguardPath, file.Name())
		newPath := oldPath + ".bak"
		os.Rename(oldPath, newPath)
	}

	os.RemoveAll(filepath.Join(vanguardPath, "Logs"))
}

func enableVanguard() {
	fmt.Println("Enabling Vanguard...")

	files, _ := os.ReadDir(vanguardPath)
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".bak") {
			oldPath := filepath.Join(vanguardPath, name)
			newPath := filepath.Join(vanguardPath, strings.TrimSuffix(name, ".bak"))
			os.Rename(oldPath, newPath)
		}
	}

	_ = controlService("vgc", "demand")
	_ = controlService("vgk", "system")

	if prompt("Changes require a system restart. Do you want to restart now?") {
		shutdownWindows()
	}
}

func shutdownWindows() {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")

	exitWindowsEx := user32.NewProc("ExitWindowsEx")
	adjustTokenPrivileges := advapi32.NewProc("AdjustTokenPrivileges")
	openProcessToken := advapi32.NewProc("OpenProcessToken")
	lookupPrivilegeValue := advapi32.NewProc("LookupPrivilegeValueW")

	const SE_SHUTDOWN_NAME = "SeShutdownPrivilege"
	const TOKEN_ADJUST_PRIVILEGES = 0x20
	const TOKEN_QUERY = 0x8
	const SE_PRIVILEGE_ENABLED = 0x2
	const EWX_REBOOT = 0x2

	var hToken syscall.Handle
	handle, _ := syscall.GetCurrentProcess()
	r1, _, err := openProcessToken.Call(uintptr(handle), TOKEN_ADJUST_PRIVILEGES|TOKEN_QUERY, uintptr(unsafe.Pointer(&hToken)))
	if r1 == 0 {
		log.Printf("Failed to open process token: %v", err)
		return
	}
	defer syscall.CloseHandle(hToken)

	var luid windows.LUID
	seName, _ := syscall.UTF16PtrFromString(SE_SHUTDOWN_NAME)
	r1, _, err = lookupPrivilegeValue.Call(0, uintptr(unsafe.Pointer(seName)), uintptr(unsafe.Pointer(&luid)))
	if r1 == 0 {
		log.Printf("Failed to lookup shutdown privilege: %v", err)
		return
	}

	type tokenPrivileges struct {
		PrivilegeCount uint32
		Luid           windows.LUID
		Attributes     uint32
	}

	tp := tokenPrivileges{
		PrivilegeCount: 1,
		Luid:           luid,
		Attributes:     SE_PRIVILEGE_ENABLED,
	}

	r1, _, err = adjustTokenPrivileges.Call(uintptr(hToken), 0, uintptr(unsafe.Pointer(&tp)), 0, 0, 0)
	if r1 == 0 {
		log.Printf("Failed to adjust token privileges: %v", err)
		return
	}

	exitWindowsEx.Call(EWX_REBOOT, 0)
}

func main() {
	elevateIfNeeded()

	if !checkVanguardInstalled() {
		fmt.Println("Vanguard is not installed in the default location.")
		os.Exit(1)
	}

	sysFile := filepath.Join(vanguardPath, "vgk.sys")
	if _, err := os.Stat(sysFile); err == nil {
		fmt.Println("Vanguard is currently enabled.")
		if prompt("Do you want to disable it?") {
			disableVanguard()
		}
	} else {
		fmt.Println("Vanguard is currently disabled.")
		if prompt("Do you want to enable it?") {
			enableVanguard()
		}
	}
}
