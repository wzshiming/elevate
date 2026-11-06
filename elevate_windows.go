package elevate

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	seeMaskFlagNoUI       = 0x00000400
)

// shellExecuteInfo is SHELLEXECUTEINFOW, which x/sys/windows lacks.
type shellExecuteInfo struct {
	cbSize        uint32
	fMask         uint32
	hwnd          windows.HWND
	verb          *uint16
	file          *uint16
	parameters    *uint16
	directory     *uint16
	show          int32
	instApp       windows.Handle
	idList        uintptr
	class         *uint16
	keyClass      windows.Handle
	hotKey        uint32
	iconOrMonitor windows.Handle
	process       windows.Handle
}

var procShellExecuteExW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// IsElevated reports whether the process token is elevated.
func IsElevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

// winProc is a process launched by Launch; it owns the process handle.
type winProc struct {
	ctx  context.Context
	h    windows.Handle
	once sync.Once
	err  error
}

func newWinProc(ctx context.Context, h windows.Handle) *winProc {
	return &winProc{ctx: ctx, h: h}
}

// Wait waits for the process to exit or for ctx to be done, whichever comes first, and then closes the process handle; a done ctx does not terminate the elevated process, which keeps running.
// Later calls return the first call's result.
func (p *winProc) Wait() error {
	p.once.Do(func() {
		p.err = p.wait()
		windows.CloseHandle(p.h)
	})
	return p.err
}

func (p *winProc) wait() error {
	done, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return fmt.Errorf("elevate: CreateEvent: %w", err)
	}
	defer windows.CloseHandle(done)
	fired := make(chan struct{})
	stop := context.AfterFunc(p.ctx, func() {
		windows.SetEvent(done)
		close(fired)
	})
	// The event must outlive a SetEvent already under way.
	defer func() {
		if !stop() {
			<-fired
		}
	}()

	event, err := windows.WaitForMultipleObjects([]windows.Handle{p.h, done}, false, windows.INFINITE)
	if err != nil {
		return fmt.Errorf("elevate: WaitForMultipleObjects: %w", err)
	}
	if event != windows.WAIT_OBJECT_0 {
		return p.ctx.Err()
	}
	var code uint32
	if err := windows.GetExitCodeProcess(p.h, &code); err != nil {
		return fmt.Errorf("elevate: GetExitCodeProcess: %w", err)
	}
	if code != 0 {
		return &ExitError{Code: int(code)}
	}
	return nil
}

// shellExecuteInfoFor describes running exe with args through the runas verb, hidden and returning the process handle.
func shellExecuteInfoFor(exe string, args []string) (shellExecuteInfo, error) {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return shellExecuteInfo{}, err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return shellExecuteInfo{}, err
	}
	params, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(args))
	if err != nil {
		return shellExecuteInfo{}, err
	}
	info := shellExecuteInfo{
		fMask:      seeMaskNoCloseProcess | seeMaskNoAsync | seeMaskFlagNoUI,
		verb:       verb,
		file:       file,
		parameters: params,
		show:       windows.SW_HIDE,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	return info, nil
}

// Launch runs exe with args elevated after the user approves the UAC prompt, which it waits for and which ctx cannot cancel once shown; ctx bounds the returned Proc's Wait.
func Launch(ctx context.Context, exe string, args []string) (Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := shellExecuteInfoFor(exe, args)
	if err != nil {
		return nil, err
	}

	// ShellExecuteEx may use COM, which wants an initialized apartment on the calling thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err == nil || err == windows.Errno(windows.S_FALSE) {
		defer windows.CoUninitialize()
	}
	ok, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return nil, shellExecuteError(callErr, info.instApp)
	}
	if info.process == 0 {
		return nil, errors.New("elevate: ShellExecuteEx started no process")
	}
	return newWinProc(ctx, info.process), nil
}

// shellExecuteError reports a failed ShellExecuteEx from its last error, falling back to the SE_ERR code in hInstApp when the last error is unset, and ErrDenied when the user declined the prompt.
func shellExecuteError(lastErr error, instApp windows.Handle) error {
	if errors.Is(lastErr, windows.ERROR_CANCELLED) {
		return ErrDenied
	}
	if lastErr == nil || errors.Is(lastErr, windows.Errno(0)) {
		return fmt.Errorf("elevate: ShellExecuteEx runas failed with SE_ERR %d", instApp)
	}
	return fmt.Errorf("elevate: ShellExecuteEx runas: %w", lastErr)
}
