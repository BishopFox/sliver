This documentation is based on our [OAI specification](https://github.com/sendgrid/sendgrid-oai).

# INITIALIZATION

```go
package main

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"log"
	"os"
)

apiKey := os.Getenv("SENDGRID_API_KEY")
host := "https://api.sendgrid.com"
```

# Table of Contents

* [ACCESS SETTINGS](#access-settings)
* [ALERTS](#alerts)
* [API KEYS](#api-keys)
* [ASM](#asm)
* [BROWSERS](#browsers)
* [CAMPAIGNS](#campaigns)
* [CATEGORIES](#categories)
* [CLIENTS](#clients)
* [CONTACTDB](#contactdb)
* [DEVICES](#devices)
* [EMAIL ACTIVITY](#email-activity)
* [GEO](#geo)
* [IPS](#ips)
* [MAIL](#mail)
* [MAIL SETTINGS](#mail-settings)
* [MAILBOX PROVIDERS](#mailbox-providers)
* [PARTNER SETTINGS](#partner-settings)
* [SCOPES](#scopes)
* [SENDERS](#senders)
* [SENDER AUTHENTICATION](#sender-authentication)
* [STATS](#stats)
* [SUBUSERS](#subusers)
* [SUPPRESSION](#suppression)
* [TEMPLATES](#templates)
* [TRACKING SETTINGS](#tracking-settings)
* [ON-BEHALF OF SUBUSER](#on-behalf-of)
* [USER](#user)

<a name="access-settings"></a>
# ACCESS SETTINGS

## Retrieve all recent access attempts

**This endpoint allows you to retrieve a list of all of the IP addresses that recently attempted to access your account either through the User Interface or the API.**

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### GET /access_settings/activity

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/activity", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add one or more IPs to the whitelist

**This endpoint allows you to add one or more IP addresses to your IP whitelist.**

When adding an IP to your whitelist, include the IP address in an array. You can whitelist one IP at a time, or you can whitelist multiple IPs at once.

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### POST /access_settings/whitelist

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/whitelist", host)
request.Method = "POST"
request.Body = []byte(` {
  "ips": [
    {
      "ip": "192.168.1.1"
    },
    {
      "ip": "192.*.*.*"
    },
    {
      "ip": "192.168.1.3/32"
    }
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a list of currently whitelisted IPs

**This endpoint allows you to retrieve a list of IP addresses that are currently whitelisted.**

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### GET /access_settings/whitelist

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/whitelist", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Remove one or more IPs from the whitelist

**This endpoint allows you to remove one or more IPs from your IP whitelist.**

You can remove one IP at a time, or you can remove multiple IP addresses.

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### DELETE /access_settings/whitelist

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/whitelist", host)
request.Method = "DELETE"
request.Body = []byte(` {
  "ids": [
    1,
    2,
    3
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific whitelisted IP

**This endpoint allows you to retrieve a specific IP address that has been whitelisted.**

You must include the ID for the specific IP address you want to retrieve in your call.

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### GET /access_settings/whitelist/{rule_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/whitelist/{rule_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Remove a specific IP from the whitelist

**This endpoint allows you to remove a specific IP address from your IP whitelist.**

When removing a specific IP address from your whitelist, you must include the ID in your call.

IP Access Management allows you to control which IP addresses can be used to access your account, either through the User Interface or the API. There is no limit to the number of IP addresses that you can add to your whitelist. It is possible to remove your own IP address from the whitelist, thus preventing yourself from accessing your account.

For more information, please see our [User Guide](http://sendgrid.com/docs/User_Guide/Settings/ip_access_management.html).

### DELETE /access_settings/whitelist/{rule_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/access_settings/whitelist/{rule_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="alerts"></a>
# ALERTS

## Create a new Alert

**This endpoint allows you to create a new alert.**

Alerts allow you to specify an email address to receive notifications regarding your email usage or statistics.
* Usage alerts allow you to set the threshold at which an alert will be sent.
* Stats notifications allow you to set how frequently you would like to receive email statistics reports. For example, "daily", "weekly", or "monthly".

For more information about alerts, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/alerts.html).

### POST /alerts

```go
request := sendgrid.GetRequest(apiKey, "/v3/alerts", host)
request.Method = "POST"
request.Body = []byte(` {
  "email_to": "example@example.com",
  "frequency": "daily",
  "type": "stats_notification"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all alerts

**This endpoint allows you to retrieve all of your alerts.**

Alerts allow you to specify an email address to receive notifications regarding your email usage or statistics.
* Usage alerts allow you to set the threshold at which an alert will be sent.
* Stats notifications allow you to set how frequently you would like to receive email statistics reports. For example, "daily", "weekly", or "monthly".

For more information about alerts, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/alerts.html).

### GET /alerts

```go
request := sendgrid.GetRequest(apiKey, "/v3/alerts", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update an alert

**This endpoint allows you to update an alert.**

Alerts allow you to specify an email address to receive notifications regarding your email usage or statistics.
* Usage alerts allow you to set the threshold at which an alert will be sent.
* Stats notifications allow you to set how frequently you would like to receive email statistics reports. For example, "daily", "weekly", or "monthly".

For more information about alerts, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/alerts.html).

### PATCH /alerts/{alert_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/alerts/{alert_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "email_to": "example@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific alert

**This endpoint allows you to retrieve a specific alert.**

Alerts allow you to specify an email address to receive notifications regarding your email usage or statistics.
* Usage alerts allow you to set the threshold at which an alert will be sent.
* Stats notifications allow you to set how frequently you would like to receive email statistics reports. For example, "daily", "weekly", or "monthly".

For more information about alerts, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/alerts.html).

### GET /alerts/{alert_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/alerts/{alert_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete an alert

**This endpoint allows you to delete an alert.**

Alerts allow you to specify an email address to receive notifications regarding your email usage or statistics.
* Usage alerts allow you to set the threshold at which an alert will be sent.
* Stats notifications allow you to set how frequently you would like to receive email statistics reports. For example, "daily", "weekly", or "monthly".

For more information about alerts, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/alerts.html).

### DELETE /alerts/{alert_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/alerts/{alert_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="api-keys"></a>
# API KEYS

## Create API keys

**This endpoint allows you to create a new random API Key for the user.**

A JSON request body containing a "name" property is required. If the number of maximum keys is reached, HTTP 403 will be returned.

There is a limit of 100 API Keys on your account.

The API Keys feature allows customers to be able to generate an API Key credential which can be used for authentication with the Twilio SendGrid v3 Web API or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).

See the [API Key Permissions List](https://sendgrid.com/docs/API_Reference/Web_API_v3/API_Keys/api_key_permissions_list.html) for a list of all available scopes.

### POST /api_keys

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys", host)
request.Method = "POST"
request.Body = []byte(` {
  "name": "My API Key",
  "sample": "data",
  "scopes": [
    "mail.send",
    "alerts.create",
    "alerts.read"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all API Keys belonging to the authenticated user

**This endpoint allows you to retrieve all API Keys that belong to the authenticated user.**

The API Keys feature allows customers to be able to generate an API Key credential which can be used for authentication with the Twilio SendGrid v3 Web API or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).

### GET /api_keys

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update the name & scopes of an API Key

**This endpoint allows you to update the name and scopes of a given API key.**

A JSON request body with a "name" property is required.
Most provide the list of all the scopes an API key should have.

The API Keys feature allows customers to be able to generate an API Key credential which can be used for authentication with the Twilio SendGrid v3 Web API or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).


### PUT /api_keys/{api_key_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys/{api_key_id}", host)
request.Method = "PUT"
request.Body = []byte(` {
  "name": "A New Hope",
  "scopes": [
    "user.profile.read",
    "user.profile.update"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update API keys

**This endpoint allows you to update the name of an existing API Key.**

A JSON request body with a "name" property is required.

The API Keys feature allows customers to be able to generate an API Key credential which can be used for authentication with the Twilio SendGrid v3 Web API or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).

## URI Parameters

| URI Parameter   | Type  | Required?  | Description  |
|---|---|---|---|
|api_key_id |string | required | The ID of the API Key you are updating.|

### PATCH /api_keys/{api_key_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys/{api_key_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "name": "A New Hope"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve an existing API Key

**This endpoint allows you to retrieve a single API key.**

If the API Key ID does not exist an HTTP 404 will be returned.

### GET /api_keys/{api_key_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys/{api_key_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete API keys

**This endpoint allows you to revoke an existing API Key**

Authentications using this API Key will fail after this request is made, with some small propagation delay. If the API Key ID does not exist an HTTP 404 will be returned.

The API Keys feature allows customers to be able to generate an API Key credential which can be used for authentication with the Twilio SendGrid v3 Web API or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).

## URI Parameters

| URI Parameter   | Type  | Required?  | Description  |
|---|---|---|---|
|api_key_id |string | required | The ID of the API Key you are deleting.|

### DELETE /api_keys/{api_key_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/api_keys/{api_key_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="asm"></a>
# ASM

## Create a new suppression group

**This endpoint allows you to create a new suppression group.**

Suppression groups, or unsubscribe groups, are specific types or categories of email that you would like your recipients to be able to unsubscribe from. For example: Daily Newsletters, Invoices, System Alerts.

The **name** and **description** of the unsubscribe group will be visible by recipients when they are managing their subscriptions.

Each user can create up to 25 different suppression groups.

### POST /asm/groups

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups", host)
request.Method = "POST"
request.Body = []byte(` {
  "description": "Suggestions for products our users might like.",
  "is_default": true,
  "name": "Product Suggestions"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve information about multiple suppression groups

**This endpoint allows you to retrieve information about multiple suppression groups.**

This endpoint will return information for each group ID that you include in your request. To add a group ID to your request, simply append `&id=` followed by the group ID.

Suppressions are a list of email addresses that will not receive content sent under a given [group](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html).

Suppression groups, or [unsubscribe groups](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html), allow you to label a category of content that you regularly send. This gives your recipients the ability to opt out of a specific set of your email. For example, you might define a group for your transactional email, and one for your marketing email so that your users can continue receiving your transactional email without having to receive your marketing content.

### GET /asm/groups

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a suppression group.

**This endpoint allows you to update or change a suppression group.**

Suppression groups, or unsubscribe groups, are specific types or categories of email that you would like your recipients to be able to unsubscribe from. For example: Daily Newsletters, Invoices, System Alerts.

The **name** and **description** of the unsubscribe group will be visible by recipients when they are managing their subscriptions.

Each user can create up to 25 different suppression groups.

### PATCH /asm/groups/{group_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "description": "Suggestions for items our users might like.",
  "id": 103,
  "name": "Item Suggestions"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Get information on a single suppression group.

**This endpoint allows you to retrieve a single suppression group.**

Suppression groups, or unsubscribe groups, are specific types or categories of email that you would like your recipients to be able to unsubscribe from. For example: Daily Newsletters, Invoices, System Alerts.

The **name** and **description** of the unsubscribe group will be visible by recipients when they are managing their subscriptions.

Each user can create up to 25 different suppression groups.

### GET /asm/groups/{group_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a suppression group.

**This endpoint allows you to delete a suppression group.**

You can only delete groups that have not been attached to sent mail in the last 60 days. If a recipient uses the "one-click unsubscribe" option on an email associated with a deleted group, that recipient will be added to the global suppression list.

Suppression groups, or unsubscribe groups, are specific types or categories of email that you would like your recipients to be able to unsubscribe from. For example: Daily Newsletters, Invoices, System Alerts.

The **name** and **description** of the unsubscribe group will be visible by recipients when they are managing their subscriptions.

Each user can create up to 25 different suppression groups.

### DELETE /asm/groups/{group_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add suppressions to a suppression group

**This endpoint allows you to add email addresses to an unsubscribe group.**

If you attempt to add suppressions to a group that has been deleted or does not exist, the suppressions will be added to the global suppressions list.

Suppressions are recipient email addresses that are added to [unsubscribe groups](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html). Once a recipient's address is on the suppressions list for an unsubscribe group, they will not receive any emails that are tagged with that unsubscribe group.

### POST /asm/groups/{group_id}/suppressions

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}/suppressions", host)
request.Method = "POST"
request.Body = []byte(` {
  "recipient_emails": [
    "test1@example.com",
    "test2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all suppressions for a suppression group

**This endpoint allows you to retrieve all suppressed email addresses belonging to the given group.**

Suppressions are recipient email addresses that are added to [unsubscribe groups](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html). Once a recipient's address is on the suppressions list for an unsubscribe group, they will not receive any emails that are tagged with that unsubscribe group.

### GET /asm/groups/{group_id}/suppressions

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}/suppressions", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Search for suppressions within a group

**This endpoint allows you to search a suppression group for multiple suppressions.**

When given a list of email addresses and a group ID, this endpoint will return only the email addresses that have been unsubscribed from the given group.

Suppressions are a list of email addresses that will not receive content sent under a given [group](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html).

### POST /asm/groups/{group_id}/suppressions/search

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}/suppressions/search", host)
request.Method = "POST"
request.Body = []byte(` {
  "recipient_emails": [
    "exists1@example.com",
    "exists2@example.com",
    "doesnotexists@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a suppression from a suppression group

**This endpoint allows you to remove a suppressed email address from the given suppression group.**

Suppressions are recipient email addresses that are added to [unsubscribe groups](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html). Once a recipient's address is on the suppressions list for an unsubscribe group, they will not receive any emails that are tagged with that unsubscribe group.

### DELETE /asm/groups/{group_id}/suppressions/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/groups/{group_id}/suppressions/{email}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all suppressions

**This endpoint allows you to retrieve a list of all suppressions.**

Suppressions are a list of email addresses that will not receive content sent under a given [group](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html).

### GET /asm/suppressions

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/suppressions", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add recipient addresses to the global suppression group.

**This endpoint allows you to add one or more email addresses to the global suppressions group.**

A global suppression (or global unsubscribe) is an email address of a recipient who does not want to receive any of your messages. A globally suppressed recipient will be removed from any email you send. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/global_unsubscribes.html).

### POST /asm/suppressions/global

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/suppressions/global", host)
request.Method = "POST"
request.Body = []byte(` {
  "recipient_emails": [
    "test1@example.com",
    "test2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Global Suppression

**This endpoint allows you to retrieve a global suppression. You can also use this endpoint to confirm if an email address is already globally suppressed.**

If the email address you include in the URL path parameter `{email}` is already globally suppressed, the response will include that email address. If the address you enter for `{email}` is not globally suppressed, an empty JSON object `{}` will be returned.

A global suppression (or global unsubscribe) is an email address of a recipient who does not want to receive any of your messages. A globally suppressed recipient will be removed from any email you send. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/global_unsubscribes.html).

### GET /asm/suppressions/global/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/suppressions/global/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Global Suppression

**This endpoint allows you to remove an email address from the global suppressions group.**

A global suppression (or global unsubscribe) is an email address of a recipient who does not want to receive any of your messages. A globally suppressed recipient will be removed from any email you send. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/global_unsubscribes.html).

### DELETE /asm/suppressions/global/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/suppressions/global/{email}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all suppression groups for an email address

**This endpoint returns the list of all groups that the given email address has been unsubscribed from.**

Suppressions are a list of email addresses that will not receive content sent under a given [group](https://sendgrid.com/docs/API_Reference/Web_API_v3/Suppression_Management/groups.html).

### GET /asm/suppressions/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/asm/suppressions/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="browsers"></a>
# BROWSERS

## Retrieve email statistics by browser.

**This endpoint allows you to retrieve your email statistics segmented by browser type.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /browsers/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/browsers/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["aggregated_by"] = "day"
queryParams["browsers"] = "test_string"
queryParams["limit"] = "test_string"
queryParams["offset"] = "test_string"
queryParams["start_date"] = "2016-01-01"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="campaigns"></a>
# CAMPAIGNS

## Create a Campaign

**This endpoint allows you to create a campaign.**

Our Marketing Campaigns API lets you create, manage, send, and schedule campaigns.

Note: In order to send or schedule the campaign, you will be required to provide a subject, sender ID, content (we suggest both HTML and plain text), and at least one list or segment ID. This information is not required when you create a campaign.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### POST /campaigns

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns", host)
request.Method = "POST"
request.Body = []byte(` {
  "categories": [
    "spring line"
  ],
  "custom_unsubscribe_url": "",
  "html_content": "<html><head><title></title></head><body><p>Check out our spring line!</p></body></html>",
  "ip_pool": "marketing",
  "list_ids": [
    110,
    124
  ],
  "plain_content": "Check out our spring line!",
  "segment_ids": [
    110
  ],
  "sender_id": 124451,
  "subject": "New Products for Spring!",
  "suppression_group_id": 42,
  "title": "March Newsletter"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all Campaigns

**This endpoint allows you to retrieve a list of all of your campaigns.**

Returns campaigns in reverse order they were created (newest first).

Returns an empty array if no campaigns exist.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### GET /campaigns

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a Campaign

Update a campaign. This is especially useful if you only set up the campaign using POST /campaigns, but didn't set many of the parameters.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### PATCH /campaigns/{campaign_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "categories": [
    "summer line"
  ],
  "html_content": "<html><head><title></title></head><body><p>Check out our summer line!</p></body></html>",
  "plain_content": "Check out our summer line!",
  "subject": "New Products for Summer!",
  "title": "May Newsletter"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a single campaign

**This endpoint allows you to retrieve a specific campaign.**

Our Marketing Campaigns API lets you create, manage, send, and schedule campaigns.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### GET /campaigns/{campaign_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Campaign

**This endpoint allows you to delete a specific campaign.**

Our Marketing Campaigns API lets you create, manage, send, and schedule campaigns.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### DELETE /campaigns/{campaign_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a Scheduled Campaign

**This endpoint allows to you change the scheduled time and date for a campaign to be sent.**

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### PATCH /campaigns/{campaign_id}/schedules

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "send_at": 1489451436
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Schedule a Campaign

**This endpoint allows you to schedule a specific date and time for your campaign to be sent.**

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### POST /campaigns/{campaign_id}/schedules

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules", host)
request.Method = "POST"
request.Body = []byte(` {
  "send_at": 1489771528
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## View Scheduled Time of a Campaign

**This endpoint allows you to retrieve the date and time that the given campaign has been scheduled to be sent.**

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### GET /campaigns/{campaign_id}/schedules

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Unschedule a Scheduled Campaign

**This endpoint allows you to unschedule a campaign that has already been scheduled to be sent.**

A successful unschedule will return a 204.
If the specified campaign is in the process of being sent, the only option is to cancel (a different method).

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### DELETE /campaigns/{campaign_id}/schedules

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Send a Campaign

**This endpoint allows you to immediately send a campaign at the time you make the API call.**

Normally a POST would have a request body, but since this endpoint is telling us to send a resource that is already created, a request body is not needed.

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### POST /campaigns/{campaign_id}/schedules/now

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules/now", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Send a Test Campaign

**This endpoint allows you to send a test campaign.**

To send to multiple addresses, use an array for the JSON "to" value ["one@address","two@address"]

For more information:

* [User Guide > Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html)

### POST /campaigns/{campaign_id}/schedules/test

```go
request := sendgrid.GetRequest(apiKey, "/v3/campaigns/{campaign_id}/schedules/test", host)
request.Method = "POST"
request.Body = []byte(` {
  "to": "your.email@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="categories"></a>
# CATEGORIES

## Retrieve all categories

**This endpoint allows you to retrieve a list of all of your categories.**

Categories can help organize your email analytics by enabling you to tag emails by type or broad topic. You can define your own custom categories. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/categories.html).

### GET /categories

```go
request := sendgrid.GetRequest(apiKey, "/v3/categories", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["category"] = "test_string"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Email Statistics for Categories

**This endpoint allows you to retrieve all of your email statistics for each of your categories.**

If you do not define any query parameters, this endpoint will return a sum for each category in groups of 10.

Categories allow you to group your emails together according to broad topics that you define. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/categories.html).

### GET /categories/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/categories/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["categories"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve sums of email stats for each category [Needs: Stats object defined, has category ID?]

**This endpoint allows you to retrieve the total sum of each email statistic for every category over the given date range.**

If you do not define any query parameters, this endpoint will return a sum for each category in groups of 10.

Categories allow you to group your emails together according to broad topics that you define. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/categories.html).

### GET /categories/stats/sums

```go
request := sendgrid.GetRequest(apiKey, "/v3/categories/stats/sums", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["sort_by_metric"] = "test_string"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["sort_by_direction"] = "asc"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="clients"></a>
# CLIENTS

## Retrieve email statistics by client type.

**This endpoint allows you to retrieve your email statistics segmented by client type.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /clients/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/clients/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["aggregated_by"] = "day"
queryParams["start_date"] = "2016-01-01"
queryParams["end_date"] = "2016-04-01"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve stats by a specific client type.

**This endpoint allows you to retrieve your email statistics segmented by a specific client type.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

## Available Client Types
- phone
- tablet
- webmail
- desktop

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /clients/{client_type}/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/clients/{client_type}/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["aggregated_by"] = "day"
queryParams["start_date"] = "2016-01-01"
queryParams["end_date"] = "2016-04-01"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="contactdb"></a>
# CONTACTDB

## Create a Custom Field

**This endpoint allows you to create a custom field.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### POST /contactdb/custom_fields

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/custom_fields", host)
request.Method = "POST"
request.Body = []byte(` {
  "name": "pet",
  "type": "text"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all custom fields

**This endpoint allows you to retrieve all custom fields.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### GET /contactdb/custom_fields

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/custom_fields", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Custom Field

**This endpoint allows you to retrieve a custom field by ID.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### GET /contactdb/custom_fields/{custom_field_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/custom_fields/{custom_field_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Custom Field

**This endpoint allows you to delete a custom field by ID.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### DELETE /contactdb/custom_fields/{custom_field_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/custom_fields/{custom_field_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create a List

**This endpoint allows you to create a list for your recipients.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### POST /contactdb/lists

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists", host)
request.Method = "POST"
request.Body = []byte(` {
  "name": "your list name"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all lists

**This endpoint allows you to retrieve all of your recipient lists. If you don't have any lists, an empty array will be returned.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/lists

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete Multiple lists

**This endpoint allows you to delete multiple recipient lists.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### DELETE /contactdb/lists

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists", host)
request.Method = "DELETE"
request.Body = []byte(` [
  1,
  2,
  3,
  4
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a List

**This endpoint allows you to update the name of one of your recipient lists.**


The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### PATCH /contactdb/lists/{list_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "name": "newlistname"
}`)
queryParams := make(map[string]string)
queryParams["list_id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a single list

This endpoint allows you to retrieve a single recipient list.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/lists/{list_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["list_id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a List

**This endpoint allows you to delete a specific recipient list with the given ID.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### DELETE /contactdb/lists/{list_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}", host)
request.Method = "DELETE"
queryParams := make(map[string]string)
queryParams["delete_contacts"] = "true"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add Multiple Recipients to a List

**This endpoint allows you to add multiple recipients to a list.**

Adds existing recipients to a list, passing in the recipient IDs to add. Recipient IDs should be passed exactly as they are returned from recipient endpoints.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### POST /contactdb/lists/{list_id}/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}/recipients", host)
request.Method = "POST"
request.Body = []byte(` [
  "recipient_id1",
  "recipient_id2"
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all recipients on a List

**This endpoint allows you to retrieve all recipients on the list with the given ID.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/lists/{list_id}/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}/recipients", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["page"] = "1"
queryParams["page_size"] = "1"
queryParams["list_id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add a Single Recipient to a List

**This endpoint allows you to add a single recipient to a list.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### POST /contactdb/lists/{list_id}/recipients/{recipient_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}/recipients/{recipient_id}", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Single Recipient from a Single List

**This endpoint allows you to delete a single recipient from a list.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### DELETE /contactdb/lists/{list_id}/recipients/{recipient_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/lists/{list_id}/recipients/{recipient_id}", host)
request.Method = "DELETE"
queryParams := make(map[string]string)
queryParams["recipient_id"] = "1"
queryParams["list_id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Recipient

**This endpoint allows you to update one or more recipients.**

The body of an API call to this endpoint must include an array of one or more recipient objects.

It is of note that you can add custom field data as parameters on recipient objects. We have provided an example using some of the default custom fields Twilio SendGrid provides.

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### PATCH /contactdb/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients", host)
request.Method = "PATCH"
request.Body = []byte(` [
  {
    "email": "jones@example.com",
    "first_name": "Guy",
    "last_name": "Jones"
  }
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add recipients

**This endpoint allows you to add a Marketing Campaigns recipient.**

It is of note that you can add custom field data as a parameter on this endpoint. We have provided an example using some of the default custom fields Twilio SendGrid provides.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### POST /contactdb/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients", host)
request.Method = "POST"
request.Body = []byte(` [
  {
    "age": 25,
    "email": "example@example.com",
    "first_name": "",
    "last_name": "User"
  },
  {
    "age": 25,
    "email": "example2@example.com",
    "first_name": "Example",
    "last_name": "User"
  }
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve recipients

**This endpoint allows you to retrieve all of your Marketing Campaigns recipients.**

Batch deletion of a page makes it possible to receive an empty page of recipients before reaching the end of
the list of recipients. To avoid this issue; iterate over pages until a 404 is retrieved.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["page"] = "1"
queryParams["page_size"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete Recipient

**This endpoint allows you to delete one or more recipients.**

The body of an API call to this endpoint must include an array of recipient IDs of the recipients you want to delete.

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### DELETE /contactdb/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients", host)
request.Method = "DELETE"
request.Body = []byte(` [
  "recipient_id1",
  "recipient_id2"
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve the count of billable recipients

**This endpoint allows you to retrieve the number of Marketing Campaigns recipients that you will be billed for.**

You are billed for marketing campaigns based on the highest number of recipients you have had in your account at one time. This endpoint will allow you to know the current billable count value.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/recipients/billable_count

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/billable_count", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Count of Recipients

**This endpoint allows you to retrieve the total number of Marketing Campaigns recipients.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### GET /contactdb/recipients/count

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/count", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve recipients matching search criteria

**This endpoint allows you to perform a search on all of your Marketing Campaigns recipients.**

field_name:

* is a variable that is substituted for your actual custom field name from your recipient.
* Text fields must be url-encoded. Date fields are searchable only by unix timestamp (e.g. 2/2/2015 becomes 1422835200)
* If field_name is a 'reserved' date field, such as created_at or updated_at, the system will internally convert
your epoch time to a date range encompassing the entire day. For example, an epoch time of 1422835600 converts to
Mon, 02 Feb 2015 00:06:40 GMT, but internally the system will search from Mon, 02 Feb 2015 00:00:00 GMT through
Mon, 02 Feb 2015 23:59:59 GMT.

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### GET /contactdb/recipients/search

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/search", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["{field_name}"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a single recipient

**This endpoint allows you to retrieve a single recipient by ID from your contact database.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/recipients/{recipient_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/{recipient_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Recipient

**This endpoint allows you to delete a single recipient with the given ID from your contact database.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### DELETE /contactdb/recipients/{recipient_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/{recipient_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve the lists that a recipient is on

**This endpoint allows you to retrieve the lists that a given recipient belongs to.**

Each recipient can be on many lists. This endpoint gives you all of the lists that any one recipient has been added to.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

### GET /contactdb/recipients/{recipient_id}/lists

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/recipients/{recipient_id}/lists", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve reserved fields

**This endpoint allows you to list all fields that are reserved and can't be used for custom field names.**

The contactdb is a database of your contacts for [Twilio SendGrid Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html).

### GET /contactdb/reserved_fields

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/reserved_fields", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create a Segment

**This endpoint allows you to create a segment.**

All recipients in your contactdb will be added or removed automatically depending on whether they match the criteria for this segment.

List Id:

* Send this to segment from an existing list
* Don't send this in order to segment from your entire contactdb.

Valid operators for create and update depend on the type of the field you are segmenting:

* **Dates:** "eq", "ne", "lt" (before), "gt" (after)
* **Text:** "contains", "eq" (is - matches the full field), "ne" (is not - matches any field where the entire field is not the condition value)
* **Numbers:** "eq", "lt", "gt"
* **Email Clicks and Opens:** "eq" (opened), "ne" (not opened)

Segment conditions using "eq" or "ne" for email clicks and opens should provide a "field" of either *clicks.campaign_identifier* or *opens.campaign_identifier*. The condition value should be a string containing the id of a completed campaign.

Segments may contain multiple conditions, joined by an "and" or "or" in the "and_or" field. The first condition in the conditions list must have an empty "and_or", and subsequent conditions must all specify an "and_or".

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### POST /contactdb/segments

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments", host)
request.Method = "POST"
request.Body = []byte(` {
  "conditions": [
    {
      "and_or": "",
      "field": "last_name",
      "operator": "eq",
      "value": "Miller"
    },
    {
      "and_or": "and",
      "field": "last_clicked",
      "operator": "gt",
      "value": "01/02/2015"
    },
    {
      "and_or": "or",
      "field": "clicks.campaign_identifier",
      "operator": "eq",
      "value": "513"
    }
  ],
  "list_id": 4,
  "name": "Last Name Miller"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all segments

**This endpoint allows you to retrieve all of your segments.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### GET /contactdb/segments

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a segment

**This endpoint allows you to update a segment.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### PATCH /contactdb/segments/{segment_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments/{segment_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "conditions": [
    {
      "and_or": "",
      "field": "last_name",
      "operator": "eq",
      "value": "Miller"
    }
  ],
  "list_id": 5,
  "name": "The Millers"
}`)
queryParams := make(map[string]string)
queryParams["segment_id"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a segment

**This endpoint allows you to retrieve a single segment with the given ID.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### GET /contactdb/segments/{segment_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments/{segment_id}", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["segment_id"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a segment

**This endpoint allows you to delete a segment from your recipient's database.**

You also have the option to delete all the contacts from your Marketing Campaigns recipient database who were in this segment.

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### DELETE /contactdb/segments/{segment_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments/{segment_id}", host)
request.Method = "DELETE"
queryParams := make(map[string]string)
queryParams["delete_contacts"] = "true"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve recipients on a segment

**This endpoint allows you to retrieve all of the recipients in a segment with the given ID.**

The Contacts API helps you manage your [Marketing Campaigns](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/index.html) recipients.

For more information about segments in Marketing Campaigns, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/lists.html#-Create-a-Segment).

### GET /contactdb/segments/{segment_id}/recipients

```go
request := sendgrid.GetRequest(apiKey, "/v3/contactdb/segments/{segment_id}/recipients", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["page"] = "1"
queryParams["page_size"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="devices"></a>
# DEVICES

## Retrieve email statistics by device type.

**This endpoint allows you to retrieve your email statistics segmented by the device type.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

## Available Device Types
| **Device** | **Description** | **Example** |
|---|---|---|
| Desktop | Email software on desktop computer. | I.E., Outlook, Sparrow, or Apple Mail. |
| Webmail |	A web-based email client. | I.E., Yahoo, Google, AOL, or Outlook.com. |
| Phone | A smart phone. | iPhone, Android, Blackberry, etc.
| Tablet | A tablet computer. | iPad, android based tablet, etc. |
| Other | An unrecognized device. |

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /devices/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/devices/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["end_date"] = "2016-04-01"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="email-activity"></a>
# Email Activity

## Filter all messages

> In order to gain access to the Email Activity Feed API, you must purchase [additional email activity history](https://app.sendgrid.com/settings/billing/addons/email_activity).

Filter all messages to search your Email Activity.

Queries may need to be [URL encoded](https://meyerweb.com/eric/tools/dencoder/). URL encoding depends on how you're using the API - if you are trying it out here, or using one of the Libraries, we handle the encoding for you. If you are using cURL, or your own implementation, you probably need to encode it.

Queries have this format:

`query={query_type}="{query_content}"`

encoded, this would look like this:

`query=type%3D%22query_content%22`

for example:

Filter by a specific email - `query=to_email%3D%22example%40example.com%22`

Filter by subject line - `query=subject%3d%22A%20Great%20Subject%22`

You can filter by other operators besides `=`. We also accept `!=`, `<`, and `>`.

For a tutorial on how to get started, check out [Getting Started with the Email Activity API](https://sendgrid.com/docs/API_Reference/Web_API_v3/Tutorials/getting_started_email_activity_api.html).

For information about building combined queries, see [Building compound Email Activity queries](https://sendgrid.com/docs/API_Reference/Web_API_v3/Tutorials/getting_started_email_activity_api.html#-Creating-compound-queries).

### GET /messages

```go
request := sendgrid.GetRequest(apiKey, "/v3/messages", host)
request.Method = "GET"

filterKey := "to_email"
filterOperator := url.QueryEscape("=")
filterValue := "testing@sendgrid.net"
filterValue = url.QueryEscape(fmt.Sprintf("\"%s\"", filterValue))

queryParams := make(map[string]string)
queryParams["query"] = fmt.Sprintf("%s%s%s", filterKey, filterOperator, filterValue)
queryParams["limit"] = "1"
request.QueryParams = queryParams

response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Filter messages by message ID

> In order to gain access to the Email Activity Feed API, you must purchase [additional email activity history](https://app.sendgrid.com/settings/billing/addons/email_activity).

Get all of the details about the specified message.

### GET /messages/{msg_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/messages/{msg_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Request a CSV

### POST /messages/download

> In order to gain access to the Email Activity Feed API, you must purchase [additional email activity history](https://app.sendgrid.com/settings/billing/addons/email_activity).

This request kicks of a process to generate a CSV file. When the file is generated, the email that is listed as the account owner gets an email that links out to the file that is ready for download. The link expires in 3 days.

The CSV fill contain the last 1 million messages. This endpoint will be rate limited to 1 request every 12 hours.

```go
request := sendgrid.GetRequest(apiKey, "/v3/messages/download", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Download CSV

### GET /messages/download/{download_uuid}

> In order to gain access to the Email Activity Feed API, you must purchase [additional email activity history](https://app.sendgrid.com/settings/billing/addons/email_activity).

Download the CSV that you requested with the POST Request a CSV.

When the file is generated, the email that is listed as the account owner gets an email that links out to the file that is ready for download. The link expires in 3 days.

The CSV fill contain the last 1 million messages. This endpoint will be rate limited to 1 request every 12 hours.

```go
request := sendgrid.GetRequest(apiKey, "/v3/messages/download/{download_uuid}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```
<a name="geo"></a>
# GEO

## Retrieve email statistics by country and state/province.

**This endpoint allows you to retrieve your email statistics segmented by country and state/province.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /geo/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/geo/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["country"] = "US"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="ips"></a>
# IPS

## Retrieve all IP addresses

**This endpoint allows you to retrieve a list of all assigned and unassigned IPs.**

The response includes warm up status, pools, assigned subusers, and authentication info. The start_date field corresponds to when warmup started for that IP.

A single IP address or a range of IP addresses may be dedicated to an account in order to send email for multiple domains. The reputation of this IP is based on the aggregate performance of all the senders who use it.

### GET /ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["subuser"] = "test_string"
queryParams["ip"] = "test_string"
queryParams["limit"] = "1"
queryParams["exclude_whitelabels"] = "true"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all assigned IPs

**This endpoint allows you to retrieve only assigned IP addresses.**

A single IP address or a range of IP addresses may be dedicated to an account in order to send email for multiple domains. The reputation of this IP is based on the aggregate performance of all the senders who use it.

### GET /ips/assigned

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/assigned", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create an IP pool.

**This endpoint allows you to create an IP pool.**

**Each user can create up to 10 different IP pools.**

IP Pools allow you to group your dedicated Twilio SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email. When sending marketing emails, specify that you want to use the marketing IP pool. This allows you to maintain separate reputations for your different email traffic.

IP pools can only be used with authenticated IP addresses.

If an IP pool is NOT specified for an email, it will use any IP available, including ones in pools.

### POST /ips/pools

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools", host)
request.Method = "POST"
request.Body = []byte(` {
  "name": "marketing"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all IP pools.

**This endpoint allows you to retrieve all of your IP pools.**

IP Pools allow you to group your dedicated Twilio SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email. When sending marketing emails, specify that you want to use the marketing IP pool. This allows you to maintain separate reputations for your different email traffic.

IP pools can only be used with authenticated IP addresses.

If an IP pool is NOT specified for an email, it will use any IP available, including ones in pools.

### GET /ips/pools

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update an IP pools name.

**This endpoint allows you to update the name of an IP pool.**

IP Pools allow you to group your dedicated Twilio SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email. When sending marketing emails, specify that you want to use the marketing IP pool. This allows you to maintain separate reputations for your different email traffic.

IP pools can only be used with authenticated IP addresses.

If an IP pool is NOT specified for an email, it will use any IP available, including ones in pools.

### PUT /ips/pools/{pool_name}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools/{pool_name}", host)
request.Method = "PUT"
request.Body = []byte(` {
  "name": "new_pool_name"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all IPs in a specified pool.

**This endpoint allows you to list all of the IP addresses that are in a specific IP pool.**

IP Pools allow you to group your dedicated Twilio SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email. When sending marketing emails, specify that you want to use the marketing IP pool. This allows you to maintain separate reputations for your different email traffic.

IP pools can only be used with authenticated IP addresses.

If an IP pool is NOT specified for an email, it will use any IP available, including ones in pools.

### GET /ips/pools/{pool_name}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools/{pool_name}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete an IP pool.

**This endpoint allows you to delete an IP pool.**

IP Pools allow you to group your dedicated Twilio SendGrid IP addresses together. For example, you could create separate pools for your transactional and marketing email. When sending marketing emails, specify that you want to use the marketing IP pool. This allows you to maintain separate reputations for your different email traffic.

IP pools can only be used with authenticated IP addresses.

If an IP pool is NOT specified for an email, it will use any IP available, including ones in pools.

### DELETE /ips/pools/{pool_name}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools/{pool_name}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add an IP address to a pool

**This endpoint allows you to add an IP address to an IP pool.**

You can add the same IP address to multiple pools. It may take up to 60 seconds for your IP address to be added to a pool after your request is made.

A single IP address or a range of IP addresses may be dedicated to an account in order to send email for multiple domains. The reputation of this IP is based on the aggregate performance of all the senders who use it.

### POST /ips/pools/{pool_name}/ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools/{pool_name}/ips", host)
request.Method = "POST"
request.Body = []byte(` {
  "ip": "0.0.0.0"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Remove an IP address from a pool.

**This endpoint allows you to remove an IP address from an IP pool.**

The same IP address can be added to multiple IP pools.

A single IP address or a range of IP addresses may be dedicated to an account in order to send email for multiple domains. The reputation of this IP is based on the aggregate performance of all the senders who use it.

### DELETE /ips/pools/{pool_name}/ips/{ip}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/pools/{pool_name}/ips/{ip}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add an IP to warmup

**This endpoint allows you to enter an IP address into warmup mode.**

Twilio SendGrid can automatically warm up dedicated IP addresses by limiting the amount of mail that can be sent through them per hour, with the limit determined by how long the IP address has been in warmup. See the [warmup schedule](https://sendgrid.com/docs/API_Reference/Web_API_v3/IP_Management/ip_warmup_schedule.html) for more details on how Twilio SendGrid limits your email traffic for IPs in warmup.

For more general information about warming up IPs, please see our [Classroom](https://sendgrid.com/docs/Classroom/Deliver/Delivery_Introduction/warming_up_ips.html).

### POST /ips/warmup

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/warmup", host)
request.Method = "POST"
request.Body = []byte(` {
  "ip": "0.0.0.0"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all IPs currently in warmup

**This endpoint allows you to retrieve all of your IP addresses that are currently warming up.**

Twilio SendGrid can automatically warm up dedicated IP addresses by limiting the amount of mail that can be sent through them per hour, with the limit determined by how long the IP address has been in warmup. See the [warmup schedule](https://sendgrid.com/docs/API_Reference/Web_API_v3/IP_Management/ip_warmup_schedule.html) for more details on how Twilio SendGrid limits your email traffic for IPs in warmup.

For more general information about warming up IPs, please see our [Classroom](https://sendgrid.com/docs/Classroom/Deliver/Delivery_Introduction/warming_up_ips.html).

### GET /ips/warmup

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/warmup", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve warmup status for a specific IP address

**This endpoint allows you to retrieve the warmup status for a specific IP address.**

Twilio SendGrid can automatically warm up dedicated IP addresses by limiting the amount of mail that can be sent through them per hour, with the limit determined by how long the IP address has been in warmup. See the [warmup schedule](https://sendgrid.com/docs/API_Reference/Web_API_v3/IP_Management/ip_warmup_schedule.html) for more details on how Twilio SendGrid limits your email traffic for IPs in warmup.

For more general information about warming up IPs, please see our [Classroom](https://sendgrid.com/docs/Classroom/Deliver/Delivery_Introduction/warming_up_ips.html).

### GET /ips/warmup/{ip_address}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/warmup/{ip_address}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Remove an IP from warmup

**This endpoint allows you to remove an IP address from warmup mode.**

Twilio SendGrid can automatically warm up dedicated IP addresses by limiting the amount of mail that can be sent through them per hour, with the limit determined by how long the IP address has been in warmup. See the [warmup schedule](https://sendgrid.com/docs/API_Reference/Web_API_v3/IP_Management/ip_warmup_schedule.html) for more details on how Twilio SendGrid limits your email traffic for IPs in warmup.

For more general information about warming up IPs, please see our [Classroom](https://sendgrid.com/docs/Classroom/Deliver/Delivery_Introduction/warming_up_ips.html).

### DELETE /ips/warmup/{ip_address}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/warmup/{ip_address}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all IP pools an IP address belongs to

**This endpoint allows you to see which IP pools a particular IP address has been added to.**

The same IP address can be added to multiple IP pools.

A single IP address or a range of IP addresses may be dedicated to an account in order to send email for multiple domains. The reputation of this IP is based on the aggregate performance of all the senders who use it.

### GET /ips/{ip_address}

```go
request := sendgrid.GetRequest(apiKey, "/v3/ips/{ip_address}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="mail"></a>
# MAIL

## Create a batch ID

**This endpoint allows you to generate a new batch ID. This batch ID can be associated with scheduled sends via the mail/send endpoint.**

If you set the SMTPAPI header `batch_id`, it allows you to then associate multiple scheduled mail/send requests together with the same ID. Then at any time up to 10 minutes before the scheduled date, you can cancel all of the mail/send requests that have this batch ID by calling the Cancel Scheduled Send endpoint.

More Information:

* [Scheduling Parameters > Batch ID](https://sendgrid.com/docs/API_Reference/SMTP_API/scheduling_parameters.html)

### POST /mail/batch

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail/batch", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Validate batch ID

**This endpoint allows you to validate a batch ID.**

If you set the SMTPAPI header `batch_id`, it allows you to then associate multiple scheduled mail/send requests together with the same ID. Then at any time up to 10 minutes before the scheduled date, you can cancel all of the mail/send requests that have this batch ID by calling the Cancel Scheduled Send endpoint.

More Information:

* [Scheduling Parameters > Batch ID](https://sendgrid.com/docs/API_Reference/SMTP_API/scheduling_parameters.html)

### GET /mail/batch/{batch_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail/batch/{batch_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## v3 Mail Send

This endpoint allows you to send an email over Twilio SendGrid's v3 Web API, the most recent version of our API. If you are looking for documentation about the v2 Mail Send endpoint, please see our [v2 API Reference](https://sendgrid.com/docs/API_Reference/Web_API/mail.html).

* Top level parameters are referred to as "global".
* Individual fields within the personalizations array will override any other global, or message level, parameters that are defined outside of personalizations.

For an overview of the v3 Mail Send endpoint, please visit our [v3 API Reference](https://sendgrid.com/docs/API_Reference/Web_API_v3/Mail/index.html)

For more detailed information about how to use the v3 Mail Send endpoint, please visit our [Classroom](https://sendgrid.com/docs/Classroom/Send/v3_Mail_Send/index.html).

### POST /mail/send
This endpoint has a helper, check it out [here](helpers/mail/README.md).

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail/send", host)
request.Method = "POST"
request.Body = []byte(` {
  "asm": {
    "group_id": 1,
    "groups_to_display": [
      1,
      2,
      3
    ]
  },
  "attachments": [
    {
      "content": "[BASE64 encoded content block here]",
      "content_id": "ii_139db99fdb5c3704",
      "disposition": "inline",
      "filename": "file1.jpg",
      "name": "file1",
      "type": "jpg"
    }
  ],
  "batch_id": "[YOUR BATCH ID GOES HERE]",
  "categories": [
    "category1",
    "category2"
  ],
  "content": [
    {
      "type": "text/html",
      "value": "<html><p>Hello, world!</p><img src=[CID GOES HERE]></img></html>"
    }
  ],
  "custom_args": {
    "New Argument 1": "New Value 1",
    "activationAttempt": "1",
    "customerAccountNumber": "[CUSTOMER ACCOUNT NUMBER GOES HERE]"
  },
  "from": {
    "email": "sam.smith@example.com",
    "name": "Sam Smith"
  },
  "headers": {},
  "ip_pool_name": "[YOUR POOL NAME GOES HERE]",
  "mail_settings": {
    "bcc": {
      "email": "ben.doe@example.com",
      "enable": true
    },
    "bypass_bounce_management": {
      "enable": true
    },
    "bypass_list_management": {
      "enable": true
    },
    "bypass_spam_management": {
      "enable": true
    },
    "bypass_unsubscribe_management": {
      "enable": true
    },
    "footer": {
      "enable": true,
      "html": "<p>Thanks</br>The Twilio SendGrid Team</p>",
      "text": "Thanks,/n The Twilio SendGrid Team"
    },
    "sandbox_mode": {
      "enable": false
    },
    "spam_check": {
      "enable": true,
      "post_to_url": "http://example.com/compliance",
      "threshold": 3
    }
  },
  "personalizations": [
    {
      "bcc": [
        {
          "email": "sam.doe@example.com",
          "name": "Sam Doe"
        }
      ],
      "cc": [
        {
          "email": "jane.doe@example.com",
          "name": "Jane Doe"
        }
      ],
      "custom_args": {
        "New Argument 1": "New Value 1",
        "activationAttempt": "1",
        "customerAccountNumber": "[CUSTOMER ACCOUNT NUMBER GOES HERE]"
      },
      "headers": {
        "X-Accept-Language": "en",
        "X-Mailer": "MyApp"
      },
      "send_at": 1409348513,
      "subject": "Hello, World!",
      "substitutions": {
        "id": "substitutions",
        "type": "object"
      },
      "to": [
        {
          "email": "john.doe@example.com",
          "name": "John Doe"
        }
      ]
    }
  ],
  "reply_to": {
    "email": "sam.smith@example.com",
    "name": "Sam Smith"
  },
  "sections": {
    "section": {
      ":sectionName1": "section 1 text",
      ":sectionName2": "section 2 text"
    }
  },
  "send_at": 1409348513,
  "subject": "Hello, World!",
  "template_id": "[YOUR TEMPLATE ID GOES HERE]",
  "tracking_settings": {
    "click_tracking": {
      "enable": true,
      "enable_text": true
    },
    "ganalytics": {
      "enable": true,
      "utm_campaign": "[NAME OF YOUR REFERRER SOURCE]",
      "utm_content": "[USE THIS SPACE TO DIFFERENTIATE YOUR EMAIL FROM ADS]",
      "utm_medium": "[NAME OF YOUR MARKETING MEDIUM e.g. email]",
      "utm_name": "[NAME OF YOUR CAMPAIGN]",
      "utm_term": "[IDENTIFY PAID KEYWORDS HERE]"
    },
    "open_tracking": {
      "enable": true,
      "substitution_tag": "%opentrack"
    },
    "subscription_tracking": {
      "enable": true,
      "html": "If you would like to unsubscribe and stop receiving these emails <% clickhere %>.",
      "substitution_tag": "<%click here%>",
      "text": "If you would like to unsubscribe and stop receiving these emails <% click here %>."
    }
  }
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="mail-settings"></a>
# MAIL SETTINGS

## Retrieve all mail settings

**This endpoint allows you to retrieve a list of all mail settings.**

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update address whitelist mail settings

**This endpoint allows you to update your current email address whitelist settings.**

The address whitelist setting whitelists a specified email address or domain for which mail should never be suppressed. For example, you own the domain example.com, and one or more of your recipients use email@example.com addresses, by placing example.com in the address whitelist setting, all bounces, blocks, and unsubscribes logged for that domain will be ignored and sent as if under normal sending conditions.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/address_whitelist

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/address_whitelist", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "list": [
    "email1@example.com",
    "example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve address whitelist mail settings

**This endpoint allows you to retrieve your current email address whitelist settings.**

The address whitelist setting whitelists a specified email address or domain for which mail should never be suppressed. For example, you own the domain example.com, and one or more of your recipients use email@example.com addresses, by placing example.com in the address whitelist setting, all bounces, blocks, and unsubscribes logged for that domain will be ignored and sent as if under normal sending conditions.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/address_whitelist

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/address_whitelist", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update BCC mail settings

**This endpoint allows you to update your current BCC mail settings.**

When the BCC mail setting is enabled, Twilio SendGrid will automatically send a blind carbon copy (BCC) to an address for every email sent without adding that address to the header. Please note that only one email address may be entered in this field, if you wish to distribute BCCs to multiple addresses you will need to create a distribution group or use forwarding rules.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/bcc

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/bcc", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "email": "email@example.com",
  "enabled": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all BCC mail settings

**This endpoint allows you to retrieve your current BCC mail settings.**

When the BCC mail setting is enabled, Twilio SendGrid will automatically send a blind carbon copy (BCC) to an address for every email sent without adding that address to the header. Please note that only one email address may be entered in this field, if you wish to distribute BCCs to multiple addresses you will need to create a distribution group or use forwarding rules.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/bcc

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/bcc", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update bounce purge mail settings

**This endpoint allows you to update your current bounce purge settings.**

This setting allows you to set a schedule for Twilio SendGrid to automatically delete contacts from your soft and hard bounce suppression lists.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/bounce_purge

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/bounce_purge", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "hard_bounces": 5,
  "soft_bounces": 5
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve bounce purge mail settings

**This endpoint allows you to retrieve your current bounce purge settings.**

This setting allows you to set a schedule for Twilio SendGrid to automatically delete contacts from your soft and hard bounce suppression lists.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/bounce_purge

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/bounce_purge", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update footer mail settings

**This endpoint allows you to update your current Footer mail settings.**

The footer setting will insert a custom footer at the bottom of the text and HTML bodies. Use the embedded HTML editor and plain text entry fields to create the content of the footers to be inserted into your emails.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/footer

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/footer", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "html_content": "...",
  "plain_content": "..."
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve footer mail settings

**This endpoint allows you to retrieve your current Footer mail settings.**

The footer setting will insert a custom footer at the bottom of the text and HTML bodies. Use the embedded HTML editor and plain text entry fields to create the content of the footers to be inserted into your emails.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/footer

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/footer", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update forward bounce mail settings

**This endpoint allows you to update your current bounce forwarding mail settings.**

Activating this setting allows you to specify an email address to which bounce reports are forwarded.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/forward_bounce

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/forward_bounce", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "email": "example@example.com",
  "enabled": true
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve forward bounce mail settings

**This endpoint allows you to retrieve your current bounce forwarding mail settings.**

Activating this setting allows you to specify an email address to which bounce reports are forwarded.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/forward_bounce

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/forward_bounce", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update forward spam mail settings

**This endpoint allows you to update your current Forward Spam mail settings.**

Enabling the forward spam setting allows you to specify an email address to which spam reports will be forwarded.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/forward_spam

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/forward_spam", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "email": "",
  "enabled": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve forward spam mail settings

**This endpoint allows you to retrieve your current Forward Spam mail settings.**

Enabling the forward spam setting allows you to specify an email address to which spam reports will be forwarded.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/forward_spam

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/forward_spam", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update plain content mail settings

**This endpoint allows you to update your current Plain Content mail settings.**

The plain content setting will automatically convert any plain text emails that you send to HTML before sending.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/plain_content

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/plain_content", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve plain content mail settings

**This endpoint allows you to retrieve your current Plain Content mail settings.**

The plain content setting will automatically convert any plain text emails that you send to HTML before sending.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/plain_content

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/plain_content", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update spam check mail settings

**This endpoint allows you to update your current spam checker mail settings.**

The spam checker filter notifies you when emails are detected that exceed a predefined spam threshold.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/spam_check

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/spam_check", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "max_score": 5,
  "url": "url"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve spam check mail settings

**This endpoint allows you to retrieve your current Spam Checker mail settings.**

The spam checker filter notifies you when emails are detected that exceed a predefined spam threshold.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/spam_check

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/spam_check", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update template mail settings

**This endpoint allows you to update your current legacy email template settings.**

This setting refers to our original email templates. We currently support more fully featured [transactional templates](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

The legacy email template setting wraps an HTML template around your email content. This can be useful for sending out marketing email and/or other HTML formatted messages.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### PATCH /mail_settings/template

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/template", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "html_content": "<% body %>"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve legacy template mail settings

**This endpoint allows you to retrieve your current legacy email template settings.**

This setting refers to our original email templates. We currently support more fully featured [transactional templates](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

The legacy email template setting wraps an HTML template around your email content. This can be useful for sending out marketing email and/or other HTML formatted messages.

Mail settings allow you to tell Twilio SendGrid specific things to do to every email that you send to your recipients over Twilio SendGrid's [Web API](https://sendgrid.com/docs/API_Reference/Web_API/mail.html) or [SMTP Relay](https://sendgrid.com/docs/API_Reference/SMTP_API/index.html).

### GET /mail_settings/template

```go
request := sendgrid.GetRequest(apiKey, "/v3/mail_settings/template", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="mailbox-providers"></a>
# MAILBOX PROVIDERS

## Retrieve email statistics by mailbox provider.

**This endpoint allows you to retrieve your email statistics segmented by recipient mailbox provider.**

**We only store up to 7 days of email activity in our database.** By default, 500 items will be returned per request via the Advanced Stats API endpoints.

Advanced Stats provide a more in-depth view of your email statistics and the actions taken by your recipients. You can segment these statistics by geographic location, device type, client type, browser, and mailbox provider. For more information about statistics, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/index.html).

### GET /mailbox_providers/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/mailbox_providers/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["mailbox_providers"] = "test_string"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="partner-settings"></a>
# PARTNER SETTINGS

## Returns a list of all partner settings.

**This endpoint allows you to retrieve a list of all partner settings that you can enable.**

Our partner settings allow you to integrate your Twilio SendGrid account with our partners to increase your Twilio SendGrid experience and functionality. For more information about our partners, and how you can begin integrating with them, please visit our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/partners.html).

### GET /partner_settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/partner_settings", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Updates New Relic partner settings.

**This endpoint allows you to update or change your New Relic partner settings.**

Our partner settings allow you to integrate your Twilio SendGrid account with our partners to increase your Twilio SendGrid experience and functionality. For more information about our partners, and how you can begin integrating with them, please visit our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/partners.html).

By integrating with New Relic, you can send your Twilio SendGrid email statistics to your New Relic Dashboard. If you enable this setting, your stats will be sent to New Relic every 5 minutes. You will need your New Relic License Key to enable this setting. For more information, please see our [Classroom](https://sendgrid.com/docs/Classroom/Track/Collecting_Data/new_relic.html).

### PATCH /partner_settings/new_relic

```go
request := sendgrid.GetRequest(apiKey, "/v3/partner_settings/new_relic", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enable_subuser_statistics": true,
  "enabled": true,
  "license_key": ""
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Returns all New Relic partner settings.

**This endpoint allows you to retrieve your current New Relic partner settings.**

Our partner settings allow you to integrate your Twilio SendGrid account with our partners to increase your Twilio SendGrid experience and functionality. For more information about our partners, and how you can begin integrating with them, please visit our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/partners.html).

By integrating with New Relic, you can send your Twilio SendGrid email statistics to your New Relic Dashboard. If you enable this setting, your stats will be sent to New Relic every 5 minutes. You will need your New Relic License Key to enable this setting. For more information, please see our [Classroom](https://sendgrid.com/docs/Classroom/Track/Collecting_Data/new_relic.html).

### GET /partner_settings/new_relic

```go
request := sendgrid.GetRequest(apiKey, "/v3/partner_settings/new_relic", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="scopes"></a>
# SCOPES

## Retrieve a list of scopes for which this user has access.

**This endpoint returns a list of all scopes that this user has access to.**

API Keys can be used to authenticate the use of [Twilio SendGrid's v3 Web API](https://sendgrid.com/docs/API_Reference/Web_API_v3/index.html), or the [Mail API Endpoint](https://sendgrid.com/docs/API_Reference/Web_API/mail.html). API Keys may be assigned certain permissions, or scopes, that limit which API endpoints they are able to access. For a more detailed explanation of how you can use API Key permissions, please visit our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/api_keys.html#-API-Key-Permissions) or [Classroom](https://sendgrid.com/docs/Classroom/Basics/API/api_key_permissions.html).

### GET /scopes

```go
request := sendgrid.GetRequest(apiKey, "/v3/scopes", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="senders"></a>
# SENDERS

## Create a Sender Identity

**This endpoint allows you to create a new sender identity.**

*You may create up to 100 unique sender identities.*

Sender Identities are required to be verified before use. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

### POST /senders

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders", host)
request.Method = "POST"
request.Body = []byte(` {
  "address": "123 Elm St.",
  "address_2": "Apt. 456",
  "city": "Denver",
  "country": "United States",
  "from": {
    "email": "from@example.com",
    "name": "Example INC"
  },
  "nickname": "My Sender ID",
  "reply_to": {
    "email": "replyto@example.com",
    "name": "Example INC"
  },
  "state": "Colorado",
  "zip": "80202"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Get all Sender Identities

**This endpoint allows you to retrieve a list of all sender identities that have been created for your account.**

Sender Identities are required to be verified before use. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

### GET /senders

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a Sender Identity

**This endpoint allows you to update a sender identity.**

Updates to `from.email` require re-verification. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

Partial updates are allowed, but fields that are marked as "required" in the POST (create) endpoint must not be nil if that field is included in the PATCH request.

### PATCH /senders/{sender_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders/{sender_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "address": "123 Elm St.",
  "address_2": "Apt. 456",
  "city": "Denver",
  "country": "United States",
  "from": {
    "email": "from@example.com",
    "name": "Example INC"
  },
  "nickname": "My Sender ID",
  "reply_to": {
    "email": "replyto@example.com",
    "name": "Example INC"
  },
  "state": "Colorado",
  "zip": "80202"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## View a Sender Identity

**This endpoint allows you to retrieve a specific sender identity.**

Sender Identities are required to be verified before use. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

### GET /senders/{sender_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders/{sender_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Sender Identity

**This endpoint allows you to delete one of your sender identities.**

Sender Identities are required to be verified before use. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

### DELETE /senders/{sender_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders/{sender_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Resend Sender Identity Verification

**This endpoint allows you to resend a sender identity verification email.**

Sender Identities are required to be verified before use. If your domain has been authenticated, it will auto verify on creation. Otherwise, an email will be sent to the `from.email`.

### POST /senders/{sender_id}/resend_verification

```go
request := sendgrid.GetRequest(apiKey, "/v3/senders/{sender_id}/resend_verification", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="sender-authentication"></a>
# SENDER AUTHENTICATION

## Create an authenticated domain.

**This endpoint allows you to create a domain authentication for one of your domains.**

If you are creating a domain authentication that you would like a subuser to use, you have two options:
1. Use the "username" parameter. This allows you to create am authenticated subuser. This means the subuser is able to see and modify the created authentication.
2. Use the Association workflow (see Associate Domain section). This allows you to assign a domain authentication created by the parent to a subuser. This means the subuser will default to the assigned domain authentication, but will not be able to see or modify that authentication. However, if the subuser creates their own domain authentication it will overwrite the assigned domain authentication.

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

### POST /whitelabel/domains

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains", host)
request.Method = "POST"
request.Body = []byte(` {
  "automatic_security": false,
  "custom_spf": true,
  "default": true,
  "domain": "example.com",
  "ips": [
    "192.168.1.1",
    "192.168.1.2"
  ],
  "subdomain": "news",
  "username": "john@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## List all domain authentications.

**This endpoint allows you to retrieve a list of all domain authentications you have created.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)


### GET /whitelabel/domains

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["username"] = "test_string"
queryParams["domain"] = "test_string"
queryParams["exclude_subusers"] = "true"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Get the default domain authentication.

**This endpoint allows you to retrieve the default authentication for a domain.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type   | Description  |
|---|---|---|
| domain | string  |The domain to find a default domain whitelabel for. |

### GET /whitelabel/domains/default

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/default", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## List the domain authentication associated with the given user.

**This endpoint allows you to retrieve all of the domain authentications that have been assigned to a specific subuser.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

Domain authentications can be associated with (i.e. assigned to) subusers from a parent account. This functionality allows subusers to send mail using their parent's authenticated domains. To associate a domain authentication with a subuser, the parent account must first create the domain authentication and validate it. The parent may then associate the domain authentication via the subuser management tools.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type  | Description  |
|---|---|---|
| username | string  | Username of the subuser to find associated whitelabels for. |

### GET /whitelabel/domains/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/subuser", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Disassociate a domain authentication from a given user.

**This endpoint allows you to disassociate a specific domain authentication from a subuser.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

Domain authentications can be associated with (i.e. assigned to) subusers from a parent account. This functionality allows subusers to send mail using their parent's authenticated domains. To associate a domain authentication with a subuser, the parent account must first create the domain authentication and validate it. The parent may then associate the domain authentication via the subuser management tools.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type  | Required?  | Description  |
|---|---|---|---|
| username | string  | required  | Username for the subuser to find associated whitelabels for. |

### DELETE /whitelabel/domains/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/subuser", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a domain authentication.

**This endpoint allows you to update the settings for a domain authentication.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

### PATCH /whitelabel/domains/{domain_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{domain_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "custom_spf": true,
  "default": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a domain authentication.

**This endpoint allows you to retrieve a specific domain authentication.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)


### GET /whitelabel/domains/{domain_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{domain_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a domain authentication.

**This endpoint allows you to delete a domain authentication.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

### DELETE /whitelabel/domains/{domain_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{domain_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Associate a domain authentication with a given user.

**This endpoint allows you to associate a specific domain authentication with a subuser.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

Domain authentications can be associated with (i.e. assigned to) subusers from a parent account. This functionality allows subusers to send mail using their parent's authenticated domains. To associate a domain authentication with a subuser, the parent account must first create the domain authentication and validate it. The parent may then associate the domain authentication via the subuser management tools.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type   | Description  |
|---|---|---|
| domain_id | integer   | ID of the domain whitelabel to associate with the subuser. |

### POST /whitelabel/domains/{domain_id}/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{domain_id}/subuser", host)
request.Method = "POST"
request.Body = []byte(` {
  "username": "jane@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Add an IP to a domain authentication.

**This endpoint allows you to add an IP address to a domain authentication.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type  |  Description  |
|---|---|---|
| id | integer  | ID of the domain to which you are adding an IP |

### POST /whitelabel/domains/{id}/ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{id}/ips", host)
request.Method = "POST"
request.Body = []byte(` {
  "ip": "192.168.0.1"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Remove an IP from a domain authentication.

**This endpoint allows you to remove a domain's IP address from that domain's authentication.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type  | Description  |
|---|---|---|
| id | integer  | ID of the domain whitelabel to delete the IP from. |
| ip | string | IP to remove from the domain whitelabel. |

### DELETE /whitelabel/domains/{id}/ips/{ip}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{id}/ips/{ip}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Validate a domain authentication.

**This endpoint allows you to validate a domain authentication. If it fails, it will return an error message describing why the domain could not be validated.**

A domain authentication allows you to remove the via or sent on behalf of message that your recipients see when they read your emails. Authenticating a domain allows you to replace sendgrid.net with your personal sending domain. You will be required to create a subdomain so that SendGrid can generate the DNS records which you must give to your host provider. If you choose to use Automated Security, SendGrid will provide you with 3 CNAME records. If you turn Automated Security off, you will be given 2 TXT records and 1 MX record.

For more information on domain authentication, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-domain-authentication/)

## URI Parameters
| URI Parameter   | Type   | Description  |
|---|---|---|
| id | integer  |ID of the domain whitelabel to validate. |

### POST /whitelabel/domains/{id}/validate

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/domains/{id}/validate", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create reverse DNS record

**This endpoint allows you to create a reverse DNS record.**

When creating a reverse DNS record, you should use the same subdomain that you used when you created a domain authentication.

Reverse DNS consists of a subdomain and domain that will be used to generate a record for a given IP. Once Twilio SendGrid has verified that the appropriate A record for the IP has been created, the appropriate reverse DNS record for the IP is generated.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/).

### POST /whitelabel/ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/ips", host)
request.Method = "POST"
request.Body = []byte(` {
  "domain": "example.com",
  "ip": "192.168.1.1",
  "subdomain": "email"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all reverse DNS records

**This endpoint allows you to retrieve all of the reverse DNS records that have been created by this account.**

You may include a search key by using the "ip" parameter. This enables you to perform a prefix search for a given IP segment (e.g. "192.").

Reverse DNS consists of a subdomain and domain that will be used to generate a record for a given IP. Once Twilio SendGrid has verified that the appropriate A record for the IP has been created, the appropriate reverse DNS record for the IP is generated.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/).

### GET /whitelabel/ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/ips", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["ip"] = "test_string"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve an reverse DNS record

**This endpoint allows you to retrieve a reverse DNS record.**

Reverse DNS consists of a subdomain and domain that will be used to generate a record for a given IP. Once Twilio SendGrid has verified that the appropriate A record for the IP has been created, the appropriate reverse DNS record for the IP is generated.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/).

### GET /whitelabel/ips/{id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/ips/{id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete an reverse DNS record

**This endpoint allows you to delete a reverse DNS record.**

Reverse DNS consists of a subdomain and domain that will be used to generate a record for a given IP. Once Twilio SendGrid has verified that the appropriate A record for the IP has been created, the appropriate reverse DNS record for the IP is generated.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/).

### DELETE /whitelabel/ips/{id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/ips/{id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Validate a reverse DNS

**This endpoint allows you to validate a reverse DNS record.**

Reverse DNS consists of a subdomain and domain that will be used to generate a record for a given IP. Once Twilio SendGrid has verified that the appropriate A record for the IP has been created, the appropriate reverse DNS record for the IP is generated.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/).

### POST /whitelabel/ips/{id}/validate

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/ips/{id}/validate", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create a Branded Link

**This endpoint allows you to create a new link branding.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### POST /whitelabel/links

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links", host)
request.Method = "POST"
request.Body = []byte(` {
  "default": true,
  "domain": "example.com",
  "subdomain": "mail"
}`)
queryParams := make(map[string]string)
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all link brandings

**This endpoint allows you to retrieve all link brandings.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### GET /whitelabel/links

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Default Link Branding

**This endpoint allows you to retrieve the default link branding.**

Default link branding is the actual link branding to be used when sending messages. If there are multiple link brandings, the default is determined by the following order:
<ul>
  <li>Validated link branding marked as "default"</li>
  <li>Legacy link brands (migrated from the whitelabel wizard)</li>
  <li>Default SendGrid link whitelabel (i.e. 100.ct.sendgrid.net)</li>
</ul>

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### GET /whitelabel/links/default

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/default", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["domain"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Associated Link Branding

**This endpoint allows you to retrieve the associated link branding for a subuser.**

Link branding can be associated with subusers from the parent account. This functionality allows
subusers to send mail using their parent's link brands. To associate a link branding, the parent account
must first create a branded link and validate it. The parent may then associate that branded link with a subuser via the API or the Subuser Management page in the user interface.

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### GET /whitelabel/links/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/subuser", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["username"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Disassociate a Link Branding

**This endpoint allows you to disassociate a link branding from a subuser.**

Link branding can be associated with subusers from the parent account. This functionality allows
subusers to send mail using their parent's link brands. To associate a link branding, the parent account
must first create a branded link and validate it. The parent may then associate that branded link with a subuser via the API or the Subuser Management page in the user interface.

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### DELETE /whitelabel/links/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/subuser", host)
request.Method = "DELETE"
queryParams := make(map[string]string)
queryParams["username"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a Link Branding

**This endpoint allows you to update a specific link branding. You can use this endpoint to change a branded link's default status.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### PATCH /whitelabel/links/{id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/{id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "default": true
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Link Branding

**This endpoint allows you to retrieve a specific link branding.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### GET /whitelabel/links/{id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/{id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a Link Branding

**This endpoint allows you to delete a link branding.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### DELETE /whitelabel/links/{id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/{id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Validate a Link Branding

**This endpoint allows you to validate a link branding.**

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### POST /whitelabel/links/{id}/validate

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/{id}/validate", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Associate a Link Branding

**This endpoint allows you to associate a link branding with a subuser account.**

Link branding can be associated with subusers from the parent account. This functionality allows
subusers to send mail using their parent's link brands. To associate a link branding, the parent account
must first create a branded link and validate it. The parent may then associate that branded link with a subuser via the API or the Subuser Management page in the user interface.

Email link branding allow all of the click-tracked links you send in your emails to include the URL of your domain instead of sendgrid.net.

For more information, please see our [User Guide](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-link-branding/).

### POST /whitelabel/links/{link_id}/subuser

```go
request := sendgrid.GetRequest(apiKey, "/v3/whitelabel/links/{link_id}/subuser", host)
request.Method = "POST"
request.Body = []byte(` {
  "username": "jane@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="stats"></a>
# STATS

## Retrieve global email statistics

**This endpoint allows you to retrieve all of your global email statistics between a given date range.**

Parent accounts will see aggregated stats for their account and all subuser accounts. Subuser accounts will only see their own stats.

### GET /stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["end_date"] = "2016-04-01"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="subusers"></a>
# SUBUSERS

## Create Subuser

This endpoint allows you to retrieve a list of all of your subusers. You can choose to retrieve specific subusers as well as limit the results that come back from the API.

For more information about Subusers:

* [User Guide > Subusers](https://sendgrid.com/docs/User_Guide/Settings/Subusers/index.html)
* [Classroom > How do I add more subusers to my account?](https://sendgrid.com/docs/Classroom/Basics/Account/how_do_i_add_more_subusers_to_my_account.html)

### POST /subusers

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers", host)
request.Method = "POST"
request.Body = []byte(` {
  "email": "John@example.com",
  "ips": [
    "1.1.1.1",
    "2.2.2.2"
  ],
  "password": "johns_password",
  "username": "John@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## List all Subusers

This endpoint allows you to retrieve a list of all of your subusers. You can choose to retrieve specific subusers as well as limit the results that come back from the API.

For more information about Subusers:

* [User Guide > Subusers](https://sendgrid.com/docs/User_Guide/Settings/Subusers/index.html)
* [Classroom > How do I add more subusers to my account?](https://sendgrid.com/docs/Classroom/Basics/Account/how_do_i_add_more_subusers_to_my_account.html)

### GET /subusers

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["username"] = "test_string"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Subuser Reputations

Subuser sender reputations give a good idea how well a sender is doing with regards to how recipients and recipient servers react to the mail that is being received. When a bounce, spam report, or other negative action happens on a sent email, it will effect your sender rating.

This endpoint allows you to request the reputations for your subusers.

### GET /subusers/reputations

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/reputations", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["usernames"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve email statistics for your subusers.

**This endpoint allows you to retrieve the email statistics for the given subusers.**

You may retrieve statistics for up to 10 different subusers by including an additional _subusers_ parameter for each additional subuser.

While you can always view the statistics for all email activity on your account, subuser statistics enable you to view specific segments of your stats. Emails sent, bounces, and spam reports are always tracked for subusers. Unsubscribes, clicks, and opens are tracked if you have enabled the required settings.

For more information, see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/subuser.html).

### GET /subusers/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["subusers"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve monthly stats for all subusers

**This endpoint allows you to retrieve the monthly email statistics for all subusers over the given date range.**

While you can always view the statistics for all email activity on your account, subuser statistics enable you to view specific segments of your stats for your subusers. Emails sent, bounces, and spam reports are always tracked for subusers. Unsubscribes, clicks, and opens are tracked if you have enabled the required settings.

When using the `sort_by_metric` to sort your stats by a specific metric, you can not sort by the following metrics:
`bounce_drops`, `deferred`, `invalid_emails`, `processed`, `spam_report_drops`, `spam_reports`, or `unsubscribe_drops`.

For more information, see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/subuser.html).

### GET /subusers/stats/monthly

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/stats/monthly", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["subuser"] = "test_string"
queryParams["limit"] = "1"
queryParams["sort_by_metric"] = "test_string"
queryParams["offset"] = "1"
queryParams["date"] = "test_string"
queryParams["sort_by_direction"] = "asc"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

##  Retrieve the totals for each email statistic metric for all subusers.

**This endpoint allows you to retrieve the total sums of each email statistic metric for all subusers over the given date range.**


While you can always view the statistics for all email activity on your account, subuser statistics enable you to view specific segments of your stats. Emails sent, bounces, and spam reports are always tracked for subusers. Unsubscribes, clicks, and opens are tracked if you have enabled the required settings.

For more information, see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/subuser.html).

### GET /subusers/stats/sums

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/stats/sums", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["end_date"] = "2016-04-01"
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "1"
queryParams["sort_by_metric"] = "test_string"
queryParams["offset"] = "1"
queryParams["start_date"] = "2016-01-01"
queryParams["sort_by_direction"] = "asc"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Enable/disable a subuser

This endpoint allows you to enable or disable a subuser.

For more information about Subusers:

* [User Guide > Subusers](https://sendgrid.com/docs/User_Guide/Settings/Subusers/index.html)
* [Classroom > How do I add more subusers to my account?](https://sendgrid.com/docs/Classroom/Basics/Account/how_do_i_add_more_subusers_to_my_account.html)

### PATCH /subusers/{subuser_name}

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "disabled": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a subuser

This endpoint allows you to delete a subuser. This is a permanent action, once deleted a subuser cannot be retrieved.

For more information about Subusers:

* [User Guide > Subusers](https://sendgrid.com/docs/User_Guide/Settings/Subusers/index.html)
* [Classroom > How do I add more subusers to my account?](https://sendgrid.com/docs/Classroom/Basics/Account/how_do_i_add_more_subusers_to_my_account.html)

### DELETE /subusers/{subuser_name}

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update IPs assigned to a subuser

Each subuser should be assigned to an IP address, from which all of this subuser's mail will be sent. Often, this is the same IP as the parent account, but each subuser can have their own, or multiple, IP addresses as well.

More information:

* [How to request more IPs](https://sendgrid.com/docs/Classroom/Basics/Account/adding_an_additional_dedicated_ip_to_your_account.html)
* [Setup Reverse DNS](https://sendgrid.com/docs/ui/account-and-settings/how-to-set-up-reverse-dns/)

### PUT /subusers/{subuser_name}/ips

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/ips", host)
request.Method = "PUT"
request.Body = []byte(` [
  "127.0.0.1"
]`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Monitor Settings for a subuser

Subuser monitor settings allow you to receive a sample of an outgoing message by a specific customer at a specific frequency of emails.

### PUT /subusers/{subuser_name}/monitor

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/monitor", host)
request.Method = "PUT"
request.Body = []byte(` {
  "email": "example@example.com",
  "frequency": 500
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create monitor settings

Subuser monitor settings allow you to receive a sample of an outgoing message by a specific customer at a specific frequency of emails.

### POST /subusers/{subuser_name}/monitor

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/monitor", host)
request.Method = "POST"
request.Body = []byte(` {
  "email": "example@example.com",
  "frequency": 50000
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve monitor settings for a subuser

Subuser monitor settings allow you to receive a sample of an outgoing message by a specific customer at a specific frequency of emails.

### GET /subusers/{subuser_name}/monitor

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/monitor", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete monitor settings

Subuser monitor settings allow you to receive a sample of an outgoing message by a specific customer at a specific frequency of emails.

### DELETE /subusers/{subuser_name}/monitor

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/monitor", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve the monthly email statistics for a single subuser

**This endpoint allows you to retrieve the monthly email statistics for a specific subuser.**

While you can always view the statistics for all email activity on your account, subuser statistics enable you to view specific segments of your stats for your subusers. Emails sent, bounces, and spam reports are always tracked for subusers. Unsubscribes, clicks, and opens are tracked if you have enabled the required settings.

When using the `sort_by_metric` to sort your stats by a specific metric, you can not sort by the following metrics:
`bounce_drops`, `deferred`, `invalid_emails`, `processed`, `spam_report_drops`, `spam_reports`, or `unsubscribe_drops`.

For more information, see our [User Guide](https://sendgrid.com/docs/User_Guide/Statistics/subuser.html).

### GET /subusers/{subuser_name}/stats/monthly

```go
request := sendgrid.GetRequest(apiKey, "/v3/subusers/{subuser_name}/stats/monthly", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["date"] = "test_string"
queryParams["sort_by_direction"] = "asc"
queryParams["limit"] = "1"
queryParams["sort_by_metric"] = "test_string"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="suppression"></a>
# SUPPRESSION

## Retrieve all blocks

**This endpoint allows you to retrieve a list of all email addresses that are currently on your blocks list.**

[Blocks](https://sendgrid.com/docs/Glossary/blocks.html) happen when your message was rejected for a reason related to the message, not the recipient address. This can happen when your mail server IP address has been added to a blacklist or blocked by an ISP, or if the message content is flagged by a filter on the receiving server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/blocks.html).

### GET /suppression/blocks

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/blocks", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["start_time"] = "1"
queryParams["limit"] = "1"
queryParams["end_time"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete blocks

**This endpoint allows you to delete all email addresses on your blocks list.**

There are two options for deleting blocked emails:

1. You can delete all blocked emails by setting `delete_all` to true in the request body.
2. You can delete some blocked emails by specifying the email addresses in an array in the request body.

[Blocks](https://sendgrid.com/docs/Glossary/blocks.html) happen when your message was rejected for a reason related to the message, not the recipient address. This can happen when your mail server IP address has been added to a blacklist or blocked by an ISP, or if the message content is flagged by a filter on the receiving server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/blocks.html).

### DELETE /suppression/blocks

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/blocks", host)
request.Method = "DELETE"
request.Body = []byte(` {
  "delete_all": false,
  "emails": [
    "example1@example.com",
    "example2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific block

**This endpoint allows you to retrieve a specific email address from your blocks list.**

[Blocks](https://sendgrid.com/docs/Glossary/blocks.html) happen when your message was rejected for a reason related to the message, not the recipient address. This can happen when your mail server IP address has been added to a blacklist or blocked by an ISP, or if the message content is flagged by a filter on the receiving server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/blocks.html).

### GET /suppression/blocks/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/blocks/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a specific block

**This endpoint allows you to delete a specific email address from your blocks list.**

[Blocks](https://sendgrid.com/docs/Glossary/blocks.html) happen when your message was rejected for a reason related to the message, not the recipient address. This can happen when your mail server IP address has been added to a blacklist or blocked by an ISP, or if the message content is flagged by a filter on the receiving server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/blocks.html).

### DELETE /suppression/blocks/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/blocks/{email}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all bounces

**This endpoint allows you to retrieve all of your bounces.**

Bounces are messages that are returned to the server that sent it.

For more information see:

* [User Guide > Bounces](https://sendgrid.com/docs/User_Guide/Suppressions/bounces.html) for more information
* [Glossary > Bounces](https://sendgrid.com/docs/Glossary/Bounces.html)

### GET /suppression/bounces

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/bounces", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["start_time"] = "1"
queryParams["end_time"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete bounces

**This endpoint allows you to delete all of your bounces. You can also use this endpoint to remove a specific email address from your bounce list.**

Bounces are messages that are returned to the server that sent it.

For more information see:

* [User Guide > Bounces](https://sendgrid.com/docs/User_Guide/Suppressions/bounces.html) for more information
* [Glossary > Bounces](https://sendgrid.com/docs/Glossary/Bounces.html)
* [Classroom > List Scrubbing Guide](https://sendgrid.com/docs/Classroom/Deliver/list_scrubbing.html)

Note: the `delete_all` and `emails` parameters should be used independently of each other as they have different purposes.

### DELETE /suppression/bounces

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/bounces", host)
request.Method = "DELETE"
request.Body = []byte(` {
  "delete_all": true,
  "emails": [
    "example@example.com",
    "example2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a Bounce

**This endpoint allows you to retrieve a specific bounce for a given email address.**

Bounces are messages that are returned to the server that sent it.

For more information see:

* [User Guide > Bounces](https://sendgrid.com/docs/User_Guide/Suppressions/bounces.html) for more information
* [Glossary > Bounces](https://sendgrid.com/docs/Glossary/Bounces.html)
* [Classroom > List Scrubbing Guide](https://sendgrid.com/docs/Classroom/Deliver/list_scrubbing.html)

### GET /suppression/bounces/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/bounces/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a bounce

**This endpoint allows you to remove an email address from your bounce list.**

Bounces are messages that are returned to the server that sent it. This endpoint allows you to delete a single email addresses from your bounce list.

For more information see:

* [User Guide > Bounces](https://sendgrid.com/docs/User_Guide/Suppressions/bounces.html) for more information
* [Glossary > Bounces](https://sendgrid.com/docs/Glossary/Bounces.html)
* [Classroom > List Scrubbing Guide](https://sendgrid.com/docs/Classroom/Deliver/list_scrubbing.html)

### DELETE /suppression/bounces/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/bounces/{email}", host)
request.Method = "DELETE"
queryParams := make(map[string]string)
queryParams["email_address"] = "example@example.com"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all invalid emails

**This endpoint allows you to retrieve a list of all invalid email addresses.**

An invalid email occurs when you attempt to send email to an address that is formatted in a manner that does not meet internet email format standards or the email does not exist at the recipient's mail server.

Examples include addresses without the @ sign or addresses that include certain special characters and/or spaces. This response can come from our own server or the recipient mail server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/invalid_emails.html).

### GET /suppression/invalid_emails

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/invalid_emails", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["start_time"] = "1"
queryParams["limit"] = "1"
queryParams["end_time"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete invalid emails

**This endpoint allows you to remove email addresses from your invalid email address list.**

There are two options for deleting invalid email addresses:

1) You can delete all invalid email addresses by setting `delete_all` to true in the request body.
2) You can delete some invalid email addresses by specifying certain addresses in an array in the request body.

An invalid email occurs when you attempt to send email to an address that is formatted in a manner that does not meet internet email format standards or the email does not exist at the recipient's mail server.

Examples include addresses without the @ sign or addresses that include certain special characters and/or spaces. This response can come from our own server or the recipient mail server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/invalid_emails.html).

### DELETE /suppression/invalid_emails

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/invalid_emails", host)
request.Method = "DELETE"
request.Body = []byte(` {
  "delete_all": false,
  "emails": [
    "example1@example.com",
    "example2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific invalid email

**This endpoint allows you to retrieve a specific invalid email addresses.**

An invalid email occurs when you attempt to send email to an address that is formatted in a manner that does not meet internet email format standards or the email does not exist at the recipient's mail server.

Examples include addresses without the @ sign or addresses that include certain special characters and/or spaces. This response can come from our own server or the recipient mail server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/invalid_emails.html).

### GET /suppression/invalid_emails/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/invalid_emails/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a specific invalid email

**This endpoint allows you to remove a specific email address from the invalid email address list.**

An invalid email occurs when you attempt to send email to an address that is formatted in a manner that does not meet internet email format standards or the email does not exist at the recipient's mail server.

Examples include addresses without the @ sign or addresses that include certain special characters and/or spaces. This response can come from our own server or the recipient mail server.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/invalid_emails.html).

### DELETE /suppression/invalid_emails/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/invalid_emails/{email}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific spam report

**This endpoint allows you to retrieve a specific spam report.**

[Spam reports](https://sendgrid.com/docs/Glossary/spam_reports.html) happen when a recipient indicates that they think your email is [spam](https://sendgrid.com/docs/Glossary/spam.html) and then their email provider reports this to Twilio SendGrid.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/spam_reports.html).

### GET /suppression/spam_report/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/spam_report/{email}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a specific spam report

**This endpoint allows you to delete a specific spam report.**

[Spam reports](https://sendgrid.com/docs/Glossary/spam_reports.html) happen when a recipient indicates that they think your email is [spam](https://sendgrid.com/docs/Glossary/spam.html) and then their email provider reports this to Twilio SendGrid.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/spam_reports.html).

### DELETE /suppression/spam_report/{email}

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/spam_report/{email}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all spam reports

**This endpoint allows you to retrieve all spam reports.**

[Spam reports](https://sendgrid.com/docs/Glossary/spam_reports.html) happen when a recipient indicates that they think your email is [spam](https://sendgrid.com/docs/Glossary/spam.html) and then their email provider reports this to Twilio SendGrid.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/spam_reports.html).

### GET /suppression/spam_reports

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/spam_reports", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["start_time"] = "1"
queryParams["limit"] = "1"
queryParams["end_time"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete spam reports

**This endpoint allows you to delete your spam reports.**

There are two options for deleting spam reports:

1) You can delete all spam reports by setting "delete_all" to true in the request body.
2) You can delete some spam reports by specifying the email addresses in an array in the request body.

[Spam reports](https://sendgrid.com/docs/Glossary/spam_reports.html) happen when a recipient indicates that they think your email is [spam](https://sendgrid.com/docs/Glossary/spam.html) and then their email provider reports this to Twilio SendGrid.

For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/spam_reports.html).

### DELETE /suppression/spam_reports

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/spam_reports", host)
request.Method = "DELETE"
request.Body = []byte(` {
  "delete_all": false,
  "emails": [
    "example1@example.com",
    "example2@example.com"
  ]
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all global suppressions

**This endpoint allows you to retrieve a list of all email addresses that are globally suppressed.**

A global suppression (or global unsubscribe) is an email address of a recipient who does not want to receive any of your messages. A globally suppressed recipient will be removed from any email you send. For more information, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Suppressions/global_unsubscribes.html).

### GET /suppression/unsubscribes

```go
request := sendgrid.GetRequest(apiKey, "/v3/suppression/unsubscribes", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["start_time"] = "1"
queryParams["limit"] = "1"
queryParams["end_time"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="templates"></a>
# TEMPLATES

## Create a transactional template.

**This endpoint allows you to create a transactional template.**

Each user can create up to 300 different transactional templates. Transactional templates are specific to accounts and subusers. Templates created on a parent account will not be accessible from the subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns templates](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/templates.html). For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

### POST /templates

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates", host)
request.Method = "POST"
request.Body = []byte(` {
  "name": "example_name"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all transactional templates (legacy & dynamic).

**This endpoint allows you to retrieve all transactional templates.**

Each user can create up to 300 different transactional templates. Transactional templates are specific to accounts and subusers. Templates created on a parent account will not be accessible from the subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns templates](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/templates.html). For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

### GET /templates

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["generations"] = "legacy,dynamic"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Edit a transactional template.

**This endpoint allows you to edit a transactional template.**

Each user can create up to 300 different transactional templates. Transactional templates are specific to accounts and subusers. Templates created on a parent account will not be accessible from the subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns templates](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/templates.html). For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).


### PATCH /templates/{template_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "name": "new_example_name"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a single transactional template.

**This endpoint allows you to retrieve a single transactional template.**

Each user can create up to 300 different transactional templates. Transactional templates are specific to accounts and subusers. Templates created on a parent account will not be accessible from the subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns templates](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/templates.html). For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).


### GET /templates/{template_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a template.

**This endpoint allows you to delete a transactional template.**

Each user can create up to 300 different transactional templates. Transactional templates are specific to accounts and subusers. Templates created on a parent account will not be accessible from the subuser accounts.

Transactional templates are templates created specifically for transactional email and are not to be confused with [Marketing Campaigns templates](https://sendgrid.com/docs/User_Guide/Marketing_Campaigns/templates.html). For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).


### DELETE /templates/{template_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create a new transactional template version.

**This endpoint allows you to create a new version of a template.**

Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across all templates.

For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).


### POST /templates/{template_id}/versions

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}/versions", host)
request.Method = "POST"
request.Body = []byte(` {
  "active": 1,
  "html_content": "<%body%>",
  "name": "example_version_name",
  "plain_content": "<%body%>",
  "subject": "<%subject%>",
  "template_id": "ddb96bbc-9b92-425e-8979-99464621b543"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Edit a transactional template version.

**This endpoint allows you to edit a version of one of your transactional templates.**

Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across all templates.

For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

## URI Parameters
| URI Parameter | Type | Description |
|---|---|---|
| template_id | string | The ID of the original template |
| version_id | string | The ID of the template version |

### PATCH /templates/{template_id}/versions/{version_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}/versions/{version_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "active": 1,
  "html_content": "<%body%>",
  "name": "updated_example_name",
  "plain_content": "<%body%>",
  "subject": "<%subject%>"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific transactional template version.

**This endpoint allows you to retrieve a specific version of a template.**

Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across across all templates.

For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

## URI Parameters
| URI Parameter | Type | Description |
|---|---|---|
| template_id | string | The ID of the original template |
| version_id | string |  The ID of the template version |

### GET /templates/{template_id}/versions/{version_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}/versions/{version_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a transactional template version.

**This endpoint allows you to delete one of your transactional template versions.**

Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across across all templates.

For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

## URI Parameters
| URI Parameter | Type | Description |
|---|---|---|
| template_id | string | The ID of the original template |
| version_id | string | The ID of the template version |

### DELETE /templates/{template_id}/versions/{version_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}/versions/{version_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Activate a transactional template version.

**This endpoint allows you to activate a version of one of your templates.**

Each transactional template can have multiple versions, each version with its own subject and content. Each user can have up to 300 versions across across all templates.


For more information about transactional templates, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Transactional_Templates/index.html).

## URI Parameters
| URI Parameter | Type | Description |
|---|---|---|
| template_id | string | The ID of the original template |
| version_id | string |  The ID of the template version |

### POST /templates/{template_id}/versions/{version_id}/activate

```go
request := sendgrid.GetRequest(apiKey, "/v3/templates/{template_id}/versions/{version_id}/activate", host)
request.Method = "POST"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="tracking-settings"></a>
# TRACKING SETTINGS

## Retrieve Tracking Settings

**This endpoint allows you to retrieve a list of all tracking settings that you can enable on your account.**

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### GET /tracking_settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["limit"] = "1"
queryParams["offset"] = "1"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Click Tracking Settings

**This endpoint allows you to change your current click tracking setting. You can enable, or disable, click tracking using this endpoint.**

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### PATCH /tracking_settings/click

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/click", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Click Track Settings

**This endpoint allows you to retrieve your current click tracking setting.**

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### GET /tracking_settings/click

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/click", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Google Analytics Settings

**This endpoint allows you to update your current setting for Google Analytics.**

For more information about using Google Analytics, please refer to [Googles URL Builder](https://support.google.com/analytics/answer/1033867?hl=en) and their article on ["Best Practices for Campaign Building"](https://support.google.com/analytics/answer/1037445).

We default the settings to Googles recommendations. For more information, see [Google Analytics Demystified](https://sendgrid.com/docs/Classroom/Track/Collecting_Data/google_analytics_demystified_ga_statistics_vs_sg_statistics.html).

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### PATCH /tracking_settings/google_analytics

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/google_analytics", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "utm_campaign": "website",
  "utm_content": "",
  "utm_medium": "email",
  "utm_source": "sendgrid.com",
  "utm_term": ""
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Google Analytics Settings

**This endpoint allows you to retrieve your current setting for Google Analytics.**

For more information about using Google Analytics, please refer to [Googles URL Builder](https://support.google.com/analytics/answer/1033867?hl=en) and their article on ["Best Practices for Campaign Building"](https://support.google.com/analytics/answer/1037445).

We default the settings to Googles recommendations. For more information, see [Google Analytics Demystified](https://sendgrid.com/docs/Classroom/Track/Collecting_Data/google_analytics_demystified_ga_statistics_vs_sg_statistics.html).

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### GET /tracking_settings/google_analytics

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/google_analytics", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Open Tracking Settings

**This endpoint allows you to update your current settings for open tracking.**

Open Tracking adds an invisible image at the end of the email which can track email opens. If the email recipient has images enabled on their email client, a request to Twilio SendGrid's server for the invisible image is executed and an open event is logged. These events are logged in the Statistics portal, Email Activity interface, and are reported by the Event Webhook.

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### PATCH /tracking_settings/open

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/open", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Get Open Tracking Settings

**This endpoint allows you to retrieve your current settings for open tracking.**

Open Tracking adds an invisible image at the end of the email which can track email opens. If the email recipient has images enabled on their email client, a request to Twilio SendGrid's server for the invisible image is executed and an open event is logged. These events are logged in the Statistics portal, Email Activity interface, and are reported by the Event Webhook.

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### GET /tracking_settings/open

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/open", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Subscription Tracking Settings

**This endpoint allows you to update your current settings for subscription tracking.**

Subscription tracking adds links to the bottom of your emails that allows your recipients to subscribe to, or unsubscribe from, your emails.

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### PATCH /tracking_settings/subscription

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/subscription", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "enabled": true,
  "html_content": "html content",
  "landing": "landing page html",
  "plain_content": "text content",
  "replace": "replacement tag",
  "url": "url"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Subscription Tracking Settings

**This endpoint allows you to retrieve your current settings for subscription tracking.**

Subscription tracking adds links to the bottom of your emails that allows your recipients to subscribe to, or unsubscribe from, your emails.

You can track a variety of the actions your recipients may take when interacting with your emails including opening your emails, clicking on links in your emails, and subscribing to (or unsubscribing from) your emails.

For more information about tracking, please see our [User Guide](https://sendgrid.com/docs/User_Guide/Settings/tracking.html).

### GET /tracking_settings/subscription

```go
request := sendgrid.GetRequest(apiKey, "/v3/tracking_settings/subscription", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

<a name="on-behalf-of"></a>
# On-Behalf of Subuser

The `on-behalf-of` header allows you to make calls for a particular subuser through the parent account. This can be 
useful for automating bulk updates or administering a subuser without changing authentication in your code.

## Note
The v3/mail/send endpoint does not support the `on-behalf-of` header. ([Source](
    https://docs.sendgrid.com/api-reference/how-to-use-the-sendgrid-v3-api/on-behalf-of-subuser
))

```go

request := sendgrid.GetRequestSubuser(
	os.Getenv("SENDGRID_API_KEY"), "/v3/tracking_settings/subscription", 
	"https://api.sendgrid.com", "SUBUSER_USERNAME",
)

```

<a name="user"></a>
# USER

## Get a user's account information.

**This endpoint allows you to retrieve your user account details.**

Your user's account information includes the user's account type and reputation.

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### GET /user/account

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/account", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve your credit balance

**This endpoint allows you to retrieve the current credit balance for your account.**

Your monthly credit allotment limits the number of emails you may send before incurring overage charges. For more information about credits and billing, please visit our [Classroom](https://sendgrid.com/docs/Classroom/Basics/Billing/billing_info_and_faqs.html).

### GET /user/credits

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/credits", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update your account email address

**This endpoint allows you to update the email address currently on file for your account.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### PUT /user/email

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/email", host)
request.Method = "PUT"
request.Body = []byte(` {
  "email": "example@example.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve your account email address

**This endpoint allows you to retrieve the email address currently on file for your account.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### GET /user/email

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/email", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update your password

**This endpoint allows you to update your password.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### PUT /user/password

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/password", host)
request.Method = "PUT"
request.Body = []byte(` {
  "new_password": "new_password",
  "old_password": "old_password"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a user's profile

**This endpoint allows you to update your current profile details.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

It should be noted that any one or more of the parameters can be updated via the PATCH /user/profile endpoint. The only requirement is that you include at least one when you PATCH.

### PATCH /user/profile

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/profile", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "city": "Orange",
  "first_name": "Example",
  "last_name": "User"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Get a user's profile

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### GET /user/profile

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/profile", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Cancel or pause a scheduled send

**This endpoint allows you to cancel or pause an email that has been scheduled to be sent.**

If the maximum number of cancellations/pauses are added, HTTP 400 will
be returned.

The Cancel Scheduled Sends feature allows the customer to cancel a scheduled send based on a Batch ID included in the SMTPAPI header. Scheduled sends canceled less than 10 minutes before the scheduled time are not guaranteed to be canceled.

### POST /user/scheduled_sends

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/scheduled_sends", host)
request.Method = "POST"
request.Body = []byte(` {
  "batch_id": "YOUR_BATCH_ID",
  "status": "pause"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all scheduled sends

**This endpoint allows you to retrieve all cancel/paused scheduled send information.**

The Cancel Scheduled Sends feature allows the customer to cancel a scheduled send based on a Batch ID included in the SMTPAPI header. Scheduled sends canceled less than 10 minutes before the scheduled time are not guaranteed to be canceled.

### GET /user/scheduled_sends

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/scheduled_sends", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update user scheduled send information

**This endpoint allows you to update the status of a scheduled send for the given `batch_id`.**

The Cancel Scheduled Sends feature allows the customer to cancel a scheduled send based on a Batch ID included in the SMTPAPI header. Scheduled sends canceled less than 10 minutes before the scheduled time are not guaranteed to be canceled.

### PATCH /user/scheduled_sends/{batch_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/scheduled_sends/{batch_id}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "status": "pause"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve scheduled send

**This endpoint allows you to retrieve the cancel/paused scheduled send information for a specific `batch_id`.**

The Cancel Scheduled Sends feature allows the customer to cancel a scheduled send based on a Batch ID included in the SMTPAPI header. Scheduled sends canceled less than 10 minutes before the scheduled time are not guaranteed to be canceled.

### GET /user/scheduled_sends/{batch_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/scheduled_sends/{batch_id}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a cancellation or pause of a scheduled send

**This endpoint allows you to delete the cancellation/pause of a scheduled send.**

The Cancel Scheduled Sends feature allows the customer to cancel a scheduled send based on a Batch ID included in the SMTPAPI header. Scheduled sends canceled less than 10 minutes before the scheduled time are not guaranteed to be canceled.

### DELETE /user/scheduled_sends/{batch_id}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/scheduled_sends/{batch_id}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Enforced TLS settings

**This endpoint allows you to update your current Enforced TLS settings.**

The Enforced TLS settings specify whether or not the recipient is required to support TLS or have a valid certificate. See the [SMTP Ports User Guide](https://sendgrid.com/docs/Classroom/Basics/Email_Infrastructure/smtp_ports.html) for more information on opportunistic TLS.

**Note:** If either setting is enabled and the recipient does not support TLS or have a valid certificate, we drop the message and send a block event with TLS required but not supported as the description.

### PATCH /user/settings/enforced_tls

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/settings/enforced_tls", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "require_tls": true,
  "require_valid_cert": false
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve current Enforced TLS settings.

**This endpoint allows you to retrieve your current Enforced TLS settings.**

The Enforced TLS settings specify whether or not the recipient is required to support TLS or have a valid certificate. See the [SMTP Ports User Guide](https://sendgrid.com/docs/Classroom/Basics/Email_Infrastructure/smtp_ports.html) for more information on opportunistic TLS.

**Note:** If either setting is enabled and the recipient does not support TLS or have a valid certificate, we drop the message and send a block event with TLS required but not supported as the description.

### GET /user/settings/enforced_tls

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/settings/enforced_tls", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update your username

**This endpoint allows you to update the username for your account.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### PUT /user/username

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/username", host)
request.Method = "PUT"
request.Body = []byte(` {
  "username": "test_username"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve your username

**This endpoint allows you to retrieve your current account username.**

Keeping your user profile up to date is important. This will help Twilio SendGrid to verify who you are as well as contact you should we need to.

For more information about your user profile:

* [Twilio SendGrid Account Settings](https://sendgrid.com/docs/User_Guide/Settings/account.html)

### GET /user/username

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/username", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update Event Notification Settings

**This endpoint allows you to update your current event webhook settings.**

If an event type is marked as `true`, then the event webhook will include information about that event.

Twilio SendGrid's Event Webhook will notify a URL of your choice via HTTP POST with information about events that occur as Twilio SendGrid processes your email.

Common uses of this data are to remove unsubscribes, react to spam reports, determine unengaged recipients, identify bounced email addresses, or create advanced analytics of your email program.

### PATCH /user/webhooks/event/settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/event/settings", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "bounce": true,
  "click": true,
  "deferred": true,
  "delivered": true,
  "dropped": true,
  "enabled": true,
  "group_resubscribe": true,
  "group_unsubscribe": true,
  "open": true,
  "processed": true,
  "spam_report": true,
  "unsubscribe": true,
  "url": "url"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve Event Webhook settings

**This endpoint allows you to retrieve your current event webhook settings.**

If an event type is marked as `true`, then the event webhook will include information about that event.

Twilio SendGrid's Event Webhook will notify a URL of your choice via HTTP POST with information about events that occur as Twilio SendGrid processes your email.

Common uses of this data are to remove unsubscribes, react to spam reports, determine unengaged recipients, identify bounced email addresses, or create advanced analytics of your email program.

### GET /user/webhooks/event/settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/event/settings", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Test Event Notification Settings

**This endpoint allows you to test your event webhook by sending a fake event notification post to the provided URL.**

Twilio SendGrid's Event Webhook will notify a URL of your choice via HTTP POST with information about events that occur as Twilio SendGrid processes your email.

Common uses of this data are to remove unsubscribes, react to spam reports, determine unengaged recipients, identify bounced email addresses, or create advanced analytics of your email program.

### POST /user/webhooks/event/test

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/event/test", host)
request.Method = "POST"
request.Body = []byte(` {
  "url": "url"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Create a parse setting

**This endpoint allows you to create a new inbound parse setting.**

The inbound parse webhook allows you to have incoming emails parsed, extracting some or all of the content, and then have that content POSTed by Twilio SendGrid to a URL of your choosing. For more information, please see our [User Guide](https://sendgrid.com/docs/API_Reference/Webhooks/parse.html).

### POST /user/webhooks/parse/settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/settings", host)
request.Method = "POST"
request.Body = []byte(` {
  "hostname": "myhostname.com",
  "send_raw": false,
  "spam_check": true,
  "url": "http://email.myhosthame.com"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve all parse settings

**This endpoint allows you to retrieve all of your current inbound parse settings.**

The inbound parse webhook allows you to have incoming emails parsed, extracting some or all of the content, and then have that content POSTed by Twilio SendGrid to a URL of your choosing. For more information, please see our [User Guide](https://sendgrid.com/docs/API_Reference/Webhooks/parse.html).

### GET /user/webhooks/parse/settings

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/settings", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Update a parse setting

**This endpoint allows you to update a specific inbound parse setting.**

The inbound parse webhook allows you to have incoming emails parsed, extracting some or all of the content, and then have that content POSTed by Twilio SendGrid to a URL of your choosing. For more information, please see our [User Guide](https://sendgrid.com/docs/API_Reference/Webhooks/parse.html).

### PATCH /user/webhooks/parse/settings/{hostname}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/settings/{hostname}", host)
request.Method = "PATCH"
request.Body = []byte(` {
  "send_raw": true,
  "spam_check": false,
  "url": "http://newdomain.com/parse"
}`)
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieve a specific parse setting

**This endpoint allows you to retrieve a specific inbound parse setting.**

The inbound parse webhook allows you to have incoming emails parsed, extracting some or all of the content, and then have that content POSTed by Twilio SendGrid to a URL of your choosing. For more information, please see our [User Guide](https://sendgrid.com/docs/API_Reference/Webhooks/parse.html).

### GET /user/webhooks/parse/settings/{hostname}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/settings/{hostname}", host)
request.Method = "GET"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Delete a parse setting

**This endpoint allows you to delete a specific inbound parse setting.**

The inbound parse webhook allows you to have incoming emails parsed, extracting some or all of the content, and then have that content POSTed by Twilio SendGrid to a URL of your choosing. For more information, please see our [User Guide](https://sendgrid.com/docs/API_Reference/Webhooks/parse.html).

### DELETE /user/webhooks/parse/settings/{hostname}

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/settings/{hostname}", host)
request.Method = "DELETE"
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```

## Retrieves Inbound Parse Webhook statistics.

**This endpoint allows you to retrieve the statistics for your Parse Webhook usage.**

Twilio SendGrid's Inbound Parse Webhook allows you to parse the contents and attachments of incoming emails. The Parse API can then POST the parsed emails to a URL that you specify. The Inbound Parse Webhook cannot parse messages greater than 20MB in size, including all attachments.

There are a number of pre-made integrations for the Twilio SendGrid Parse Webhook which make processing events easy. You can find these integrations in the [Library Index](https://sendgrid.com/docs/Integrate/libraries.html#-Webhook-Libraries).

### GET /user/webhooks/parse/stats

```go
request := sendgrid.GetRequest(apiKey, "/v3/user/webhooks/parse/stats", host)
request.Method = "GET"
queryParams := make(map[string]string)
queryParams["aggregated_by"] = "day"
queryParams["limit"] = "test_string"
queryParams["start_date"] = "2016-01-01"
queryParams["end_date"] = "2016-04-01"
queryParams["offset"] = "test_string"
request.QueryParams = queryParams
response, err := sendgrid.API(request)
if err != nil {
  log.Println(err)
} else {
  fmt.Println(response.StatusCode)
  fmt.Println(response.Body)
  fmt.Println(response.Headers)
}
```
