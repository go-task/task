// This code is released under the MIT License
// Copyright (c) 2020 Marco Molteni and the timeit contributors.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

const usage = `sleepit: sleep for the specified duration, optionally handling signals
When the line "sleepit: ready" is printed, it means that it is safe to send signals to it
Usage: sleepit <command> [<args>]
Commands
  default     Use default action: on reception of SIGINT terminate abruptly
  handle      Handle signals: on reception of SIGINT perform cleanup before exiting
  version     Show the sleepit version`

// Filled by the linker.
var fullVersion = "unknown" // example: v0.0.9-8-g941583d027-dirty

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}

	defaultCmd := flag.NewFlagSet("default", flag.ExitOnError)
	defaultSleep := defaultCmd.Duration("sleep", 5*time.Second, "Sleep duration")

	handleCmd := flag.NewFlagSet("handle", flag.ExitOnError)
	handleSleep := handleCmd.Duration("sleep", 5*time.Second, "Sleep duration")
	handleCleanup := handleCmd.Duration("cleanup", 5*time.Second, "Cleanup duration")
	handleTermAfter := handleCmd.Int("term-after", 0,
		"Terminate immediately after `N` signals.\n"+
			"Default is to terminate only when the cleanup phase has completed.")

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	switch args[0] {

	case "default":
		_ = defaultCmd.Parse(args[1:])
		if len(defaultCmd.Args()) > 0 {
			fmt.Fprintf(os.Stderr, "default: unexpected arguments: %v\n", defaultCmd.Args())
			return 2
		}
		return supervisor(*defaultSleep, 0, 0, nil)

	case "handle":
		_ = handleCmd.Parse(args[1:])
		if *handleTermAfter == 1 {
			fmt.Fprintf(os.Stderr, "handle: term-after cannot be 1\n")
			return 2
		}
		if len(handleCmd.Args()) > 0 {
			fmt.Fprintf(os.Stderr, "handle: unexpected arguments: %v\n", handleCmd.Args())
			return 2
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt) // Ctrl-C -> SIGINT
		return supervisor(*handleSleep, *handleCleanup, *handleTermAfter, sigCh)

	case "version":
		_ = versionCmd.Parse(args[1:])
		if len(versionCmd.Args()) > 0 {
			fmt.Fprintf(os.Stderr, "version: unexpected arguments: %v\n", versionCmd.Args())
			return 2
		}
		fmt.Printf("sleepit version %s\n", fullVersion)
		return 0

	default:
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
}

func supervisor(
	sleep time.Duration,
	cleanup time.Duration,
	termAfter int,
	sigCh <-chan os.Signal,
) int {
	fmt.Printf("sleepit: ready\n")
	fmt.Printf("sleepit: PID=%d sleep=%v cleanup=%v\n",
		os.Getpid(), sleep, cleanup)

	cancelWork := make(chan struct{})
	workerDone := worker(cancelWork, sleep, "work")

	cancelCleaner := make(chan struct{})
	var cleanerDone <-chan struct{}

	sigCount := 0
	for {
		select {
		case sig := <-sigCh:
			sigCount++
			fmt.Printf("sleepit: got signal=%s count=%d\n", sig, sigCount)
			if sigCount == 1 {
				// since `cancelWork` is unbuffered, sending will be synchronous:
				// we are ensured that the worker has terminated before starting cleanup.
				// This is important in some real-life situations.
				cancelWork <- struct{}{}
				cleanerDone = worker(cancelCleaner, cleanup, "cleanup")
			}
			if sigCount == termAfter {
				cancelCleaner <- struct{}{}
				return 4
			}
		case <-workerDone:
			return 0
		case <-cleanerDone:
			return 3
		}
	}
}

// Start a worker goroutine and return immediately a `workerDone` channel.
// The goroutine will prepend its prints with the prefix `name`.
// The goroutine will simulate some work and will terminate when one of the following
// conditions happens:
//  1. When `howlong` is elapsed. This case will be signaled on the `workerDone` channel.
//  2. When something happens on channel `canceled`. Note that this simulates real-life,
//     so cancellation is not instantaneous: if the caller wants a synchronous cancel,
//     it should send a message; if instead it wants an asynchronous cancel, it should
//     close the channel.
func worker(
	canceled <-chan struct{},
	howlong time.Duration,
	name string,
) <-chan struct{} {
	workerDone := make(chan struct{})
	deadline := time.Now().Add(howlong)
	go func() {
		fmt.Printf("sleepit: %s started\n", name)
		for {
			select {
			case <-canceled:
				fmt.Printf("sleepit: %s canceled\n", name)
				return
			default:
				if doSomeWork(deadline) {
					fmt.Printf("sleepit: %s done\n", name) // <== NOTE THIS LINE
					workerDone <- struct{}{}
					return
				}
			}
		}
	}()
	return workerDone
}

// Do some work and then return, so that the caller can decide whether to continue or not.
// Return true when all work is done.
func doSomeWork(deadline time.Time) bool {
	if time.Now().After(deadline) {
		return true
	}
	timeout := 100 * time.Millisecond
	time.Sleep(timeout)
	return false
}
