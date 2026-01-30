If you have an issue logging into your Twilio SendGrid account, please read this [document](https://sendgrid.com/docs/ui/account-and-settings/troubleshooting-login/). For any questions regarding login issues, please contact our [support team](https://support.sendgrid.com).

If you have a non-library Twilio SendGrid issue, please contact our [support team](https://support.sendgrid.com).

If you can't find a solution below, please open an [issue](https://github.com/sendgrid/sendgrid-go/issues).


## Table of Contents

* [Migrating from v2 to v3](#migrating)
* [Continue Using v2](#v2)
* [Testing v3 /mail/send Calls Directly](#testing)
* [Error Messages](#error)
* [Versions](#versions)
* [Environment Variables and Your Twilio SendGrid API Key](#environment)
* [Viewing the Request Body](#request-body)
* [Verifying Event Webhooks](#signed-webhooks)

<a name="migrating"></a>
## Migrating from v2 to v3

Please review [our guide](https://sendgrid.com/docs/Classroom/Send/v3_Mail_Send/how_to_migrate_from_v2_to_v3_mail_send.html) on how to migrate from v2 to v3.

<a name="v2"></a>
## Continue Using v2

[Here](https://github.com/sendgrid/sendgrid-go/tree/0bf6332788d0230b7da84a1ae68d7531073200e1) is the last working version with v2 support.

Download:

Click the "Clone or download" green button in [GitHub](https://github.com/sendgrid/sendgrid-go/tree/0bf6332788d0230b7da84a1ae68d7531073200e1) and choose download.

<a name="testing"></a>
## Testing v3 /mail/send Calls Directly

[Here](https://sendgrid.com/docs/for-developers/sending-email/curl-examples) are some cURL examples for common use cases.

<a name="error"></a>
## Error Messages

An error is returned if caused by client policy (such as CheckRedirect), or failure to speak HTTP (such as a network connectivity problem).

To read the error message returned by Twilio SendGrid's API:

```go
func main() {
	from := mail.NewEmail("Example User", "test@example.com")
	subject := "Hello World from the Twilio SendGrid Go Library"
	to := mail.NewEmail("Example User", "test@example.com")
	content := mail.NewContent("text/plain", "some text here")
	m := mail.NewV3MailInit(from, subject, to, content)

	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KE"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)
	response, err := sendgrid.API(request)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

__CAUTION__: A non-2xx status code doesn't cause an error on sendgrid.API and the application has to verify the response:

```golang
resp, err := sendgrid.API(request)
if err != nil {
	return err
}
if resp.StatusCode >= 400 {
	// something goes wrong and you have to handle (e.g. returning an error to the user or logging the problem)
	log.Printf("api response: HTTP %d: %s", resp.StatusCode, resp.Body)
	// OR
	// return fmt.Errorf("api response: HTTP %d: %s", resp.StatusCode, resp.Body)
}
```

<a name="versions"></a>
## Versions

We follow the MAJOR.MINOR.PATCH versioning scheme as described by [SemVer.org](http://semver.org). Therefore, we recommend that you always pin (or vendor) the particular version you are working with to your code and never auto-update to the latest version. Especially when there is a MAJOR point release since that is guaranteed to be a breaking change. Changes are documented in the [CHANGELOG](CHANGELOG.md) and [releases](https://github.com/sendgrid/sendgrid-go/releases) section.

<a name="environment"></a>
## Environment Variables and Your Twilio SendGrid API Key

All of our examples assume you are using [environment variables](https://github.com/sendgrid/sendgrid-go#setup-environment-variables) to hold your Twilio SendGrid API key.

If you choose to add your Twilio SendGrid API key directly (not recommended):

`os.Getenv("SENDGRID_API_KEY")`

becomes

`"SENDGRID_API_KEY"`

In the first case, SENDGRID_API_KEY is in reference to the name of the environment variable, while the second case references the actual Twilio SendGrid API Key.

<a name="request-body"></a>
## Viewing the Request Body

When debugging or testing, it may be useful to examine the raw request body to compare against the [documented format](https://sendgrid.com/docs/API_Reference/api_v3.html).

You can do this right before you call `response, err := client.Send(message)` like so:

```go
fmt.Println(string(mail.GetRequestBody(message)))
```

<a name="signed-webhooks"></a>
## Signed Webhook Verification

Twilio SendGrid's Event Webhook will notify a URL via HTTP POST with information about events that occur as your mail is processed. [This](https://docs.sendgrid.com/for-developers/tracking-events/getting-started-event-webhook-security-features) article covers all you need to know to secure the Event Webhook, allowing you to verify that incoming requests originate from Twilio SendGrid. The sendgrid-go library can help you verify these Signed Event Webhooks.

You can find the end-to-end usage example [here](helpers/eventwebhook/README.md) and the tests [here](helpers/eventwebhook/eventwebhook_test.go). 
If you are still having trouble getting the validation to work, follow the following instructions:
- Be sure to use the *raw* payload for validation
- Be sure to include a trailing carriage return and newline in your payload
- In case of multi-event webhooks, make sure you include the trailing newline and carriage return after *each* event
