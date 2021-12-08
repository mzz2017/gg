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
