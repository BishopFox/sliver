package reddit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-querystring/query"
	"golang.org/x/oauth2"
)

const (
	libraryName    = "github.com/caarlos0/go-reddit"
	libraryVersion = "2.0.0"

	defaultBaseURL         = "https://oauth.reddit.com"
	defaultBaseURLReadonly = "https://reddit.com"
	defaultTokenURL        = "https://www.reddit.com/api/v1/access_token"

	mediaTypeJSON = "application/json"
	mediaTypeForm = "application/x-www-form-urlencoded"

	headerContentType = "Content-Type"
	headerAccept      = "Accept"
	headerUserAgent   = "User-Agent"

	headerRateLimitRemaining = "x-ratelimit-remaining"
	headerRateLimitUsed      = "x-ratelimit-used"
	headerRateLimitReset     = "x-ratelimit-reset"
)

var defaultClient, _ = NewReadonlyClient()

// DefaultClient returns a valid, read-only client with limited access to the Reddit API.
func DefaultClient() *Client {
	return defaultClient
}

// RequestCompletionCallback defines the type of the request callback function.
type RequestCompletionCallback func(*http.Request, *http.Response)

// Credentials are used to authenticate to make requests to the Reddit API.
type Credentials struct {
	ID       string
	Secret   string
	Username string
	Password string
}

// Client manages communication with the Reddit API.
type Client struct {
	// HTTP client used to communicate with the Reddit API.
	client *http.Client

	BaseURL  *url.URL
	TokenURL *url.URL

	userAgent string

	rateMu sync.Mutex
	rate   Rate

	ID       string
	Secret   string
	Username string
	Password string

	// This is the client's user ID in Reddit's database.
	redditID string

	Account    *AccountService
	Collection *CollectionService
	Comment    *CommentService
	Emoji      *EmojiService
	Flair      *FlairService
	Gold       *GoldService
	Listings   *ListingsService
	LiveThread *LiveThreadService
	Message    *MessageService
	Moderation *ModerationService
	Multi      *MultiService
	Post       *PostService
	Stream     *StreamService
	Subreddit  *SubredditService
	User       *UserService
	Widget     *WidgetService
	Wiki       *WikiService

	oauth2Transport *oauth2.Transport

	onRequestCompleted RequestCompletionCallback
}

// OnRequestCompleted sets the client's request completion callback.
func (c *Client) OnRequestCompleted(rc RequestCompletionCallback) {
	c.onRequestCompleted = rc
}

func newClient() *Client {
	baseURL, _ := url.Parse(defaultBaseURL)
	tokenURL, _ := url.Parse(defaultTokenURL)

	client := &Client{client: &http.Client{}, BaseURL: baseURL, TokenURL: tokenURL}

	client.Account = &AccountService{client: client}
	client.Collection = &CollectionService{client: client}
	client.Emoji = &EmojiService{client: client}
	client.Flair = &FlairService{client: client}
	client.Gold = &GoldService{client: client}
	client.Listings = &ListingsService{client: client}
	client.LiveThread = &LiveThreadService{client: client}
	client.Message = &MessageService{client: client}
	client.Moderation = &ModerationService{client: client}
	client.Multi = &MultiService{client: client}
	client.Stream = &StreamService{client: client}
	client.Subreddit = &SubredditService{client: client}
	client.User = &UserService{client: client}
	client.Widget = &WidgetService{client: client}
	client.Wiki = &WikiService{client: client}

	postAndCommentService := &postAndCommentService{client: client}
	client.Comment = &CommentService{client: client, postAndCommentService: postAndCommentService}
	client.Post = &PostService{client: client, postAndCommentService: postAndCommentService}

	return client
}

// NewClient returns a new Reddit API client.
// Use an Opt to configure the client credentials, such as WithHTTPClient or WithUserAgent.
// If the FromEnv option is used with the correct environment variables, an empty struct can
// be passed in as the credentials, since they will be overridden.
func NewClient(credentials Credentials, opts ...Opt) (*Client, error) {
	client := newClient()

	client.ID = credentials.ID
	client.Secret = credentials.Secret
	client.Username = credentials.Username
	client.Password = credentials.Password

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	userAgentTransport := &userAgentTransport{
		userAgent: client.UserAgent(),
		Base:      client.client.Transport,
	}
	client.client.Transport = userAgentTransport

	if client.client.CheckRedirect == nil {
		client.client.CheckRedirect = client.redirect
	}

	oauthTransport := oauthTransport(client)
	client.client.Transport = oauthTransport

	return client, nil
}

