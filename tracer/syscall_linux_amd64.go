//go:build linux && amd64

package tracer

import (
	"syscall"
)

// RawSockaddrInet4 is a bit different from syscall.RawSockaddrInet4 that Port should be encoded by BigEndian.
type RawSockaddrInet4 struct {
	Family uint16
	Port   [2]byte
	Addr   [4]byte /* in_addr */
	Zero   [8]uint8
}

// RawSockaddrInet6 is a bit different from syscall.RawSockaddrInet6 that fields except Family should be encoded by BigEndian.
type RawSockaddrInet6 struct {
	Family   uint16
	Port     [2]byte
	Flowinfo [4]byte
	Addr     [16]byte /* in6_addr */
	Scope_id [4]byte
}

type RawMsgHdr struct {
	MsgName       uint64
	LenMsgName    uint32
	MsgIov        uint64
	LenMsgIov     uint64
	MsgControl    uint64
	LenMsgControl uint64
	Flags         int32
}

func arguments(regs *syscall.PtraceRegs) []uint64 {
	return []uint64{
		regs.Rdi,
		regs.Rsi,
		regs.Rdx,
		regs.R10,
		regs.R8,
		regs.R9,
	}
}

func setArgument(regs *syscall.PtraceRegs, order int, val uint64) {
	switch order {
	case 0:
		regs.Rdi = val
	case 1:
		regs.Rsi = val
	case 2:
		regs.Rdx = val
	case 3:
		regs.R10 = val
	case 4:
		regs.R8 = val
	case 5:
		regs.R9 = val
	}
}

func returnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	if int64(regs.Rax) < 0 {
		return int(regs.Rax), syscall.Errno(^regs.Rax)
	}
	return int(regs.Rax), 0
}

func isEntryStop(regs *syscall.PtraceRegs) bool {
	return int64(regs.Rax) == -int64(syscall.ENOSYS)
}

func inst(regs *syscall.PtraceRegs) int {
	return int(regs.Orig_rax)
}

func ptraceSetRegs(pid int, regs *syscall.PtraceRegs) error {
	return syscall.PtraceSetRegs(pid, regs)
}

func ptraceGetRegs(pid int, regs *syscall.PtraceRegs) error {
	return syscall.PtraceGetRegs(pid, regs)
}
