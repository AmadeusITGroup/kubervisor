package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// HandleSignal used to listen several os signal and then execute the cancel function
func HandleSignal(cancelFunc context.CancelFunc) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sigc
	cancelFunc()
}
