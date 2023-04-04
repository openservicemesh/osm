package ns

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
)

// GetCurrentNS returns an object representing the current OS thread's network namespace
func GetCurrentNS() (NetNS, error) {
	// Lock the thread in case other goroutine executes in it and changes its
	// network namespace after getCurrentThreadNetNSPath(), otherwise it might
	// return an unexpected network namespace.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return GetNS(getCurrentThreadNetNSPath())
}

func getCurrentThreadNetNSPath() string {
	// /proc/self/ns/net returns the namespace of the main thread, not
	// of whatever thread this goroutine is running on.  Make sure we
	// use the thread's net namespace since the thread is switching around
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), Gettid())
}

func (ns *netNS) Close() error {
	if err := ns.errorIfClosed(); err != nil {
		return err
	}

	if err := ns.file.Close(); err != nil {
		return fmt.Errorf("Failed to close %q: %v", ns.file.Name(), err)
	}
	ns.closed = true

	return nil
}

func (ns *netNS) Set() error {
	if err := ns.errorIfClosed(); err != nil {
		return err
	}

	if err := Setns(int(ns.Fd()), coneNewNet); err != nil {
		return fmt.Errorf("Error switching to ns %v: %v", ns.file.Name(), err)
	}

	return nil
}

// NetNS define netns ops
type NetNS interface {
	// Do execute the passed closure in this object's network namespace,
	// attempting to restore the original namespace before returning.
	// However, since each OS thread can have a different network namespace,
	// and Go's thread scheduling is highly variable, callers cannot
	// guarantee any specific namespace is set unless operations that
	// require that namespace are wrapped with Do().  Also, no code called
	// from Do() should call runtime.UnlockOSThread(), or the risk
	// of executing code in an incorrect namespace will be greater.  See
	// https://github.com/golang/go/wiki/LockOSThread for further details.
	Do(toRun func(NetNS) error) error

	// Set the current network namespace to this object's network namespace.
	// Note that since Go's thread scheduling is highly variable, callers
	// cannot guarantee the requested namespace will be the current namespace
	// after this function is called; to ensure this wrap operations that
	// require the namespace with Do() instead.
	Set() error

	// Path returns the filesystem path representing this object's network namespace
	Path() string

	// Fd returns a file descriptor representing this object's network namespace
	Fd() uintptr

	// Close cleans up this instance of the network namespace; if this instance
	// is the last user the namespace will be destroyed
	Close() error
}

type netNS struct {
	file   *os.File
	closed bool
}

// netNS implements the NetNS interface
var _ NetNS = &netNS{}

type nsPathNotExistErr struct{ msg string }

func (e nsPathNotExistErr) Error() string { return e.msg }

type nsPathNotNSErr struct{ msg string }

func (e nsPathNotNSErr) Error() string { return e.msg }

// IsNSorErr checks the ns path
func IsNSorErr(nspath string) error {
	stat := syscall.Statfs_t{}
	if err := syscall.Statfs(nspath, &stat); err != nil {
		if os.IsNotExist(err) {
			err = nsPathNotExistErr{msg: fmt.Sprintf("failed to Statfs %q: %v", nspath, err)}
		} else {
			err = fmt.Errorf("failed to Statfs %q: %v", nspath, err)
		}
		return err
	}

	switch stat.Type {
	case procSuperMagic, nsFsMagic:
		return nil
	default:
		return nsPathNotNSErr{msg: fmt.Sprintf("unknown FS magic on %q: %x", nspath, stat.Type)}
	}
}

// GetNS returns an object representing the namespace referred to by @path
func GetNS(nspath string) (NetNS, error) {
	err := IsNSorErr(nspath)
	if err != nil {
		return nil, err
	}

	//#nosec G304
	fd, err := os.Open(nspath)
	if err != nil {
		return nil, err
	}

	return &netNS{file: fd}, nil
}

func (ns *netNS) Path() string {
	return ns.file.Name()
}

func (ns *netNS) Fd() uintptr {
	return ns.file.Fd()
}

func (ns *netNS) errorIfClosed() error {
	if ns.closed {
		return fmt.Errorf("%q has already been closed", ns.file.Name())
	}
	return nil
}

func (ns *netNS) Do(toRun func(NetNS) error) error {
	if err := ns.errorIfClosed(); err != nil {
		return err
	}

	containedCall := func(hostNS NetNS) error {
		threadNS, err := GetCurrentNS()
		if err != nil {
			return fmt.Errorf("failed to open current netns: %v", err)
		}
		defer func(threadNS NetNS) {
			_ = threadNS.Close()
		}(threadNS)

		// switch to target namespace
		if err = ns.Set(); err != nil {
			return fmt.Errorf("error switching to ns %v: %v", ns.file.Name(), err)
		}
		defer func() {
			err := threadNS.Set() // switch back
			if err == nil {
				// Unlock the current thread only when we successfully switched back
				// to the original namespace; otherwise leave the thread locked which
				// will force the runtime to scrap the current thread, that is maybe
				// not as optimal but at least always safe to do.
				runtime.UnlockOSThread()
			}
		}()

		return toRun(hostNS)
	}

	// save a handle to current network namespace
	hostNS, err := GetCurrentNS()
	if err != nil {
		return fmt.Errorf("Failed to open current namespace: %v", err)
	}
	defer func(hostNS NetNS) {
		_ = hostNS.Close()
	}(hostNS)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start the callback in a new green thread so that if we later fail
	// to switch the namespace back to the original one, we can safely
	// leave the thread locked to die without a risk of the current thread
	// left lingering with incorrect namespace.
	var innerError error
	go func() {
		defer wg.Done()
		runtime.LockOSThread()
		innerError = containedCall(hostNS)
	}()
	wg.Wait()

	return innerError
}

// WithNetNSPath executes the passed closure under the given network
// namespace, restoring the original namespace afterwards.
func WithNetNSPath(nspath string, toRun func(NetNS) error) error {
	ns, err := GetNS(nspath)
	if err != nil {
		return err
	}
	defer func(ns NetNS) {
		_ = ns.Close()
	}(ns)
	return ns.Do(toRun)
}
