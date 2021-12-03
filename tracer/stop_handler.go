package tracer

import (
	"encoding/binary"
	"fmt"
	"github.com/mzz2017/gg/proxy"
	"github.com/mzz2017/gg/ptrace"
	"github.com/sirupsen/logrus"
	"inet.af/netaddr"
	"net"
	"reflect"
	"strconv"
	"syscall"
	"unsafe"
)

func (t *Tracer) exitHandler(pid int, regs *syscall.PtraceRegs) (err error) {
	switch regs.Orig_rax {
	case syscall.SYS_SOCKET:
		fd, errno := ptrace.ReturnValueInt(regs)
		if errno != 0 {
			logrus.Tracef("socket error: pid: %v, errno: %v", pid, errno)
			return nil
		}
		args := ptrace.Arguments(regs)
		socketInfo := SocketMetadata{
			Family: int(args[0]),
			Type:   int(args[1]),
		}
		t.saveSocketInfo(pid, fd, socketInfo)
		t.log.Tracef("socket (%v): pid: %v, fd %v", t.network(&socketInfo), pid, fd)
	case syscall.SYS_FCNTL:
		// syscall.SYS_FCNTL can be used to duplicate the file descriptor.
		args := ptrace.Arguments(regs)
		switch args[1] {
		case syscall.F_DUPFD, syscall.F_DUPFD_CLOEXEC:
		default:
			return nil
		}
		fd := args[0]
		socketInfo := t.getSocketInfo(pid, int(fd))
		if socketInfo == nil {
			logrus.Tracef("syscall.F_DUPFD: socketInfo cannot found: pid: %v, fd: %v", pid, fd)
			return nil
		}
		newFD, errno := ptrace.ReturnValueInt(regs)
		if errno != 0 {
			logrus.Tracef("socket error: pid: %v, errno: %v", pid, errno)
			return nil
		}
		t.saveSocketInfo(pid, newFD, *socketInfo)
		t.log.Tracef("SYS_FCNTL: copy %v -> %v", fd, newFD)
	case syscall.SYS_CLOSE:
		// we do not need to know if it succeeded
		fd := ptrace.Argument(regs, 0)
		t.removeSocketInfo(pid, int(fd))
	}
	return nil
}

