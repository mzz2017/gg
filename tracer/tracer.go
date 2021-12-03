package tracer

import (
	"fmt"
	"github.com/mzz2017/gg/proxy"
	"github.com/mzz2017/gg/ptrace"
	"github.com/sirupsen/logrus"
	proxy2 "golang.org/x/net/proxy"
	"os"
	"runtime"
	"syscall"
	"time"
)

type SocketMetadata struct {
	Family int
	Type   int
}

// Tracer is not thread-safe.
type Tracer struct {
	ignoreUDP  bool
	log        *logrus.Logger
	proxy      *proxy.Proxy
	mainPID    int
	socketInfo map[int]map[int]SocketMetadata
	closed     chan struct{}
	exitCode   int
	exitErr    error
}

func New(name string, argv []string, attr *os.ProcAttr, dialer proxy2.Dialer, ignoreUDP bool, logger *logrus.Logger) (*Tracer, error) {
	t := &Tracer{
		log:        logger,
		proxy:      proxy.New(logger, dialer),
		socketInfo: make(map[int]map[int]SocketMetadata),
		closed:     make(chan struct{}),
		ignoreUDP:  ignoreUDP,
	}
	go func() {
		if err := t.proxy.ListenAndServe(0); err != nil {
			t.exitCode = 1
			t.exitErr = err
			close(t.closed)
		}
	}()
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		if attr == nil {
			attr = &os.ProcAttr{}
		}
		if attr.Sys == nil {
			attr.Sys = &syscall.SysProcAttr{}
		}
		attr.Sys.Ptrace = true
		attr.Sys.Pdeathsig = syscall.SIGCHLD
		proc, err := os.StartProcess(name, argv, attr)
		if err != nil {
			close(done)
			t.exitErr = err
			return
		}
		close(done)
		time.Sleep(1 * time.Millisecond)
		code, err := t.trace(proc.Pid)
		t.exitCode = code
		t.exitErr = err
		close(t.closed)
	}()
	<-done
	if t.exitErr != nil {
		return nil, t.exitErr
	}
	return t, nil
}

func (t *Tracer) Wait() (exitCode int, err error) {
	<-t.closed
	return t.exitCode, t.exitErr
}

// Trace traces the process. proc is the process ID (main thread).
func (t *Tracer) trace(proc int) (exitCode int, err error) {
	// Thanks https://stackoverflow.com/questions/5477976/how-to-ptrace-a-multi-threaded-application and https://github.com/hmgle/graftcp
	err = syscall.PtraceAttach(proc)
	if err != nil {
		if err == syscall.EPERM {
			_, err = syscall.PtraceGetEventMsg(proc)
			if err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	options := syscall.PTRACE_O_TRACECLONE | syscall.PTRACE_O_TRACEFORK |
		syscall.PTRACE_O_TRACEVFORK | syscall.PTRACE_O_TRACEEXEC
	if err = syscall.PtraceSetOptions(proc, options); err != nil {
		return 0, err
	}
	if err = syscall.PtraceSyscall(proc, 0); err != nil {
		if err == syscall.ESRCH {
			return 0, fmt.Errorf("tracee died unexpectedly: %w", err)
		}
		return 0, fmt.Errorf("PtraceSyscall() threw: %w", err)
	}
	//logrus.Tracef("child %v created\n", proc)
	for {
		var status syscall.WaitStatus
		child, err := syscall.Wait4(-1, &status, syscall.WALL, nil)
		if err != nil {
			return 0, fmt.Errorf("wait4() threw: %w", err)
		}
		//logrus.Tracef("main: %v, child: %v\n", proc, child)
		switch {
		case status.Exited():
			//logrus.Tracef("child %v exited\n", child)
			t.removeProcessSocketInfo(child)
			if child == proc {
				return status.ExitStatus(), nil
			}
		case status.Signaled():
			//logrus.Tracef("child %v killed\n", child)
			t.removeProcessSocketInfo(child)
			if child == proc {
				return status.ExitStatus(), nil
			}
		case status.Stopped():
			switch signal := status.StopSignal(); signal {
			case syscall.SIGTRAP:
				var regs syscall.PtraceRegs
				err = syscall.PtraceGetRegs(child, &regs)
				if err == nil {
					entryStop := ptrace.IsEntryStop(&regs)
					if entryStop {
						if err := t.entryHandler(child, &regs); err != nil {
							logrus.Tracef("entryHandler: %v", err)
						}
					} else {
						if err := t.exitHandler(child, &regs); err != nil {
							logrus.Tracef("exitHandler: %v", err)
						}
					}
				}
			default:
				// urgent I/O condition, window changed, etc.
				//logrus.Tracef("%v: stopped: %v", child, signal)
			}
		}
		syscall.PtraceSyscall(child, 0)
	}
}

func (t *Tracer) getSocketInfo(pid int, socketFD int) (metadata *SocketMetadata) {
	if _, ok := t.socketInfo[pid]; !ok {
		return nil
	}
	m, ok := t.socketInfo[pid][socketFD]
	if !ok {
		return nil
	}
	return &m
}

func (t *Tracer) saveSocketInfo(pid int, socketFD int, metadata SocketMetadata) {
	if t.socketInfo[pid] == nil {
		t.socketInfo[pid] = make(map[int]SocketMetadata)
	}
	t.socketInfo[pid][socketFD] = metadata
}

func (t *Tracer) removeSocketInfo(pid int, socketFD int) {
	if _, ok := t.socketInfo[pid]; !ok {
		return
	}
	if _, ok := t.socketInfo[pid][socketFD]; ok {
		delete(t.socketInfo[pid], socketFD)
		if len(t.socketInfo[pid]) == 0 {
			delete(t.socketInfo, pid)
		}
	}
}

func (t *Tracer) removeProcessSocketInfo(pid int) {
	if _, ok := t.socketInfo[pid]; !ok {
		return
	}
	if len(t.socketInfo[pid]) == 0 {
		delete(t.socketInfo, pid)
	}
}
