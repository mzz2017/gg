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

func SetArgument(regs *syscall.PtraceRegs, order int, val uint64) {
	setArgument(regs, order, val)
}

func Arguments(regs *syscall.PtraceRegs) []uint64 {
	return arguments(regs)
}

func ReturnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	return returnValueInt(regs)
}

func IsEntryStop(regs *syscall.PtraceRegs) bool {
	return isEntryStop(regs)
}