func (t *Tracer) entryHandler(pid int, regs *syscall.PtraceRegs) (err error) {
	//logrus.Println(regs.Orig_rax)
	switch regs.Orig_rax {
	case syscall.SYS_SOCKET:
		args := ptrace.Arguments(regs)
		family := int(args[0])
		typ := int(args[1])
		// Convert all INET6 calls to INET4 calls because there is only one IPv6 loopback address but many IPv4 loopback addresses.
		// And it is okay because they are all sent to our proxy agent and restore to their original address.
		if family == syscall.AF_INET6 && (typ&syscall.SOCK_STREAM == syscall.SOCK_STREAM ||
			typ&syscall.SOCK_DGRAM == syscall.SOCK_DGRAM) {
			newRegs := *regs
			ptrace.SetArgument(&newRegs, 0, uint64(syscall.AF_INET))
			if err = syscall.PtraceSetRegs(pid, &newRegs); err != nil {
				return fmt.Errorf("set family for socket: %w", err)
			}
		}
	case syscall.SYS_CONNECT, syscall.SYS_SENDTO:
		t.log.Tracef("syscall.SYS_CONNECT || syscall.SYS_SENDTO")
		args := ptrace.Arguments(regs)
		fd := args[0]
		socketInfo, ok := t.checkSocket(pid, fd)
		if !ok {
			return nil
		}
		if t.ignoreUDP && t.network(socketInfo) == "udp" {
			return nil
		}
		var (
			pSockAddr, sockAddrLen uint64
			orderSockAddrLen       int
		)
		switch regs.Orig_rax {
		case syscall.SYS_CONNECT:
			pSockAddr = args[1]
			sockAddrLen = args[2]
			orderSockAddrLen = 2
		case syscall.SYS_SENDTO:
			pSockAddr = args[4]
			sockAddrLen = args[5]
			orderSockAddrLen = 5
			if sockAddrLen == 0 {
				// it is just used like send, which does not carry any address information.
				return nil
			}
		}
		bSockAddr := make([]byte, sockAddrLen)
		if _, err = syscall.PtracePeekData(pid, uintptr(pSockAddr), bSockAddr); err != nil {
			return fmt.Errorf("PtracePeekData: %w", err)
		}
		//t.log.Tracef("%v %v", bSockAddr, sockAddrLen)
		sockAddr := *(*syscall.RawSockaddr)(unsafe.Pointer(&bSockAddr[0]))

		switch sockAddr.Family {
		case syscall.AF_INET:
			var bSockAddrToPock []byte
			if bSockAddrToPock, err = t.handleINet4(socketInfo, bSockAddr); err != nil {
				return fmt.Errorf("handleINet4: %w", err)
			}
			if bSockAddrToPock == nil {
				return nil
			}
			if err = pokeAddrToArgument(pid, regs, bSockAddrToPock, uintptr(pSockAddr), orderSockAddrLen); err != nil {
				return fmt.Errorf("pokeAddrToArgument: %w", err)
			}
		case syscall.AF_INET6:
			var bSockAddrToPock []byte
			if bSockAddrToPock, err = t.handleINet6(socketInfo, bSockAddr); err != nil {
				return fmt.Errorf("handleINet6: %w", err)
			}
			if bSockAddrToPock == nil {
				return nil
			}
			if err = pokeAddrToArgument(pid, regs, bSockAddrToPock, uintptr(pSockAddr), orderSockAddrLen); err != nil {
				return fmt.Errorf("pokeAddrToArgument: %w", err)
			}
		}
	case syscall.SYS_SENDMSG:
		t.log.Tracef("syscall.SYS_SENDMSG")
		args := ptrace.Arguments(regs)
		fd := args[0]
		socketInfo, ok := t.checkSocket(pid, fd)
		if !ok {
			return nil
		}
		if t.ignoreUDP && t.network(socketInfo) == "udp" {
			return nil
		}
		pMsg := args[1]
		bMsg := make([]byte, binary.Size(ptrace.RawMsgHdr{}))
		_, err = syscall.PtracePeekData(pid, uintptr(pMsg), bMsg)
		if err != nil {
			return fmt.Errorf("PtracePeekData: %w", err)
		}
		msg := *(*ptrace.RawMsgHdr)(unsafe.Pointer(&bMsg[0]))
		if msg.LenMsgName == 0 {
			// no target
			return nil
		}
		bSockAddr := make([]byte, msg.LenMsgName)
		if _, err := syscall.PtracePeekData(pid, uintptr(msg.MsgName), bSockAddr); err != nil {
			return err
		}
		//t.log.Tracef("bSockAddr: %v", bSockAddr)
		sockAddr := *(*syscall.RawSockaddr)(unsafe.Pointer(&bSockAddr[0]))
		switch sockAddr.Family {
		case syscall.AF_INET:
			var bSockAddrToPock []byte
			if bSockAddrToPock, err = t.handleINet4(socketInfo, bSockAddr); err != nil {
				return fmt.Errorf("handleINet4: %w", err)
			}
			//t.log.Tracef("bSockAddrToPock: %v", bSockAddrToPock)
			if bSockAddrToPock == nil {
				return nil
			}
			if _, err := syscall.PtracePokeData(pid, uintptr(msg.MsgName), bSockAddrToPock); err != nil {
				return fmt.Errorf("set addr for SYS_SENDMSG: %w", err)
			}
		case syscall.AF_INET6:
			var bSockAddrToPock []byte
			if bSockAddrToPock, err = t.handleINet6(socketInfo, bSockAddr); err != nil {
				return fmt.Errorf("handleINet6: %w", err)
			}
			//t.log.Tracef("bSockAddrToPock: %v", bSockAddrToPock)
			if bSockAddrToPock == nil {
				return nil
			}
			if _, err := syscall.PtracePokeData(pid, uintptr(msg.MsgName), bSockAddrToPock); err != nil {
				return fmt.Errorf("set addr for SYS_SENDMSG: %w", err)
			}
			msg.LenMsgName = uint32(len(bSockAddrToPock))

			bMsg = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(&msg)),
				Cap:  binary.Size(msg),
				Len:  binary.Size(msg),
			}))
			if _, err := syscall.PtracePokeData(pid, uintptr(pMsg), bMsg); err != nil {
				return fmt.Errorf("set addr for SYS_SENDMSG: %w", err)
			}
		}
	}
	return nil
}

func pokeAddrToArgument(pid int, regs *syscall.PtraceRegs, bAddrToPoke []byte, pSockAddr uintptr, orderSockAddrLen int) (err error) {
	if _, err = syscall.PtracePokeData(pid, pSockAddr, bAddrToPoke); err != nil {
		return fmt.Errorf("pokeAddrToArgument: %w", err)
	}
	if ptrace.Argument(regs, orderSockAddrLen) == uint64(len(bAddrToPoke)) {
		// they are the same, so there is no need to set the len
		return nil
	}
	newRegs := *regs
	ptrace.SetArgument(&newRegs, orderSockAddrLen, uint64(len(bAddrToPoke)))
	if err = syscall.PtraceSetRegs(pid, &newRegs); err != nil {
		return fmt.Errorf("set addr len for connect: %w", err)
	}
	return nil
}

