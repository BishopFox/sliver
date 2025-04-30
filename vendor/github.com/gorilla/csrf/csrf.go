package csrf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/gorilla/securecookie"
)

// CSRF token length in bytes.
const tokenLength = 32

// Context/session keys & prefixes
const (
	tokenKey     string = "gorilla.csrf.Token" // #nosec G101
	formKey      string = "gorilla.csrf.Form"  // #nosec G101
	errorKey     string = "gorilla.csrf.Error"
	skipCheckKey string = "gorilla.csrf.Skip"
	cookieName   string = "_gorilla_csrf"
	errorPrefix  string = "gorilla/csrf: "
)

type contextKey string

// PlaintextHTTPContextKey is the context key used to store whether the request
// is being served via plaintext HTTP. This is used to signal to the middleware
// that strict Referer checking should not be enforced as is done for HTTPS by
// default.
const PlaintextHTTPContextKey contextKey = "plaintext"

var (
	// The name value used in form fields.
	fieldName = tokenKey
	// defaultAge sets the default MaxAge for cookies.
	defaultAge = 3600 * 12
	// The default HTTP request header to inspect
	headerName = "X-CSRF-Token"
	// Idempotent (safe) methods as defined by RFC7231 section 4.2.2.
	safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
)

// TemplateTag provides a default template tag - e.g. {{ .csrfField }} - for use
// with the TemplateField function.
var TemplateTag = "csrfField"

var (
	// ErrNoReferer is returned when a HTTPS request provides an empty Referer
	// header.
	ErrNoReferer = errors.New("referer not supplied")
	// ErrBadOrigin is returned when the Origin header is present and is not a
	// trusted origin.
	ErrBadOrigin = errors.New("origin invalid")
	// ErrBadReferer is returned when the scheme & host in the URL do not match
	// the supplied Referer header.
	ErrBadReferer = errors.New("referer invalid")
	// ErrNoToken is returned if no CSRF token is supplied in the request.
	ErrNoToken = errors.New("CSRF token not found in request")
	// ErrBadToken is returned if the CSRF token in the request does not match
	// the token in the session, or is otherwise malformed.
	ErrBadToken = errors.New("CSRF token invalid")
)

// SameSiteMode allows a server to define a cookie attribute making it impossible for
// the browser to send this cookie along with cross-site requests. The main
// goal is to mitigate the risk of cross-origin information leakage, and provide
// some protection against cross-site request forgery attacks.
//
// See https://tools.ietf.org/html/draft-ietf-httpbis-cookie-same-site-00 for details.
type SameSiteMode int

