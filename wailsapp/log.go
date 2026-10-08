//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxLogSize = 1 << 20

var logMu sync.Mutex

func logPath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			base = dir
		}
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, "QBFind", "logs", "qbfind.log")
}

func logf(format string, args ...interface{}) {
	path := logPath()
	if path == "" {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
}

func logPanic(component string, recovered interface{}) {
	logf("PANIC in %s: %v", component, recovered)
}

func callbackGuard(name string) {
	if r := recover(); r != nil {
		logPanic("ole."+name, r)
	}
}
