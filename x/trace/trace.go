// Package trace provides runtimes capabilities related to tracing program
// execution.
//
// When using functionality from this package, consider that it adds
// significant overhead to the runtime performance of your program.
package trace

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Getfl returns the filename and line of the currently executing function.
func Getfl() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	fn := filepath.Base(frame.Func.Name())
	return fmt.Sprintf("%s:%d", fn, frame.Line)
}

// File returns the name of the currently executing file.
func File() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return filepath.Clean(frame.File)
}