// NewReadonlyClient returns a new read-only Reddit API client.
// The client will have limited access to the Reddit API.
// Options that modify credentials (such as FromEnv) won't have any effect on this client.
func NewReadonlyClient(opts ...Opt) (*Client, error) {
	client := newClient()
	client.BaseURL, _ = url.Parse(defaultBaseURLReadonly)

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	if client.client == nil {
		client.client = &http.Client{}
	}

	userAgentTransport := &userAgentTransport{
		userAgent: client.UserAgent(),
		Base:      client.client.Transport,
	}
	client.client.Transport = userAgentTransport

	return client, nil
}

// todo...
// Some endpoints (notably the ones to get random subreddits/posts) redirect to a
// reddit.com url, which returns a 403 Forbidden for some reason, unless the url's
// host is changed to oauth.reddit.com
func (c *Client) redirect(req *http.Request, via []*http.Request) error {
	redirectURL := req.URL.String()
	redirectURL = strings.Replace(redirectURL, "https://www.reddit.com", defaultBaseURL, 1)

	reqURL, err := url.Parse(redirectURL)
	if err != nil {
		return err
	}
	req.URL = reqURL

	return nil
}

// The readonly Reddit url needs .json at the end of its path to return responses in JSON instead of HTML.
func (c *Client) appendJSONExtensionToRequestURLPath(req *http.Request) {
	readonlyURL, err := url.Parse(defaultBaseURLReadonly)
	if err != nil {
		return
	}

	if req.URL.Host != readonlyURL.Host {
		return
	}

	req.URL.Path += ".json"
}

// UserAgent returns the client's user agent.
func (c *Client) UserAgent() string {
	if c.userAgent == "" {
		userAgent := fmt.Sprintf("golang:%s:v%s", libraryName, libraryVersion)
		if c.Username != "" {
			userAgent += fmt.Sprintf(" (by /u/%s)", c.Username)
		}
		c.userAgent = userAgent
	}
	return c.userAgent
}

// NewRequest creates an API request with form data as the body.
// The path is the relative URL which will be resolved to the BaseURL of the Client.
// It should always be specified without a preceding slash.
func (c *Client) NewRequest(method string, path string, form url.Values) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	c.appendJSONExtensionToRequestURLPath(req)
	req.Header.Add(headerContentType, mediaTypeForm)
	req.Header.Add(headerAccept, mediaTypeJSON)

	return req, nil
}

// NewJSONRequest creates an API request with a JSON body.
// The path is the relative URL which will be resolved to the BaseURL of the Client.
// It should always be specified without a preceding slash.
func (c *Client) NewJSONRequest(method string, path string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if body != nil {
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	reqBody := bytes.NewReader(buf.Bytes())
	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	c.appendJSONExtensionToRequestURLPath(req)
	req.Header.Add(headerContentType, mediaTypeJSON)
	req.Header.Add(headerAccept, mediaTypeJSON)

	return req, nil
}

// Response is a Reddit response. This wraps the standard http.Response returned from Reddit.
type Response struct {
	*http.Response

	// Pagination anchor indicating there are more results after this id.
	After string

	// Rate limit information.
	Rate Rate
}

// newResponse creates a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
	response := Response{Response: r}
	response.Rate = parseRate(r)
	return &response
}

func (r *Response) populateAnchors(a anchor) {
	r.After = a.After()
}

// parseRate parses the rate related headers.
func parseRate(r *http.Response) Rate {
	var rate Rate
	if remaining := r.Header.Get(headerRateLimitRemaining); remaining != "" {
		v, _ := strconv.ParseFloat(remaining, 64)
		rate.Remaining = int(v)
	}
	if used := r.Header.Get(headerRateLimitUsed); used != "" {
		rate.Used, _ = strconv.Atoi(used)
	}
	if reset := r.Header.Get(headerRateLimitReset); reset != "" {
		if v, _ := strconv.ParseInt(reset, 10, 64); v != 0 {
			rate.Reset = time.Now().Truncate(time.Second).Add(time.Second * time.Duration(v))
		}
	}
	return rate
}

