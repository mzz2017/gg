package infra

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

var (
	ErrGetPtraceScope     = fmt.Errorf("error when get ptrace scope")
	ErrGetCapability      = fmt.Errorf("error when get capability")
	ErrBadPtraceScope     = fmt.Errorf("bad ptrace scope")
	ErrBadCapability      = fmt.Errorf("bad capability")
	ErrUnknownPtraceScope = fmt.Errorf("unknown ptrace scope")
)

func GetPtraceScope() (int, error) {
	b, err := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	if err != nil {
		return -1, err
	}
	scope, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return -1, err
	}
	return scope, nil
}

func CheckPtraceCapability() error {
	scope, err := GetPtraceScope()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGetPtraceScope, err)
	}
	switch scope {
	case 0, 1:
	case 2:
		var hdr unix.CapUserHeader
		if _, _, err := unix.RawSyscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&hdr)), 0, 0); err != 0 {
			return fmt.Errorf("%w: get version: %v", ErrGetCapability, err)
		}
		var data unix.CapUserData
		if err := unix.Capget(&hdr, &data); err != nil {
			return fmt.Errorf("%w: cap get: %v", ErrGetCapability, err)
		}
		if data.Permitted&(1<<unix.CAP_SYS_PTRACE) == 0 || data.Effective&(1<<unix.CAP_SYS_PTRACE) == 0 {
			return ErrBadCapability
		}
	case 3:
		return ErrBadPtraceScope
	default:
		return fmt.Errorf("%w: %v", ErrUnknownPtraceScope, scope)
	}
	return nil
}
