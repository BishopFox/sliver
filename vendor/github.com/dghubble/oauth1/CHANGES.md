# OAuth1 Changelog

Notable changes between releases.

## Latest

## v0.7.3

* Percent encode special characters in HMAC-SHA1 secrets ([#72](https://github.com/dghubble/oauth1/pull/72))
* Strip whitespace from request token body ([#56](https://github.com/dghubble/oauth1/pull/56))
* Update Go module dependencies

## v0.7.2

* Update minimum Go version from v1.17 to v1.18 ([#66](https://github.com/dghubble/oauth1/pull/66))

## v0.7.1

* Show body when `RequestToken` or `AccessToken` requests return an invalid status code ([#54](https://github.com/dghubble/oauth1/pull/54))

## v0.7.0

* Add an `HMAC256Signer` ([#40](https://github.com/dghubble/oauth1/pull/40))
* Add discogs `Endpoint` ([#39](https://github.com/dghubble/oauth1/pull/39))
* Allow custom `Noncer` for unusual OAuth1 providers ([#45](https://github.com/dghubble/oauth1/pull/45)
* Change tumblr `Endpoint` URLs to https ([#37](https://github.com/dghubble/oauth1/pull/37))

## v0.6.0

* Add Go module support ([#32](https://github.com/dghubble/oauth1/pull/32))

## v0.5.0

* Use standard library `context` ([c0a405](https://github.com/dghubble/oauth1/commit/c0a405baf29f5ed2616bc1ef6b778532c960aa5b))
  * Requires Go 1.7+
* Add `xing` package with a provider `Endpoint` ([#10](https://github.com/dghubble/oauth1/pull/10))
* Add status code checks so server errors are clearer ([09fded](https://github.com/dghubble/oauth1/commit/b0d5c93a5292844f3fd568893ce4e12bdcdb79ae))
* Move confirmed check after token check so errors are clearer ([#8](https://github.com/dghubble/oauth1/pull/8))

## v0.4.0

* Add a Signer field to the Config to allow custom Signer implementations.
* Use the HMACSigner by default. This provides the same signing behavior as in previous versions (HMAC-SHA1).
* Add an RSASigner for "RSA-SHA1" OAuth1 Providers.
* Add missing Authorization Header quotes around OAuth parameter values. Many providers allowed these quotes to be missing.
* Change `Signer` to be a signer interface.
* Remove the old Signer methods `SetAccessTokenAuthHeader`, `SetRequestAuthHeader`, and `SetRequestTokenAuthHeader`.

## v0.3.0

* Added `NoContext` which may be used in most cases.
* Allowed Transport Base http.RoundTripper to be set through a ctx.
* Changed `NewClient` to require a context.Context.
* Changed `Config.Client` to require a context.Context.

## v.0.2.0

* Improved OAuth 1 spec compliance and test coverage.
* Added `func StaticTokenSource(*Token) TokenSource`
* Added `ParseAuthorizationCallback` function. Removed `Config.HandleAuthorizationCallback` method.
* Changed `Config` method signatures to allow an interface to be defined for the OAuth1 authorization flow. Gives users of this package (and downstream packages) the freedom to use other implementations if they wish.
* Removed `RequestToken` in favor of passing token and secret value strings.
* Removed `ReuseTokenSource` struct, it was effectively a static source. Replaced by `StaticTokenSource`.

## v0.1.0

* Initial OAuth1 support for obtaining authorization and making authorized requests.
