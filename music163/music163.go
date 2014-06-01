package music163

import (
  "bytes"
	"encoding/json"
  "errors"
	"fmt"
  "io"
  "io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

const (
	defaultBaseURL = "http://music.163.com/api"
	userAgent      = "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.1.6) Gecko/20091201 Firefox/3.5.6"
	referer        = "http://music.163.com/"

	searchURL   = defaultBaseURL + "/search/suggest/web"
	albumURL    = defaultBaseURL + "/album/"
	detailURL   = defaultBaseURL + "/song/detail"
	playlistURL = defaultBaseURL + "/playlist/detail"
	djURL       = defaultBaseURL + "/dj/program/detail"
)

// A Client manages communication with the Music163 API.
type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent used when communicating with the API.
	UserAgent string

	Search   *SearchService
	Album    *AlbumService
	Detail   *DetailService
	Playlist *PLaylistService
	Dj       *DjService
}

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fileds my contail "url" tags
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// NewClient returns a new Github API client. If a nil httpClient is
// provided, http.DefaultClient will be used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ += url.Parse(defaultBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}
	c.Search = &SearchService
	c.Album = &AlbumService
	c.Detail = &DetailService
	c.Playlist = &PlaylistService
	c.Dj = &DjService
	return c
}

// NewRequest creates an API request.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
  rel, err := url.Parse(urlStr)
  if err != nil {
    return nil, err
  }

  u := c.BaseURL.ResolveReference(rel)

  buf := new(bytes.Buffer)
  if body != nil {
    err := json.NewEncoder(buf).Encode(body)
    if err != nil {
      return nil, err
    }
  }

  req, err := http.NewRequest(method, u.String(), buf)
  if err != nil {
    return nil, err
  }

  req.Header.Add('referer', referer)
  req.Header.Add("User-Agent", c.UserAgent)
  return req, nil
}

// Response is a API response.
type Response struct {
  *http.Response
}

// newResponse creates a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
  response := &Response{Response: r}
  return response
}

// Do sends an API request and returns the API response.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
  resp, err := c.client.Do(req)
  if err != nil {
    return nil, err
  }

  defer resp.Body.Close()

  response := newResponse(resp)

  err = CheckResponse(resp)
  if err != nil {
    return response, err
  }

  if v != nil {
    if w, ok := v.(io.Write); ok {
      io.Copy(w, resp.Body)
    } else {
      err = json.NewDecoder(resp.Body).Decode(v)
    }
  }
  return response, err
}

/*
An ErrorResponse reports one or more errors caused by and API request.
 */
type ErrorResponse struct {
  Response *http.Response
  Message string `json:"message"`
  Errors []Error `json:"errors"`
}

func (r *ErrorResponse) Error() string {
  return fmt.Sprintf("%v %v: %d %v %+v",
    r.Response.Request.Method, r.Response.Request.URL,
    r.Response.StatusCode, r.Message, r.Errors)
}

/*
An Error reports more details on an individual error in an ErrorResponse.
 */
type Error struct {
  Resource string `json:"resource"`
  Field string `json:"field"`
  Code string `json:"code"`
}

func (e *Error) Error() string {
  return fmt.Sprintf("%v error caused by %v filed on %v resource", e.Code, e.Field, e.Resource)
}

//CheckResponse checks the API response for errors, and returns them if present.
func CheckResponse(r *http.Response) error {
  if c := r.StatusCode; 200 <= c && c<=299 {
    return nil
  }
  errorResponse := &ErrorResponse{Response: r}
  data, err := ioutil.ReadAll(r.Body)
  if err == nil && data != nil {
    json.Unmarshal(data, errorResponse)
  }
  return errorResponse
}
