 # PagerDuty Notifications

 ## Prerequisites

 Ensure you have a valid PagerDuty API token to authenticate requests.

 ### Compatibility

 This service is compatible with the PagerDuty API for creating incidents.

 ## Usage
 ```go
 package main

 import (
     "context"
     "log"

     "github.com/nikoksr/notify"
     "github.com/nikoksr/notify/service/pagerduty"
 )

 func main() {
     // Create a new PagerDuty service. Replace 'your_pagerduty_api_token' with your actual PagerDuty API token.
     pagerDutyService, err := pagerduty.New("your_pagerduty_api_token")
     if err != nil {
         log.Fatalf("failed to create pagerduty service: %s", err)
     }

     // Set the sender address and add receivers. (required)
     pagerDutyService.SetFromAddress("sender@example.com")
     pagerDutyService.AddReceivers("ServiceDirectory1", "ServiceDirectory2")

     // Set the urgency, priority ID, and notification type. (optional)
      pagerDutyService.SetUrgency("high")
      pagerDutyService.SetPriorityID("P123456")
      pagerDutyService.SetNotificationType("incident")

     // Create a notifier instance and register the PagerDuty service to it.
     notifier := notify.New()
     notifier.UseServices(pagerDutyService)

     // Send a notification.
     err = notifier.Send(context.Background(), "Test Alert", "This is a test alert from PagerDuty service.")
     if err != nil {
         log.Fatalf("failed to send notification: %s", err)
     }

     log.Println("Notification sent successfully")
 }
 ```

 ## Configuration

 ### Required Properties
 - **API Token**: Your PagerDuty API token.
 - **From Address**: The email address of the sender. The author of the incident.
 - **Receivers**: List of PagerDuty service directories to send the incident to.

 ### Optional Properties
 - **Urgency**: The urgency of the incident (e.g., "high", "low").
 - **PriorityID**: The ID of the priority level assigned to the incident.
 - **NotificationType**: Type of notification (default is "incident").

 These properties can be set using the respective setter methods provided by the `Config` struct:
 - `SetFromAddress(string)`
 - `AddReceivers(...string)`
 - `SetUrgency(string)`
 - `SetPriorityID(string)`
 - `SetNotificationType(string)`
