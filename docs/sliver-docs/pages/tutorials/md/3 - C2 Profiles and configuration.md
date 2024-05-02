# Advanced web traffic configuration

When generating implants sliver uses a C2Profile configuration, which will be use to generate the effective network configuration of the implant. For example if configured to use /admin and /demo as callback urls, it might use one, the other or both allowing two implants using the same configuration to still seem slightly different from a network traffic perspective.

C2 profile configurations can be seen using the `c2profile` command, which also allows import and export features.

The full list of possible configuration option can be found in the references section below, but for now lets instead customise the existing configuration.

Lets imagine we’re trying to breach a customer known for using ruby-on-rails. By default sliver will use:

- `.woff` for staging
- `.js` for poll requests
- `.html` for key exchanges
- `.png` for close session
- `.php` for session messages

Let’s go ahead and update the session messages and staging with something more realistic and remove all references to woff or php.

```bash
"session_file_ext": ".css",
"stager_file_ext": ".ico",
```

TODO pull urls for ror, maybe from seclists ? 

The next step is to restart the http listener and generate our new implant.

```bash
TODO
asciinema export c2profile, updating extensions and paths
```

TODO
asciinema import custom c2profile, restart job and spin new beacon

If you now look at the debug output you’ll notice we no longer have .php urls.

```bash
2023/04/25 15:27:41 httpclient.go:672: [http] segments = [oauth2 v1 authenticate auth], filename = index, ext = css
2023/04/25 15:27:41 httpclient.go:482: [http] POST -> http://localhost/oauth2/v1/authenticate/auth/index.css?p=711x58387 (2228 bytes)
2023/04/25 15:27:41 httpclient.go:488: [http] POST request completed
2023/04/25 15:27:42 httpclient.go:287: Cancelling poll context
2023/04/25 15:27:42 httpclient.go:672: [http] segments = [assets], filename = jquery, ext = js
2023/04/25 15:27:42 httpclient.go:406: [http] GET -> http://localhost/assets/jquery.js?r=72074674
2023/04/25 15:27:42 sliver.go:198: [recv] sysHandler 12
2023/04/25 15:27:42 session.go:189: [http] send envelope ...
2023/04/25 15:27:42 httpclient.go:672: [http] segments = [oauth v1 oauth2], filename = admin, ext = css
2023/04/25 15:27:42 httpclient.go:482: [http] POST -> http://localhost/oauth/v1/oauth2/admin.css?j=56685386 (93 bytes)
```

Ideally during engagements your recon phase should inform your C2 infrastructure, reusing similar hosting providers, technologies and communication protocols can help your implant fly under the radar. 