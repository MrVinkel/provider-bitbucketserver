package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiPath       = "/rest/api/1.0/"
	jsonMediaType = "application/json"
)

// Client encapsulates a client that talks to the bitbucket server api
// API Docs: https://developer.atlassian.com/server/bitbucket/rest/v805/intro/
type Client struct {
	// client represents the HTTP client used for making HTTP requests.
	client *http.Client

	// headers are used to override request headers for every single HTTP request
	headers map[string]string

	// base URL for the bitbucket server + apiPath
	baseURL *url.URL
}

var (
	// ErrPermission represents permission related errors
	ErrPermission = errors.New("permission")
	// ErrNotFound represents errors where the resource being fetched was not found
	ErrNotFound = errors.New("not_found")
	// ErrResponseMalformed represents errors related to api responses that do not match internal representation
	ErrResponseMalformed = errors.New("response_malformed")
	// ErrConflict is used when a duplicate resource is trying to be created
	ErrConflict = errors.New("conflict")
)

// NewClient creates a new instance of the bitbucket client
func NewClient(baseURL string, base64creds string) (*Client, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	pBaseURL, err := url.Parse(fmt.Sprintf("%s%s", baseURL, apiPath))
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL: pBaseURL,
		client:  &http.Client{Timeout: time.Second * 10},
		headers: map[string]string{"Authorization": fmt.Sprintf("Bearer %s", base64creds)},
	}

	err = c.ping()
	if err != nil {
		return nil, fmt.Errorf("error creating bitbucket client: %w", err)
	}

	return c, nil
}

// ping is used to check that the client can correctly communicate with the bitbucket api
func (c *Client) ping() error {
	req, err := c.newRequest("GET", "projects", nil)
	if err != nil {
		return fmt.Errorf("error creating request for getting projects: %w", err)
	}

	err = c.do(context.Background(), req, nil)
	if err != nil {
		return fmt.Errorf("error fetching projects at %s: %w", req.URL.String(), err)
	}
	return nil
}

func (c *Client) newRequest(method string, path string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	switch method {
	case http.MethodGet:
		req, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	default:
		buf := new(bytes.Buffer)
		if body != nil {
			err = json.NewEncoder(buf).Encode(body)
			if err != nil {
				return nil, err
			}
		}

		req, err = http.NewRequest(method, u.String(), buf)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", jsonMediaType)
	}

	req.Header.Set("Accept", jsonMediaType)

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// do makes an HTTP request and populates the given struct v from the response.
func (c *Client) do(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return c.handleResponse(res, v)
}

// handleResponse makes an HTTP request and populates the given struct v from
// the response.  This is meant for internal testing and shouldn't be used
// directly. Instead please use `Client.do`.
func (c *Client) handleResponse(res *http.Response, v interface{}) error {
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case 404:
		return ErrNotFound
	case 401:
		return ErrPermission
	case 409:
		return ErrConflict
	}

	// this means we don't care about unmarshaling the response body into v
	if v == nil || res.StatusCode == http.StatusNoContent {
		return nil
	}

	err = json.Unmarshal(out, &v)
	if err != nil {
		var jsonErr *json.SyntaxError
		if errors.As(err, &jsonErr) {
			return ErrResponseMalformed
		}
		return err
	}

	return nil
}
