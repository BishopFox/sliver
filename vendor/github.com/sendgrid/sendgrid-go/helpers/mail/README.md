**This helper allows you to quickly and easily build a Mail object for sending email through Twilio SendGrid.**

## Dependencies

- [rest](https://github.com/sendgrid/rest)

# Quick Start

Run the [example](../../examples/helpers/mail/example.go) (make sure you have set your environment variable to include your SENDGRID_API_KEY).

```bash
go run examples/helpers/mail/example.go
```

## Usage

- See the [example](../../examples/helpers/mail/example.go) for a complete working example.
- [Documentation](https://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/index.html)

## Test

```bash
go test ./... -v
```

or

```bash
cd helpers/mail
go test -v
```