func (t *Tracer) checkSocket(pid int, fd uint64) (socketInfo *SocketMetadata, expected bool) {
	socketInfo = t.getSocketInfo(pid, int(fd))
	if socketInfo == nil {
		logrus.Tracef("socketInfo of socket cannot found: pid: %v, fd: %v", pid, fd)
		return nil, false
	}
	switch socketInfo.Family {
	case syscall.AF_INET, syscall.AF_INET6:
		// support only ipv4, and ipv6
	default:
		return nil, false
	}
	switch {
	// support only tcp, and udp
	case socketInfo.Type&syscall.SOCK_STREAM == syscall.SOCK_STREAM:
	case socketInfo.Type&syscall.SOCK_DGRAM == syscall.SOCK_DGRAM:
	default:
		return nil, false
	}
	return socketInfo, true
}
func (t *Tracer) port(socketInfo *SocketMetadata) int {
	switch {
	case socketInfo.Type&syscall.SOCK_STREAM == syscall.SOCK_STREAM:
		return t.proxy.TCPPort()
	case socketInfo.Type&syscall.SOCK_DGRAM == syscall.SOCK_DGRAM:
		return t.proxy.UDPPort()
	default:
		return 0
	}
}

func (t *Tracer) network(socketInfo *SocketMetadata) string {
	switch {
	case socketInfo.Type&syscall.SOCK_STREAM == syscall.SOCK_STREAM:
		return "tcp"
	case socketInfo.Type&syscall.SOCK_DGRAM == syscall.SOCK_DGRAM:
		return "udp"
	default:
		return ""
	}
}

func (t *Tracer) handleINet4(socketInfo *SocketMetadata, bSockAddr []byte) (sockAddrToPock []byte, err error) {
	network := t.network(socketInfo)
	port := t.port(socketInfo)
	addr := *(*ptrace.RawSockaddrInet4)(unsafe.Pointer(&bSockAddr[0]))
	if addr.Addr[0] == 127 {
		// skip loopback
		t.log.Tracef("skip loopback: %v:%v", addr.Addr, binary.BigEndian.Uint16(addr.Port[:]))
		return nil, nil
	}
	//logrus.Traceln("before", bSockAddr)
	var originAddr string
	if ip := netaddr.IPFrom4(addr.Addr); proxy.ReservedPrefix.Contains(ip) {
		originAddr = net.JoinHostPort(
			t.proxy.GetProjection(ip), // get original domain
			strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
		)
	} else {
		originAddr = net.JoinHostPort(
			netaddr.IPFrom4(addr.Addr).String(),
			strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
		)
	}
	loopback := t.proxy.AllocProjection(originAddr)
	addr.Addr = loopback.As4()
	binary.BigEndian.PutUint16(addr.Port[:], uint16(port))
	//logrus.Traceln("port", addr.Port)
	_bSockAddrToPock := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&addr)),
		Cap:  binary.Size(addr),
		Len:  binary.Size(addr),
	}))
	bSockAddrToPock := make([]byte, len(_bSockAddrToPock))
	copy(bSockAddrToPock, _bSockAddrToPock)
	t.log.Tracef("handleINet4 (%v): origin: %v, after: %v", network, originAddr, net.JoinHostPort(loopback.String(), strconv.Itoa(port)))
	return bSockAddrToPock, nil
}

func (t *Tracer) handleINet6(socketInfo *SocketMetadata, bSockAddr []byte) (sockAddrToPock []byte, err error) {
	if socketInfo.Family != syscall.AF_INET {
		return nil, fmt.Errorf("connect AF_INET6: unexpected socket family: %v, it should be: %v", socketInfo.Family, syscall.AF_INET)
	}

	network := t.network(socketInfo)
	port := t.port(socketInfo)

	addr := *(*ptrace.RawSockaddrInet6)(unsafe.Pointer(&bSockAddr[0]))
	logrus.Traceln(addr)
	// the new size of sock_addr is smaller, so it is safe
	originAddr := net.JoinHostPort(
		netaddr.IPFrom16(addr.Addr).String(),
		strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
	)
	loopback := t.proxy.AllocProjection(originAddr)
	addrToPoke := ptrace.RawSockaddrInet4{
		Family: syscall.AF_INET,
		Port:   [2]byte{},
		Addr:   loopback.As4(),
		Zero:   [8]uint8{},
	}
	if netaddr.IPFrom16(addr.Addr).IsLoopback() {
		// we only change the addr from ::1 to 127.0.0.1 for loopback address and keep the original port number
		addrToPoke.Port = addr.Port
	} else {
		binary.BigEndian.PutUint16(addrToPoke.Port[:], uint16(port))
	}
	bSockAddrToPock := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&addrToPoke)),
		Cap:  binary.Size(addrToPoke),
		Len:  binary.Size(addrToPoke),
	}))
	t.log.Tracef("handleINet6 (%v): origin: %v, after: %v", network, originAddr, net.JoinHostPort(loopback.String(), strconv.Itoa(port)))
	return bSockAddrToPock, nil
}
