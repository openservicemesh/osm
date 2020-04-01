package signals

import (
	"os"
	"os/signal"
	"syscall"
)

var exitSignals = []os.Signal{os.Interrupt, syscall.SIGTERM} // SIGTERM is POSIX specific

// RegisterExitHandlers returns a handle channel to wait on exit signals
func RegisterExitHandlers() chan struct{} { // TODO: needs to return a recv channel
	stop := make(chan struct{})
	s := make(chan os.Signal, len(exitSignals))
	signal.Notify(s, exitSignals...)

	go func() {
		// Wait for a singal from the OS before signalling others on 'stop' channel
		<-s
		close(stop)
	}()

	return stop
}
