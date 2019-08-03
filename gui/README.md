# Sliver GUI

Electron based Sliver GUI written in Angular/TypeScript.

## Design Goals

### Why Electron?

Because I value my development time more than your RAM.

### Security

This is my attempt at making a _reasonably_ secure Electron application. High level design is:

* __Sandboxed__ - The main WebView does NOT have `nodeIntegration` enabled; the WebView cannot directly execute native code, access the file system, etc. it has to go thru the IPC interface to perform any actions a browser normally could not. The IPC interface is called via `window.postMessage()` with `contextIsolation` enabled so there are no direct references to Node objects within the sandbox.
* __No HTTP__ - The sandboxed code does not talk to the server over HTTP. Instead it uses IPC to talk to the native Node process, which then converts the call into RPC (Protobuf over mTLS). There are no locally running HTTP servers and thus no HTTP cross-origin conerns. However, [due to a bug in Electron](https://github.com/electron/electron/issues/19603) it's possible for plugin scripts to control URI parameters. So care must be take to ensure URI parameters cannot cause a state changing event.
* __CSP__ - Strong CSP by default: `default-src none`, no direct interaction with the DOM, Angular handles all content rendering.
* __Navigation__: Navigation and redirects are disabled in all windows.
* __App Origin__: No webviews run with a `file://` origin, nor an `http://` origin, etc. Webviews run in either a `null` origin (plugin scripts) or within an `app://sliver` origin. In combination with CSP, this means the main webview cannot access any `file:` or `http:` resources.

```
                                                      |----------------- Electron ---------------|
[implant] <-(mTLS/DNS/HTTP)-> [server] <-(RPC/mTLS)-> [node process] <--(IPC)--> [browser process]
```

## Build

* Node v12.4.0
* npm v6.9.0

From this directory:

```bash
npm install
npm install -g electron-packager
npm install -g ts-protoc-gen
npm install -g @angular/cli
./protobuf.sh
npm run electron:local
```

## Source Code

Source code is organized as follows:

`main.ts` - Electron entrypoint.

`preload.js` - Electron preload script used to bridge the sandbox code to the Node process.

`ipc/` - Node IPC handler code, this translates messages from the `preload.js` script into RPC or local procedure calls that cannot be done from within the sandbox.

`rpc/` - TypeScript implementation of Sliver's RPC protocol.

`src/` - Angular source code (webview code).
