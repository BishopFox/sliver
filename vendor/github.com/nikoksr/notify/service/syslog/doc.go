/*
Package syslog provides message notification integration for local or remote syslogs.

Usage:

	package main

	import (
	    "context"
	    "log"

	    sl "log/syslog"

	    "github.com/nikoksr/notify"
	    "github.com/nikoksr/notify/service/syslog"
	)

	func main() {
	    syslogSvc, err := syslog.New(sl.LOG_USER, "")
	    if err != nil {
	        log.Fatalf("syslog.New() failed: %v", err)
	    }

	    notify.UseServices(syslogSvc)

	    err = notify.Send(context.Background(), "TEST", "Hello, World!")
	    if err != nil {
	        log.Fatalf("notify.Send() failed: %v", err)
	    }
	}
*/
package syslog
