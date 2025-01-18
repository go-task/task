package task

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-task/task/v3/internal/logger"
)

const maxInterruptSignals = 3

// NOTE(@andreynering): This function intercepts SIGINT and SIGTERM signals
// so the Task process is not killed immediately and processes running have
// time to do cleanup work.
func (e *Executor) InterceptInterruptSignals() {
	ch := make(chan os.Signal, maxInterruptSignals)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		for i := range maxInterruptSignals {
			sig := <-ch

			if i+1 >= maxInterruptSignals {
				e.Logger.Errf(logger.Red, "task: Signal received for the third time: %q. Forcing shutdown\n", sig)
				os.Exit(1)
			}

			e.Logger.Outf(logger.Yellow, "task: Signal received: %q\n", sig)
		}
	}()
}
