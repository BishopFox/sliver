# WhatsApp

## IMPORTANT

### Broken state; see [#274](https://github.com/nikoksr/notify/issues/274)

Since our previous WhatsApp client library was abandoned, we had to switch to a new one. Unfortunately, the new library
is not yet ready for production use. We are working on a solution, but it will take some time.

The broken client library caused our CI pipeline to break and for that reason we decided to turn the WhatsApp service
into a no-op service. This means that the WhatsApp service will not send any notifications, but it will not return any
errors either. This is a temporary solution until we have a working client library. We chose no-op over a hard error to
prevent breaking existing applications.

