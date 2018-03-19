// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func ignoreSignals() {
	// Ignore SIGINT and SIGQUIT so they pass through to the
	// child.
	signal.Notify(make(chan os.Signal), os.Interrupt, syscall.SIGQUIT)
}
