package tracer

import (
	"context"
	"fmt"
	"github.com/mzz2017/gg/dialer"
	"github.com/mzz2017/gg/proxy"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"syscall"
	"time"
)

type SocketMetadata struct {
	Family int
	Type   int
	Protocol int
}

// Tracer is not thread-safe.
type Tracer struct {
	ctx        context.Context
	ignoreUDP  bool
	supportUDP bool
	log        *logrus.Logger
	proxy      *proxy.Proxy
	proc       *os.Process
	storehouse Storehouse
	socketInfo map[int]map[int]SocketMetadata
	closed     chan struct{}
	exitCode   int
	exitErr    error
}

func New(ctx context.Context, name string, argv []string, attr *os.ProcAttr, dialer *dialer.Dialer, ignoreUDP bool, logger *logrus.Logger) (*Tracer, error) {
	t := &Tracer{
		ctx:        ctx,
		log:        logger,
		supportUDP: dialer.SupportUDP(),
		proxy:      proxy.New(logger, dialer),
		socketInfo: make(map[int]map[int]SocketMetadata),
		storehouse: MakeStorehouse(),
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
		t.proc = proc
		close(done)
		time.Sleep(1 * time.Millisecond)
		code, err := t.trace()
		t.exitCode = code
		t.exitErr = err
		if err != nil {
			t.proc.Kill()
		}
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
func (t *Tracer) trace() (exitCode int, err error) {
	proc := t.proc.Pid
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
	//t.log.Tracef("child %v created\n", proc)
	for {
		select {
		case <-t.ctx.Done():
			syscall.PtraceDetach(proc)
			return 1, t.ctx.Err()
		default:
		}
		var status syscall.WaitStatus
		child, err := syscall.Wait4(-1, &status, syscall.WALL, nil)
		if err != nil {
			return 0, fmt.Errorf("wait4() threw: %w", err)
		}
		select {
		case <-t.ctx.Done():
			syscall.PtraceDetach(proc)
			return 1, t.ctx.Err()
		default:
		}
		//t.log.Tracef("main: %v, child: %v\n", proc, child)
		if t.getSocketInfo(child, 0) == nil {
			t.saveSocketInfo(child, 0, SocketMetadata{
				Family: syscall.AF_LOCAL,
				Type:   syscall.SOCK_RAW,
			})
			t.saveSocketInfo(child, 1, SocketMetadata{
				Family: syscall.AF_LOCAL,
				Type:   syscall.SOCK_RAW,
			})
			t.saveSocketInfo(child, 2, SocketMetadata{
				Family: syscall.AF_LOCAL,
				Type:   syscall.SOCK_RAW,
			})
		}
		sig := 0
		switch {
		case status.Exited():
			t.log.Tracef("child %v exited\n", child)
			t.removeProcessSocketInfo(child)
			if child == proc {
				return status.ExitStatus(), nil
			}
		case status.Signaled():
			t.log.Tracef("child %v killed\n", child)
			t.removeProcessSocketInfo(child)
			if child == proc {
				return status.ExitStatus(), nil
			}
		case status.Stopped():
			switch signal := status.StopSignal(); signal {
			case syscall.SIGTRAP:
				var regs syscall.PtraceRegs
				err = ptraceGetRegs(child, &regs)
				if err == nil {
					//t.log.Tracef("pid: %v, inst: %v", child, inst(&regs))
					entryStop := isEntryStop(&regs)
					if entryStop {
						if err := t.entryHandler(child, &regs); err != nil {
							t.log.Infof("entryHandler: %v", err)
						}
					} else {
						if err := t.exitHandler(child, &regs); err != nil {
							t.log.Infof("exitHandler: %v", err)
						}
					}
				} else {
					t.log.Tracef("PtraceGetRegs: %v", err)
				}
			default:
				// urgent I/O condition, window changed, etc.
				sig = int(signal)
				switch signal {
				case syscall.SIGSTOP:
					sig = 0
				case syscall.SIGILL:
					t.log.Errorf("%v: %v", child, signal)
				default:
					t.log.Tracef("%v: %v", child, signal)
				}
			}
		}
		syscall.PtraceSyscall(child, sig)
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