// Do sends an API request and returns the API response. The API response is JSON decoded and stored in the value
// pointed to by v, or returned as an error if an API error has occurred. If v implements the io.Writer interface,
// the raw response will be written to v, without attempting to decode it.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	if err := c.checkRateLimitBeforeDo(req); err != nil {
		return &Response{
			Response: err.Response,
			Rate:     err.Rate,
		}, err
	}

	resp, err := DoRequestWithClient(ctx, c.client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if c.onRequestCompleted != nil {
		c.onRequestCompleted(req, resp)
	}

	response := newResponse(resp)

	c.rateMu.Lock()
	c.rate = response.Rate
	c.rateMu.Unlock()

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, response.Body)
			if err != nil {
				return response, err
			}
		} else {
			err = json.NewDecoder(response.Body).Decode(v)
			if err != nil {
				return response, err
			}
		}

		if anchor, ok := v.(anchor); ok {
			response.populateAnchors(anchor)
		}
	}

	return response, nil
}

func (c *Client) checkRateLimitBeforeDo(req *http.Request) *RateLimitError {
	c.rateMu.Lock()
	rate := c.rate
	c.rateMu.Unlock()

	if !rate.Reset.IsZero() && rate.Remaining == 0 && time.Now().Before(rate.Reset) {
		// Create a fake 429 response.
		resp := &http.Response{
			Status:     http.StatusText(http.StatusTooManyRequests),
			StatusCode: http.StatusTooManyRequests,
			Request:    req,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}
		return &RateLimitError{
			Rate:     rate,
			Response: resp,
			Message:  fmt.Sprintf("API rate limit still exceeded until %s, not making remote request.", rate.Reset),
		}
	}

	return nil
}

// id returns the client's Reddit ID.
func (c *Client) id(ctx context.Context) (string, *Response, error) {
	if c.redditID != "" {
		return c.redditID, nil, nil
	}

	self, resp, err := c.User.Get(ctx, c.Username)
	if err != nil {
		return "", resp, err
	}

	c.redditID = fmt.Sprintf("%s_%s", kindUser, self.ID)
	return c.redditID, resp, nil
}

// DoRequest submits an HTTP request.
func DoRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	return DoRequestWithClient(ctx, http.DefaultClient, req)
}

// DoRequestWithClient submits an HTTP request using the specified client.
func DoRequestWithClient(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return client.Do(req)
}

// CheckResponse checks the API response for errors, and returns them if present.
// A response is considered an error if it has a status code outside the 200 range.
// Reddit also sometimes sends errors with 200 codes; we check for those too.
func CheckResponse(r *http.Response) error {
	if r.Header.Get(headerRateLimitRemaining) == "0" {
		err := &RateLimitError{
			Rate:     parseRate(r),
			Response: r,
		}
		err.Message = fmt.Sprintf("API rate limit has been exceeded until %s.", err.Rate.Reset)
		return err
	}

	jsonErrorResponse := &JSONErrorResponse{Response: r}

	data, err := ioutil.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		json.Unmarshal(data, jsonErrorResponse)
		if len(jsonErrorResponse.JSON.Errors) > 0 {
			return jsonErrorResponse
		}
	}

	// reset response body
	r.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err = ioutil.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			errorResponse.Message = string(data)
		}
	}

	return errorResponse
}

// Rate represents the rate limit for the client.
type Rate struct {
	// The number of remaining requests the client can make in the current 10-minute window.
	Remaining int `json:"remaining"`
	// The number of requests the client has made in the current 10-minute window.
	Used int `json:"used"`
	// The time at which the current rate limit will reset.
	Reset time.Time `json:"reset"`
}