// SameSite options
const (
	// SameSiteDefaultMode sets the `SameSite` cookie attribute, which is
	// invalid in some older browsers due to changes in the SameSite spec. These
	// browsers will not send the cookie to the server.
	// csrf uses SameSiteLaxMode (SameSite=Lax) as the default as of v1.7.0+
	SameSiteDefaultMode SameSiteMode = iota + 1
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

type csrf struct {
	h    http.Handler
	sc   *securecookie.SecureCookie
	st   store
	opts options
}

// options contains the optional settings for the CSRF middleware.
type options struct {
	MaxAge int
	Domain string
	Path   string
	// Note that the function and field names match the case of the associated
	// http.Cookie field instead of the "correct" HTTPOnly name that golint suggests.
	HttpOnly       bool
	Secure         bool
	SameSite       SameSiteMode
	RequestHeader  string
	FieldName      string
	ErrorHandler   http.Handler
	CookieName     string
	TrustedOrigins []string
}

// Protect is HTTP middleware that provides Cross-Site Request Forgery
// protection.
//
// It securely generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original (unmasked) token is stored in the session, which is inaccessible
// by an attacker (provided you are using HTTPS). Subsequent requests are
// expected to include this token, which is compared against the session token.
// Requests that do not provide a matching token are served with a HTTP 403
// 'Forbidden' error response.
//
// Example:
//
//	package main
//
//	import (
//		"html/template"
//
//		"github.com/gorilla/csrf"
//		"github.com/gorilla/mux"
//	)
//
//	var t = template.Must(template.New("signup_form.tmpl").Parse(form))
//
//	func main() {
//		r := mux.NewRouter()
//
//		r.HandleFunc("/signup", GetSignupForm)
//		// POST requests without a valid token will return a HTTP 403 Forbidden.
//		r.HandleFunc("/signup/post", PostSignupForm)
//
//		// Add the middleware to your router.
//		http.ListenAndServe(":8000",
//		// Note that the authentication key provided should be 32 bytes
//		// long and persist across application restarts.
//			  csrf.Protect([]byte("32-byte-long-auth-key"))(r))
//	}
//
//	func GetSignupForm(w http.ResponseWriter, r *http.Request) {
//		// signup_form.tmpl just needs a {{ .csrfField }} template tag for
//		// csrf.TemplateField to inject the CSRF token into. Easy!
//		t.ExecuteTemplate(w, "signup_form.tmpl", map[string]interface{}{
//			csrf.TemplateTag: csrf.TemplateField(r),
//		})
//		// We could also retrieve the token directly from csrf.Token(r) and
//		// set it in the request header - w.Header.Set("X-CSRF-Token", token)
//		// This is useful if you're sending JSON to clients or a front-end JavaScript
//		// framework.
//	}
func Protect(authKey []byte, opts ...Option) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		cs := parseOptions(h, opts...)

		// Set the defaults if no options have been specified
		if cs.opts.ErrorHandler == nil {
			cs.opts.ErrorHandler = http.HandlerFunc(unauthorizedHandler)
		}

		if cs.opts.MaxAge < 0 {
			// Default of 12 hours
			cs.opts.MaxAge = defaultAge
		}

		if cs.opts.FieldName == "" {
			cs.opts.FieldName = fieldName
		}

		if cs.opts.CookieName == "" {
			cs.opts.CookieName = cookieName
		}

		if cs.opts.RequestHeader == "" {
			cs.opts.RequestHeader = headerName
		}

		// Create an authenticated securecookie instance.
		if cs.sc == nil {
			cs.sc = securecookie.New(authKey, nil)
			// Use JSON serialization (faster than one-off gob encoding)
			cs.sc.SetSerializer(securecookie.JSONEncoder{})
			// Set the MaxAge of the underlying securecookie.
			cs.sc.MaxAge(cs.opts.MaxAge)
		}

		if cs.st == nil {
			// Default to the cookieStore
			cs.st = &cookieStore{
				name:     cs.opts.CookieName,
				maxAge:   cs.opts.MaxAge,
				secure:   cs.opts.Secure,
				httpOnly: cs.opts.HttpOnly,
				sameSite: cs.opts.SameSite,
				path:     cs.opts.Path,
				domain:   cs.opts.Domain,
				sc:       cs.sc,
			}
		}

		return cs
	}
}

