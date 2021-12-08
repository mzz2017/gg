//go:build linux && arm

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
		uint64(regs.Uregs[17]),
		uint64(regs.Uregs[1]),
		uint64(regs.Uregs[2]),
		uint64(regs.Uregs[3]),
		uint64(regs.Uregs[4]),
		uint64(regs.Uregs[5]),
	}
}

func setArgument(regs *syscall.PtraceRegs, order int, val uint64) {
	switch order {
	case 0:
		regs.Uregs[17] = uint32(val)
	case 1:
		regs.Uregs[1] = uint32(val)
	case 2:
		regs.Uregs[2] = uint32(val)
	case 3:
		regs.Uregs[3] = uint32(val)
	case 4:
		regs.Uregs[4] = uint32(val)
	case 5:
		regs.Uregs[5] = uint32(val)
	}
}

func returnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	if int64(regs.Uregs[0]) < 0 {
		return int(regs.Uregs[0]), syscall.Errno(^regs.Uregs[0])
	}
	return int(regs.Uregs[0]), 0
}

func isEntryStop(regs *syscall.PtraceRegs) bool {
	return regs.Uregs[12] == 0
}

func inst(regs *syscall.PtraceRegs) int {
	return int(regs.Uregs[7])
}

func ptraceSetRegs(pid int, regs *syscall.PtraceRegs) error {
	return syscall.PtraceSetRegs(pid, regs)
}

func ptraceGetRegs(pid int, regs *syscall.PtraceRegs) error {
	return syscall.PtraceGetRegs(pid, regs)
}