// A lot of Reddit's responses return a "thing": { "kind": "...", "data": {...} }
// So this is just a nice convenient method to have.
func (c *Client) getThing(ctx context.Context, path string, opts interface{}) (*thing, *Response, error) {
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	t := new(thing)
	resp, err := c.Do(ctx, req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}

func (c *Client) getListing(ctx context.Context, path string, opts interface{}) (*listing, *Response, error) {
	t, resp, err := c.getThing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	l, _ := t.Listing()
	return l, resp, nil
}

// ListOptions specifies the optional parameters to various API calls that return a listing.
type ListOptions struct {
	// Maximum number of items to be returned.
	// Generally, the default is 25 and max is 100.
	Limit int `url:"limit,omitempty"`

	// The full ID of an item in the listing to use
	// as the anchor point of the list. Only items
	// appearing after it will be returned.
	After string `url:"after,omitempty"`

	// The full ID of an item in the listing to use
	// as the anchor point of the list. Only items
	// appearing before it will be returned.
	Before string `url:"before,omitempty"`
}

// ListSubredditOptions defines possible options used when searching for subreddits.
type ListSubredditOptions struct {
	ListOptions
	// One of: relevance, activity.
	Sort string `url:"sort,omitempty"`
}

// ListPostOptions defines possible options used when getting posts from a subreddit.
type ListPostOptions struct {
	ListOptions
	// One of: hour, day, week, month, year, all.
	Time string `url:"t,omitempty"`
}

// ListPostSearchOptions defines possible options used when searching for posts within a subreddit.
type ListPostSearchOptions struct {
	ListPostOptions
	// One of: relevance, hot, top, new, comments.
	Sort string `url:"sort,omitempty"`
}

// ListUserOverviewOptions defines possible options used when getting a user's post and/or comments.
type ListUserOverviewOptions struct {
	ListOptions
	// One of: hot, new, top, controversial.
	Sort string `url:"sort,omitempty"`
	// One of: hour, day, week, month, year, all.
	Time string `url:"t,omitempty"`
}

// ListDuplicatePostOptions defines possible options used when getting duplicates of a post, i.e.
// other submissions of the same URL.
type ListDuplicatePostOptions struct {
	ListOptions
	// If empty, it'll search for duplicates in all subreddits.
	Subreddit string `url:"sr,omitempty"`
	// One of: num_comments, new.
	Sort string `url:"sort,omitempty"`
	// If true, the search will only return duplicates that are
	// crossposts of the original post.
	CrosspostsOnly bool `url:"crossposts_only,omitempty"`
}

// ListModActionOptions defines possible options used when getting moderation actions in a subreddit.
type ListModActionOptions struct {
	// The max for the limit parameter here is 500.
	ListOptions
	// If empty, the search will return all action types.
	// One of: banuser, unbanuser, spamlink, removelink, approvelink, spamcomment, removecomment,
	// approvecomment, addmoderator, showcomment, invitemoderator, uninvitemoderator, acceptmoderatorinvite,
	// removemoderator, addcontributor, removecontributor, editsettings, editflair, distinguish, marknsfw,
	// wikibanned, wikicontributor, wikiunbanned, wikipagelisted, removewikicontributor, wikirevise,
	// wikipermlevel, ignorereports, unignorereports, setpermissions, setsuggestedsort, sticky, unsticky,
	// setcontestmode, unsetcontestmode, lock, unlock, muteuser, unmuteuser, createrule, editrule,
	// reorderrules, deleterule, spoiler, unspoiler, modmail_enrollment, community_styling, community_widgets,
	// markoriginalcontent, collections, events, hidden_award, add_community_topics, remove_community_topics,
	// create_scheduled_post, edit_scheduled_post, delete_scheduled_post, submit_scheduled_post,
	// edit_post_requirements, invitesubscriber, submit_content_rating_survey.
	Type string `url:"type,omitempty"`
	// If provided, only return the actions of this moderator.
	Moderator string `url:"mod,omitempty"`
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	origURL, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	origValues := origURL.Query()

	newValues, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	for k, v := range newValues {
		origValues[k] = v
	}

	origURL.RawQuery = origValues.Encode()
	return origURL.String(), nil
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

// Int is a helper routine that allocates a new int value
// to store v and returns a pointer to it.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}