// Implements http.Handler for the csrf type.
func (cs *csrf) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Skip the check if directed to. This should always be a bool.
	if val, err := contextGet(r, skipCheckKey); err == nil {
		if skip, ok := val.(bool); ok {
			if skip {
				cs.h.ServeHTTP(w, r)
				return
			}
		}
	}

	// Retrieve the token from the session.
	// An error represents either a cookie that failed HMAC validation
	// or that doesn't exist.
	realToken, err := cs.st.Get(r)
	if err != nil || len(realToken) != tokenLength {
		// If there was an error retrieving the token, the token doesn't exist
		// yet, or it's the wrong length, generate a new token.
		// Note that the new token will (correctly) fail validation downstream
		// as it will no longer match the request token.
		realToken, err = generateRandomBytes(tokenLength)
		if err != nil {
			r = envError(r, err)
			cs.opts.ErrorHandler.ServeHTTP(w, r)
			return
		}

		// Save the new (real) token in the session store.
		err = cs.st.Save(realToken, w)
		if err != nil {
			r = envError(r, err)
			cs.opts.ErrorHandler.ServeHTTP(w, r)
			return
		}
	}

	// Save the masked token to the request context
	r = contextSave(r, tokenKey, mask(realToken, r))
	// Save the field name to the request context
	r = contextSave(r, formKey, cs.opts.FieldName)

	// HTTP methods not defined as idempotent ("safe") under RFC7231 require
	// inspection.
	if !contains(safeMethods, r.Method) {
		var isPlaintext bool
		val := r.Context().Value(PlaintextHTTPContextKey)
		if val != nil {
			isPlaintext, _ = val.(bool)
		}

		// take a copy of the request URL to avoid mutating the original
		// attached to the request.
		// set the scheme & host based on the request context as these are not
		// populated by default for server requests
		// ref: https://pkg.go.dev/net/http#Request
		requestURL := *r.URL // shallow clone

		requestURL.Scheme = "https"
		if isPlaintext {
			requestURL.Scheme = "http"
		}
		if requestURL.Host == "" {
			requestURL.Host = r.Host
		}

		// if we have an Origin header, check it against our allowlist
		origin := r.Header.Get("Origin")
		if origin != "" {
			parsedOrigin, err := url.Parse(origin)
			if err != nil {
				r = envError(r, ErrBadOrigin)
				cs.opts.ErrorHandler.ServeHTTP(w, r)
				return
			}
			if !sameOrigin(&requestURL, parsedOrigin) && !slices.Contains(cs.opts.TrustedOrigins, parsedOrigin.Host) {
				r = envError(r, ErrBadOrigin)
				cs.opts.ErrorHandler.ServeHTTP(w, r)
				return
			}
		}

		// If we are serving via TLS and have no Origin header, prevent against
		// CSRF via HTTP machine in the middle attacks by enforcing strict
		// Referer origin checks. Consider an attacker who performs a
		// successful HTTP Machine-in-the-Middle attack and uses this to inject
		// a form and cause submission to our origin. We strictly disallow
		// cleartext HTTP origins and evaluate the domain against an allowlist.
		if origin == "" && !isPlaintext {
			// Fetch the Referer value. Call the error handler if it's empty or
			// otherwise fails to parse.
			referer, err := url.Parse(r.Referer())
			if err != nil || referer.String() == "" {
				r = envError(r, ErrNoReferer)
				cs.opts.ErrorHandler.ServeHTTP(w, r)
				return
			}

			// disallow cleartext HTTP referers when serving via TLS
			if referer.Scheme == "http" {
				r = envError(r, ErrBadReferer)
				cs.opts.ErrorHandler.ServeHTTP(w, r)
				return
			}

			// If the request is being served via TLS and the Referer is not the
			// same origin, check the domain against our allowlist. We only
			// check when we have host information from the referer.
			if referer.Host != "" && referer.Host != r.Host && !slices.Contains(cs.opts.TrustedOrigins, referer.Host) {
				r = envError(r, ErrBadReferer)
				cs.opts.ErrorHandler.ServeHTTP(w, r)
				return
			}
		}

		// Retrieve the combined token (pad + masked) token...
		maskedToken, err := cs.requestToken(r)
		if err != nil {
			r = envError(r, ErrBadToken)
			cs.opts.ErrorHandler.ServeHTTP(w, r)
			return
		}

		if maskedToken == nil {
			r = envError(r, ErrNoToken)
			cs.opts.ErrorHandler.ServeHTTP(w, r)
			return
		}

		// ... and unmask it.
		requestToken := unmask(maskedToken)

		// Compare the request token against the real token
		if !compareTokens(requestToken, realToken) {
			r = envError(r, ErrBadToken)
			cs.opts.ErrorHandler.ServeHTTP(w, r)
			return
		}

	}

	// Set the Vary: Cookie header to protect clients from caching the response.
	w.Header().Add("Vary", "Cookie")

	// Call the wrapped handler/router on success.
	cs.h.ServeHTTP(w, r)
	// Clear the request context after the handler has completed.
	contextClear(r)
}

// PlaintextHTTPRequest accepts as input a http.Request and returns a new
// http.Request with the PlaintextHTTPContextKey set to true. This is used to
// signal to the CSRF middleware that the request is being served over plaintext
// HTTP and that Referer-based origin allow-listing checks should be skipped.
func PlaintextHTTPRequest(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), PlaintextHTTPContextKey, true)
	return r.WithContext(ctx)
}

// unauthorizedhandler sets a HTTP 403 Forbidden status and writes the
// CSRF failure reason to the response.
func unauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("%s - %s",
		http.StatusText(http.StatusForbidden), FailureReason(r)),
		http.StatusForbidden)
}
