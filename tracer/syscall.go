package tracer

import (
	"syscall"
)

func Argument(regs *syscall.PtraceRegs, order int) uint64 {
	argsMapper := arguments(regs)
	if order >= 0 && order < len(argsMapper) {
		return argsMapper[order]
	}
	return argsMapper[0]
}

func ptrace(request int, pid int, addr uintptr, data uintptr) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_PTRACE, uintptr(request), uintptr(pid), uintptr(addr), uintptr(data), 0, 0)
	if e1 != 0 {
		err = e1
	}
	return
}
