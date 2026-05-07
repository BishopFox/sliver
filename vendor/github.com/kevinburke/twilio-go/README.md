# twilio-go

A client for accessing the Twilio API with several nice features:

- Easy-to-use helpers for purchasing phone numbers and sending MMS messages

- E.164 support, times that are parsed into a time.Time, and other smart types.

- Finer grained control over timeouts with a Context, and the library uses
  wall-clock HTTP timeouts, not socket timeouts.

- Easy debugging network traffic by setting DEBUG_HTTP_TRAFFIC=true in your
  environment.

- Easily find calls and messages that occurred between a particular
set of `time.Time`s, down to the nanosecond, with GetCallsInRange /
GetMessagesInRange.

- It's clear when the library will make a network request, there are no
unexpected latency spikes when paging from one resource to the next.

- Uses threads to fetch resources concurrently; for example, has methods to
fetch all Media for a Message concurrently.

- Usable, [one sentence descriptions of Alerts][alert-descriptions].

[alert-descriptions]: https://godoc.org/github.com/kevinburke/twilio-go#Alert.Description

Here are some example use cases:

```go
const sid = "AC123"
const token = "456bef"

client := twilio.NewClient(sid, token, nil)

// Send a message
msg, err := client.Messages.SendMessage("+14105551234", "+14105556789", "Sent via go :) ✓", nil)

// Start a phone call
var callURL, _ = url.Parse("https://kevin.burke.dev/zombo/zombocom.mp3")
call, err := client.Calls.MakeCall("+14105551234", "+14105556789", callURL)

// Buy a number
number, err := client.IncomingNumbers.BuyNumber("+14105551234")

// Get all calls from a number
data := url.Values{}
data.Set("From", "+14105551234")
callPage, err := client.Calls.GetPage(context.TODO(), data)

// Iterate over calls
iterator := client.Calls.GetPageIterator(url.Values{})
for {
    page, err := iterator.Next(context.TODO())
    if err == twilio.NoMoreResults {
        break
    }
    fmt.Println("start", page.Start)
}
```

A [complete documentation reference can be found at
godoc.org](https://godoc.org/github.com/kevinburke/twilio-go).

## In Production

twilio-go is being used by the following applications:

- [Logrole][logrole], an open source Twilio log viewer that's faster than the API.

Using twilio-go in production? [Let me know!](mailto:kevin@burke.services)

[logrole]: https://github.com/kevinburke/logrole

## Supported API's

The API is unlikely to change, and currently covers these resources:

- Alerts
- Applications
- Calls
- Conferences
- Faxes
- Incoming Phone Numbers
- Available Phone Numbers
- Keys
- Messages
- Media
- Monitor
- Outgoing Caller ID's
- Pricing
- Queues
- Recordings
- Task Router
  - Activities
  - TaskQueues
  - Workers
  - Workflows
- Transcriptions
- Wireless
- Voice Insights
- Access Tokens for IPMessaging, Video and Programmable Voice SDK

### Error Parsing

If the twilio-go client gets an error from the Twilio API, we attempt to convert
it to a [`rest.Error`][rest.error] before returning. Here's an example 404.

[rest.error]: https://godoc.org/github.com/kevinburke/rest#Error

```
&rest.Error{
    Title: "The requested resource ... was not found",
    ID: "20404",
    Detail: "",
    Instance: "",
    Type: "https://www.twilio.com/docs/errors/20404",
    StatusCode: 404
}
```

Not all errors will be a `rest.Error` however - HTTP timeouts, canceled
context.Contexts, and JSON parse errors (HTML error pages, bad gateway
responses from proxies) may also be returned as plain Go errors.

### Twiml Generation

There are no plans to support Twiml generation in this library. It may be
more readable and maintainable to manually write the XML involved in a Twiml
response.

### Errata

- Media URL's used to be returned over HTTP. twilio-go rewrites these URL's to be
  HTTPS before returning them to you.

- A subset of Notifications returned code 4107, which doesn't exist. These
  notifications should have error code 14107. We rewrite the error code
  internally before returning it to you.

- The only provided API for filtering calls or messages by date grabs all
messages for an entire day, and the day ranges are only available for UTC. Use
GetCallsInRange or GetMessagesInRange to do timezone-aware, finer-grained date
filtering.

- You can get Alerts for a given Call or MMS by passing `ResourceSid=CA123` as
a filter to Alerts.GetPage. This functionality is not documented in the API.

## Consulting

I'm available for hire, for Twilio work or general-purpose engineering. For more
on what I can do for your company, see here: https://burke.services/twilio.html.
Contact me: kevin@burke.services

## Donating

Donations free up time to review pull requests, respond to bug reports, and
add new features. In the absence of donations there are no guarantees about
timeliness for reviewing or responding to proposed changes; I don't get paid
by anyone else to work on this. You can send donations via Github's "Sponsor"
mechanism or Paypal's "Send Money" feature to kev@inburke.com. Donations are not
tax deductible in the USA.

[zero-results]: https://github.com/saintpete/twilio-go/issues/8
