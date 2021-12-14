package tracer

import (
	"encoding/binary"
	"fmt"
	"github.com/mzz2017/gg/proxy"
	"github.com/sirupsen/logrus"
	"inet.af/netaddr"
	"net"
	"reflect"
	"strconv"
	"syscall"
	"unsafe"
)

func (t *Tracer) getArgsFromStorehouse(pid, inst int) ([]uint64, error) {
	v, ok := t.storehouse.Get(pid, inst)
	if !ok {
		return nil, fmt.Errorf("cannot get the arguments from storehouse because of neverseen: pid: %v, inst: %v", pid, inst)
	}
	t.storehouse.Remove(pid, inst)
	return v.([]uint64), nil
}

func (t *Tracer) saveArgsToStorehouse(pid, inst int, args []uint64) {
	t.storehouse.Save(pid, inst, args)
}

func (t *Tracer) exitHandler(pid int, regs *syscall.PtraceRegs) (err error) {
	//t.log.Infof("exitHandler: pid: %v,inst: %v", pid, inst(regs))
	//defer t.log.Infof("exitHandler: pid: %v,inst: %v: end", pid, inst(regs))
	inst := inst(regs)
	switch inst {
	case syscall.SYS_SOCKET:
		//t.log.Tracef("exitHandler: SOCKET: %v, inst: %v", pid, inst)
		args, err := t.getArgsFromStorehouse(pid, inst)
		if err != nil {
			t.log.Infoln(err)
			return nil
		}
		fd, errno := returnValueInt(regs)
		if errno != 0 {
			logrus.Infof("socket error: pid: %v, errno: %v", pid, errno)
			return nil
		}
		socketInfo := SocketMetadata{
			Family: int(args[0]),
			// FIXME: This field may be not so exact. To reproduce: curl -v example.com
			// 		So the compromise is that TCP and UDP ports to listen at must be the same.
			Type:     int(args[1]),
			Protocol: int(args[2]),
		}
		t.saveSocketInfo(pid, fd, socketInfo)
		t.log.Tracef("new socket (%v): pid: %v, fd %v", t.network(&socketInfo), pid, fd)
	case syscall.SYS_FCNTL:
		//t.log.Tracef("exitHandler: FCNTL: %v, inst: %v", pid, inst)
		// syscall.SYS_FCNTL can be used to duplicate the file descriptor.
		args, err := t.getArgsFromStorehouse(pid, inst)
		if err != nil {
			t.log.Traceln(err)
			return nil
		}
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
		newFD, errno := returnValueInt(regs)
		if errno != 0 {
			logrus.Tracef("socket error: pid: %v, errno: %v", pid, errno)
			return nil
		}
		t.saveSocketInfo(pid, newFD, *socketInfo)
		t.log.Tracef("SYS_FCNTL: copy %v -> %v", fd, newFD)
	case syscall.SYS_CLOSE:
		//t.log.Tracef("exitHandler: CLOSE: %v, inst: %v", pid, inst)
		// we do not need to know if it succeeded
		fd := Argument(regs, 0)
		t.removeSocketInfo(pid, int(fd))
		t.log.Tracef("close: pid: %v, fd %v", pid, fd)
	}
	return nil
}

