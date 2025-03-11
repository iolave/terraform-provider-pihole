package pihole

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/iolave/go-proxmox/pkg/cloudflare"
	pihole "github.com/ryanwholey/go-pihole"
)

type Config struct {
	Password       string
	URL            string
	UserAgent      string
	Client         *http.Client
	APIToken       string
	CFServiceToken *cloudflare.ServiceToken
}

type Client struct {
	URL            string
	UserAgent      string
	password       string
	sessionID      string
	sessionToken   string
	webPassword    string
	client         *http.Client
	tokenClient    *pihole.Client
	cfServiceToken *cloudflare.ServiceToken
}

// doubleHash256 takes a string, double hashes it using the sha256 algorithm and returns the value
func doubleHash256(data string) string {
	hash := sha256.Sum256([]byte(data))
	sha1 := fmt.Sprintf("%x", hash[:])

	hash2 := sha256.Sum256([]byte(sha1))
	return fmt.Sprintf("%x", hash2[:])
}

// New returns a new Pi-hole client
func New(config Config) *Client {
	client := &Client{
		URL:            config.URL,
		UserAgent:      config.UserAgent,
		password:       config.Password,
		client:         config.Client,
		webPassword:    doubleHash256(config.Password),
		cfServiceToken: config.CFServiceToken,
	}

	if client.client == nil {
		client.client = &http.Client{}
	}

	if config.APIToken != "" {
		client.tokenClient = pihole.New(pihole.Config{
			BaseURL:    config.URL,
			APIToken:   config.APIToken,
			HttpClient: client.client,
		})
	}

	return client
}

// Init sets fields on the client which are a product of pihole network requests or other side effects
func (c *Client) Init(ctx context.Context) error {
	if c.URL == "" {
		return fmt.Errorf("%w: Pi-hole URL is not set", ErrClientValidationFailed)
	}

	if c.tokenClient != nil {
		return nil
	}

	if c.password == "" {
		return fmt.Errorf("%w: password is not set", ErrClientValidationFailed)
	}

	if c.webPassword == "" {
		return fmt.Errorf("%w: webPassword is not set", ErrClientValidationFailed)
	}

	return nil
}

// Login creates a session and sets the proper attributes on the client for session based requests (not api token reqeuests)
func (c *Client) Login(ctx context.Context) error {
	if err := c.login(ctx); err != nil {
		return fmt.Errorf("%w: %s", ErrLoginFailed, err)
	}

	if c.sessionToken == "" {
		return fmt.Errorf("%w: token not set", ErrClientValidationFailed)
	}

	if c.sessionID == "" {
		return fmt.Errorf("%w: sessionID not set", ErrClientValidationFailed)
	}

	return nil
}

// Request executes a basic unauthenticated http request
func (c *Client) Request(ctx context.Context, method string, path string, data *url.Values) (*http.Request, error) {
	d := data
	if d == nil {
		d = &url.Values{}
	}

	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", c.URL, path), strings.NewReader(d.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if c.cfServiceToken == nil {
		return req, nil
	}

	if err := c.cfServiceToken.Set(req); err != nil {
		return nil, err
	}

	return req, nil
}

// mergeURLValues merges the passed URL values into a single url.Values object
func mergeURLValues(vs ...url.Values) url.Values {
	data := url.Values{}

	for _, val := range vs {
		for k, v := range val {
			data.Add(k, v[0])
		}
	}

	return data
}

// RequestWithSession executes a request with appropriate session authentication
func (c Client) RequestWithSession(ctx context.Context, method string, path string, data *url.Values) (*http.Request, error) {
	if c.sessionToken == "" || c.sessionID == "" {
		if err := c.Login(ctx); err != nil {
			return nil, err
		}
	}

	d := mergeURLValues(url.Values{
		"token": []string{c.sessionToken},
	}, *data)
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", c.URL, path), strings.NewReader(d.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cookie", fmt.Sprintf("PHPSESSID=%s", c.sessionID))

	if c.cfServiceToken == nil {
		return req, nil
	}

	if err := c.cfServiceToken.Set(req); err != nil {
		return nil, err
	}

	return req, nil
}

// RequestWithAuth adds an auth token to the passed request
func (c Client) RequestWithAuth(ctx context.Context, method string, path string, data *url.Values) (*http.Request, error) {
	u, err := url.Parse(fmt.Sprintf("%s%s", c.URL, path))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Add("auth", c.webPassword)
	u.RawQuery = q.Encode()

	d := data
	if d == nil {
		d = &url.Values{}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), strings.NewReader(d.Encode()))
	if err != nil {
		return nil, err
	}

	if c.cfServiceToken == nil {
		return req, nil
	}

	if err := c.cfServiceToken.Set(req); err != nil {
		return nil, err
	}

	return req, nil
}

// login sets a new sessionID and csrf token in the client to be used for logged in requests
func (c *Client) login(ctx context.Context) error {
	data := map[string]any{
		"password": c.password,
	}
	b, _ := json.Marshal(data)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s%s", c.URL, "/api/auth"), bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed to format login request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %s", err)
	}

	defer res.Body.Close()
	b, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read req body: %s", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to login, got status code: %d", res.StatusCode)
	}

	type Response struct {
		Session struct {
			Valid    bool   `json:"valid"`
			TOTP     bool   `json:"totp"`
			SID      string `json:"sid"`
			CSRF     string `json:"csrf"`
			Validity int    `json:"validity"`
			Message  string `json:"message"`
		} `json:"session"`
	}

	var responseResult Response
	if err := json.Unmarshal(b, &responseResult); err != nil {
		return fmt.Errorf("unable to parse login response: %s", err)
	}

	c.sessionID = responseResult.Session.SID
	c.sessionToken = responseResult.Session.CSRF
	return nil
}

// Bool is a helper to return pointer booleans
func Bool(b bool) *bool {
	return &b
}
