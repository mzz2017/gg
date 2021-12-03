package ptrace

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"inet.af/netaddr"
	"net"
	"syscall"
	"time"
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

func BindAddr(fd uintptr, ip []byte, port int) error {
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1); err != nil {
		return fmt.Errorf("set IP_TRANSPARENT: %w", err)
	}
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("set SO_REUSEADDR: %w", err)
	}

	var sockaddr syscall.Sockaddr

	switch len(ip) {
	case net.IPv4len:
		a4 := &syscall.SockaddrInet4{
			Port: port,
		}
		copy(a4.Addr[:], ip)
		sockaddr = a4
	case net.IPv6len:
		a6 := &syscall.SockaddrInet6{
			Port: port,
		}
		copy(a6.Addr[:], ip)
		sockaddr = a6
	default:
		return fmt.Errorf("unexpected length of ip")
	}

	return syscall.Bind(int(fd), sockaddr)
}

func NewUDPDialer(laddr netaddr.IPPort, timeout time.Duration, log *logrus.Logger) (dialer *net.Dialer) {
	return &net.Dialer{
		Timeout: timeout,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				ip := laddr.IP().As4()
				if err := BindAddr(fd, ip[:], int(laddr.Port())); err != nil {
					if log != nil {
						log.Warnf("Strict DNS lookup may fail: %v", err)
					}
				}
			})
		},
	}
}
