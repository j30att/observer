package exitcheck

import (
	stdlog "log"
	"os"
)

var (
	_ = badPanic
	_ = badLogFatal
	_ = badOSExit
	_ = localFunctionsAreAllowed
)

func badPanic() {
	panic("boom") // want "usage of panic is prohibited"
}

func badLogFatal() {
	stdlog.Fatal("stop") // want "log.Fatal call is allowed only in main function of main package"
}

func badOSExit() {
	os.Exit(1) // want "os.Exit call is allowed only in main function of main package"
}

func localFunctionsAreAllowed() {
	panic := func(_ string) {}
	panic("not builtin")

	log := localLogger{}
	log.Fatal("not stdlib log")
}

type localLogger struct{}

func (localLogger) Fatal(_ string) {}
