//go:build linux && arm64

package tracer

import (
	"golang.org/x/sys/unix"
	"syscall"
	"unsafe"
)

const (
	NT_PRSTATUS = 1
	ArmRegsFlag = uint64(^-12345)
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
	MsgName       uint
	LenMsgName    uint32
	MsgIov        uint
	LenMsgIov     uint64
	MsgControl    uint
	LenMsgControl uint64
	Flags         int32
}

type PtraceRegsArm struct {
	Uregs [18]uint32
}

func arguments(regs *syscall.PtraceRegs) []uint64 {
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
		return []uint64{
			uint64(regs.Uregs[17]),
			uint64(regs.Uregs[1]),
			uint64(regs.Uregs[2]),
			uint64(regs.Uregs[3]),
			uint64(regs.Uregs[4]),
			uint64(regs.Uregs[5]),
		}
	}
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
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
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
	} else {
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
}

func returnValueInt(regs *syscall.PtraceRegs) (int, syscall.Errno) {
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
		if int64(regs.Uregs[0]) < 0 {
			return int(regs.Uregs[0]), syscall.Errno(^regs.Uregs[0])
		}
		return int(regs.Uregs[0]), 0
	} else {
		if int64(regs.Pstate) < 0 {
			return int(regs.Regs[0]), syscall.Errno(^regs.Regs[0])
		}
		return int(regs.Regs[0]), 0
	}
}

func isEntryStop(regs *syscall.PtraceRegs) bool {
	//log.Println(regs.Regs)
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
		return regs.Uregs[12] == 0
	}
	return regs.Regs[7] == 0
}

func inst(regs *syscall.PtraceRegs) int {
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
		return int(regs.Uregs[7])
	}
	return int(regs.Regs[8])
}

func ptraceSetRegs(pid int, regs *syscall.PtraceRegs) error {
	if regs.Pstate == ArmRegsFlag {
		regs := (*PtraceRegsArm)(unsafe.Pointer(regs))
		iov := getIovec((*byte)(unsafe.Pointer(regs)), int(unsafe.Sizeof(*regs)))
		return ptrace(syscall.PTRACE_SETREGSET, pid, NT_PRSTATUS, uintptr(unsafe.Pointer(&iov)))
	}
	iov := getIovec((*byte)(unsafe.Pointer(regs)), int(unsafe.Sizeof(*regs)))
	return ptrace(syscall.PTRACE_SETREGSET, pid, NT_PRSTATUS, uintptr(unsafe.Pointer(&iov)))
}

func ptraceGetRegs(pid int, regs *syscall.PtraceRegs) error {
	iov := getIovec((*byte)(unsafe.Pointer(regs)), int(unsafe.Sizeof(*regs)))
	if err := ptrace(syscall.PTRACE_GETREGSET, pid, NT_PRSTATUS, uintptr(unsafe.Pointer(&iov))); err != nil {
		return err
	}
	//log.Println("iovLen:", iov.Len)
	if iov.Len == uint64(unsafe.Sizeof(PtraceRegsArm{})) {
		regs.Pstate = ArmRegsFlag
	}
	return nil
}

func getIovec(base *byte, l int) unix.Iovec {
	return unix.Iovec{
		Base: base,
		Len:  uint64(l),
	}
}
