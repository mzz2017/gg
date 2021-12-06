//go:build linux && arm64

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
		regs.Regs[0],
		regs.Regs[1],
		regs.Regs[2],
		regs.Regs[3],
		regs.Regs[4],
		regs.Regs[5],
	}
}

func setArgument(regs *syscall.PtraceRegs, order int, val uint64) {
	switch order {
	case 0:
		regs.Regs[0] = val
	case 1:
		regs.Regs[1] = val
	case 2:
		regs.Regs[2] = val
	case 3:
		regs.Regs[3] = val
	case 4:
		regs.Regs[4] = val
	case 5:
		regs.Regs[5] = val
	}
}

func returnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	if int64(regs.Pstate) < 0 {
		return int(regs.Regs[0]), syscall.Errno(^regs.Regs[0])
	}
	return int(regs.Regs[0]), 0
}

func isEntryStop(regs *syscall.PtraceRegs) bool {
	return int64(regs.Regs[0]) == -int64(syscall.ENOSYS)
}

func inst(regs *syscall.PtraceRegs) int {
	return int(regs.Regs[8])
}
