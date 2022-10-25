package task

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-task/task/v3/internal/logger"
)

// NOTE(@andreynering): This function intercepts SIGINT and SIGTERM signals
// so the Task process is not killed immediately and processes running have
// time to do cleanup work.
func (e *Executor) InterceptInterruptSignals() {
	ch := make(chan os.Signal, 3)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		for i := 1; i <= 3; i++ {
			sig := <-ch

			if i < 3 {
				e.Logger.Outf(logger.Yellow, `task: Signal received: "%s"`, sig)
				continue
			}

			e.Logger.Errf(logger.Red, `task: Signal received for the third time: "%s". Forcing shutdown`, sig)
			_ = e.Compiler.Close()
			os.Exit(1)
		}
	}()
}
