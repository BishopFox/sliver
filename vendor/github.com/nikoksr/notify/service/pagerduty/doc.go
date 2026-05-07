// Package pagerduty provides a notifier implementation that sends notifications via PagerDuty.
// It uses the PagerDuty API to create incidents based on the provided configuration.
//
// Usage:
// To use this package, you need to create a new instance of PagerDuty with your API token.
// You can then configure it with the necessary details like the sender's address and receivers.
// Finally, you can send notifications which will create incidents in PagerDuty.
//
// Example:
//
//	package main
//
//	import (
//	    "context"
//	    "log"
//
//	    "github.com/nikoksr/notify"
//	    "github.com/nikoksr/notify/service/pagerduty"
//	)
//
//	func main() {
//	    // Create a new PagerDuty service. Replace 'your_pagerduty_api_token' with your actual PagerDuty API token.
//	    pagerDutyService, err := pagerduty.New("your_pagerduty_api_token")
//	    if err != nil {
//	        log.Fatalf("failed to create pagerduty service: %s", err)
//	    }
//
//	    // Set the sender address and add receivers.
//	    pagerDutyService.SetFromAddress("sender@example.com")
//	    pagerDutyService.AddReceivers("ServiceDirectory1", "ServiceDirectory2")
//
//	    // Create a notifier instance and register the PagerDuty service to it.
//	    notifier := notify.New()
//	    notifier.UseServices(pagerDutyService)
//
//	    // Send a notification.
//	    err = notifier.Send(context.Background(), "Test Alert", "This is a test alert from PagerDuty service.")
//	    if err != nil {
//	        log.Fatalf("failed to send notification: %s", err)
//	    }
//
//	    log.Println("Notification sent successfully")
//	}
//
// This package requires a valid PagerDuty API token to authenticate requests.
package pagerduty