func (t *Tracer) entryHandler(pid int, regs *syscall.PtraceRegs) (err error) {
	//t.log.Infof("entryHandler: pid: %v,inst: %v", pid, inst(regs))
	//defer t.log.Infof("entryHandler: pid: %v,inst: %v: end", pid, inst(regs))
	args := arguments(regs)
	switch inst(regs) {
	//case syscall.SYS_CLONE:
	//	//t.log.Tracef("entryHandler: clone: %v", pid)
	//	newRegs := *regs
	//	setArgument(&newRegs, 0, args[0] & ^uint64(syscall.CLONE_UNTRACED))
	//	if err = ptraceSetRegs(pid, &newRegs); err != nil {
	//		return err
	//	}
	case syscall.SYS_SOCKET, syscall.SYS_FCNTL:
		//t.log.Tracef("entryHandler: SOCKET, FCNTL: %v, inst: %v", pid, inst(regs))
		t.saveArgsToStorehouse(pid, inst(regs), args)
	case syscall.SYS_CONNECT, syscall.SYS_SENDTO:
		fd := args[0]
		t.log.Tracef("syscall.SYS_CONNECT, syscall.SYS_SENDTO: pid: %v, fd: %v", pid, fd)
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
		switch inst(regs) {
		case syscall.SYS_CONNECT:
			t.log.Tracef("syscall.SYS_CONNECT")
			pSockAddr = args[1]
			sockAddrLen = args[2]
			orderSockAddrLen = 2
		case syscall.SYS_SENDTO:
			t.log.Tracef("syscall.SYS_SENDTO")
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
		fd := args[0]
		t.log.Tracef("syscall.SYS_SENDMSG: pid: %v, fd: %v", pid, fd)
		socketInfo, ok := t.checkSocket(pid, fd)
		if !ok {
			return nil
		}
		if t.ignoreUDP && t.network(socketInfo) == "udp" {
			return nil
		}
		pMsg := args[1]
		bMsg := make([]byte, binary.Size(RawMsgHdr{}))
		_, err = syscall.PtracePeekData(pid, uintptr(pMsg), bMsg)
		if err != nil {
			return fmt.Errorf("PtracePeekData: %w", err)
		}
		msg := *(*RawMsgHdr)(unsafe.Pointer(&bMsg[0]))
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
	if Argument(regs, orderSockAddrLen) == uint64(len(bAddrToPoke)) {
		// they are the same, so there is no need to set the len
		return nil
	}
	newRegs := *regs
	setArgument(&newRegs, orderSockAddrLen, uint64(len(bAddrToPoke)))
	if err = ptraceSetRegs(pid, &newRegs); err != nil {
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
	case syscall.AF_INET:
	// support only ipv4, and ipv6
	case syscall.AF_INET6:
		// only filter tcp and udp traffic for ipv6
		switch t.network(socketInfo) {
		case "tcp", "udp":
		default:
			// no need to transform the fake IP to real IP
			return nil, false
		}
	default:
		return nil, false
	}
	return socketInfo, true
}
func (t *Tracer) portHackTo(socketInfo *SocketMetadata) int {
	switch t.network(socketInfo) {
	case "tcp":
		return t.proxy.TCPPort()
	case "udp":
		return t.proxy.UDPPort()
	default:
		return 0
	}
}

func (t *Tracer) network(socketInfo *SocketMetadata) string {
	if socketInfo.Family != syscall.AF_INET && socketInfo.Family != syscall.AF_INET6 {
		return ""
	}
	switch {
	case socketInfo.Type&syscall.SOCK_STREAM == syscall.SOCK_STREAM:
		switch socketInfo.Protocol {
		case 0, syscall.IPPROTO_TCP:
			return "tcp"
		}
	case socketInfo.Type&syscall.SOCK_DGRAM == syscall.SOCK_DGRAM:
		switch socketInfo.Protocol {
		case 0, syscall.IPPROTO_UDP, syscall.IPPROTO_UDPLITE:
			return "udp"
		}
	case socketInfo.Type&syscall.SOCK_RAW == syscall.SOCK_RAW:
		switch socketInfo.Protocol {
		case syscall.IPPROTO_TCP:
			return "tcp"
		case syscall.IPPROTO_UDP, syscall.IPPROTO_UDPLITE:
			return "udp"
		}
	}
	return ""
}

func (t *Tracer) handleINet4(socketInfo *SocketMetadata, bSockAddr []byte) (sockAddrToPock []byte, err error) {
	network := t.network(socketInfo)
	portHackTo := t.portHackTo(socketInfo)
	addr := *(*RawSockaddrInet4)(unsafe.Pointer(&bSockAddr[0]))
	targetPort := binary.BigEndian.Uint16(addr.Port[:])
	if network == "udp" && !t.supportUDP && targetPort != 53 {
		// skip UDP traffic
		// but only keep DNS packets sent to the port 53
		return nil, nil
	}
	isDNS := network == "udp" && targetPort == 53
	if ip := netaddr.IPFrom4(addr.Addr); (network == "tcp" || network == "udp") && ip.IsLoopback() && !isDNS {
		// skip loopback
		// but only keep DNS packets sent to the port 53
		t.log.Tracef("skip loopback: %v", netaddr.IPPortFrom(ip, binary.BigEndian.Uint16(addr.Port[:])).String())
		return nil, nil
	}
	//logrus.Traceln("before", bSockAddr)
	originAddr := net.JoinHostPort(
		netaddr.IPFrom4(addr.Addr).String(),
		strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
	)
	ip := netaddr.IPFrom4(addr.Addr)
	if network == "tcp" || network == "udp" {
		if proxy.ReservedPrefix.Contains(ip) {
			originAddr = net.JoinHostPort(
				t.proxy.GetProjection(ip), // get original domain
				strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
			)
		}
		loopback := t.proxy.AllocProjection(originAddr)
		addr.Addr = loopback.As4()
	} else if proxy.ReservedPrefix.Contains(ip) {
		if realIp, ok := t.proxy.GetRealIP(ip); ok {
			addr.Addr = realIp.As4()
		}
	}

	binary.BigEndian.PutUint16(addr.Port[:], uint16(portHackTo))
	//logrus.Traceln("port", addr.Port)
	_bSockAddrToPock := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&addr)),
		Cap:  binary.Size(addr),
		Len:  binary.Size(addr),
	}))
	bSockAddrToPock := make([]byte, len(_bSockAddrToPock))
	copy(bSockAddrToPock, _bSockAddrToPock)
	t.log.Tracef("handleINet4 (%v): origin: %v, after: %v", network, originAddr, net.JoinHostPort(netaddr.IPFrom4(addr.Addr).String(), strconv.Itoa(portHackTo)))
	return bSockAddrToPock, nil
}

