//go:build linux && 386

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
	MsgName       uint32
	LenMsgName    uint32
	MsgIov        uint32
	LenMsgIov     uint32
	MsgControl    uint32
	LenMsgControl uint32
	Flags         int32
}

func arguments(regs *syscall.PtraceRegs) []uint64 {
	return []uint64{
		uint64(regs.Ebx),
		uint64(regs.Ecx),
		uint64(regs.Edx),
		uint64(regs.Esi),
		uint64(regs.Edi),
		uint64(regs.Ebp),
	}
}

func setArgument(regs *syscall.PtraceRegs, order int, val uint64) {
	switch order {
	case 0:
		regs.Ebx = int32(val)
	case 1:
		regs.Ecx = int32(val)
	case 2:
		regs.Edx = int32(val)
	case 3:
		regs.Esi = int32(val)
	case 4:
		regs.Edi = int32(val)
	case 5:
		regs.Ebp = int32(val)
	}
}

func returnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	if int64(regs.Eax) < 0 {
		return int(regs.Eax), syscall.Errno(^regs.Eax)
	}
	return int(regs.Eax), 0
}

func isEntryStop(regs *syscall.PtraceRegs) bool {
	return regs.Eax == -int32(syscall.ENOSYS)
}

func inst(regs *syscall.PtraceRegs) int {
	return regs.Orig_eax
}
