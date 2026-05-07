// Package fcm provides Firebase Cloud Messaging functionality for Golang
//
// Here is a simple example illustrating how to use FCM library:
//
//	func main() {
//		ctx := context.Background()
//		client, err := fcm.NewClient(
//			ctx,
//			fcm.WithCredentialsFile("path/to/serviceAccountKey.json"),
//		)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//	// Send to a single device
//	token := "test"
//	resp, err := client.Send(
//
//	ctx,
//	&messaging.Message{
//		Token: token,
//		Data: map[string]string{
//			"foo": "bar",
//		},
//	},
//
// )
//
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		fmt.Println("success count:", resp.SuccessCount)
//		fmt.Println("failure count:", resp.FailureCount)
//		fmt.Println("message id:", resp.Responses[0].MessageID)
//		fmt.Println("error msg:", resp.Responses[0].Error)
//	}
package fcm