func (t *Tracer) handleINet6(socketInfo *SocketMetadata, bSockAddr []byte) (sockAddrToPock []byte, err error) {
	network := t.network(socketInfo)
	portHackTo := t.portHackTo(socketInfo)

	addr := *(*RawSockaddrInet6)(unsafe.Pointer(&bSockAddr[0]))
	targetPort := binary.BigEndian.Uint16(addr.Port[:])

	if network == "udp" && !t.supportUDP && targetPort != 53 {
		// skip UDP traffic
		// but only keep DNS packets sent to the port 53
		return nil, nil
	}
	ip := netaddr.IPFrom16(addr.Addr)
	if ip.Is4in6() {
		ip = netaddr.IPFrom4(ip.As4())
	}
	if ip.IsLoopback() && !(network == "udp" && targetPort == 53) {
		// skip loopback
		// but only keep DNS packets sent to the port 53
		t.log.Tracef("skip loopback: %v", netaddr.IPPortFrom(ip, binary.BigEndian.Uint16(addr.Port[:])).String())
		return nil, nil
	}
	var originAddr string
	if proxy.ReservedPrefix.Contains(ip) {
		originAddr = net.JoinHostPort(
			t.proxy.GetProjection(ip), // get original domain
			strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
		)
	} else {
		originAddr = net.JoinHostPort(
			netaddr.IPFrom16(addr.Addr).String(),
			strconv.Itoa(int(binary.BigEndian.Uint16(addr.Port[:]))),
		)
	}
	loopback := t.proxy.AllocProjection(originAddr)
	ipv4MappedIPv6, err := netaddr.ParseIP("::ffff:" + loopback.String())
	if err != nil {
		return nil, err
	}
	addr.Addr = ipv4MappedIPv6.As16()
	binary.BigEndian.PutUint16(addr.Port[:], uint16(portHackTo))
	_bSockAddrToPock := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&addr)),
		Cap:  binary.Size(addr),
		Len:  binary.Size(addr),
	}))
	t.log.Tracef("handleINet6 (%v): origin: %v, after: %v", network, originAddr, net.JoinHostPort(ipv4MappedIPv6.String(), strconv.Itoa(portHackTo)))
	bSockAddrToPock := make([]byte, len(_bSockAddrToPock))
	copy(bSockAddrToPock, _bSockAddrToPock)
	return bSockAddrToPock, nil
}
