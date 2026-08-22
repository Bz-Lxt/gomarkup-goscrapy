package fetcher

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"goscrapy/internal/proxy"
	"goscrapy/internal/urlx"
)

const DefaultUA = "GoScrapy/1.0 (+https://goscrapy.local)"

type Result struct {
	URL        string
	Status     int
	Body       []byte
	FinalURL   string
	Duration   time.Duration
	ProxyURL   string
	Blocked    bool
	RobotsSkip bool
}

type Client struct {
	ua      string
	timeout time.Duration
	pool    *proxy.Pool
	robots  *RobotsCache
	jars    sync.Map // host -> *cookiejar.Jar
	base    *http.Transport
}

func New(pool *proxy.Pool, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c := &Client{ua: DefaultUA, timeout: timeout, pool: pool, base: tr}
	c.robots = NewRobotsCache(&http.Client{Timeout: 8 * time.Second, Transport: tr}, DefaultUA)
	return c
}

func (c *Client) Fetch(ctx context.Context, rawURL string, respectRobots bool) (*Result, error) {
	rawURL = urlx.Canonical(rawURL)
	if !urlx.IsHTTP(rawURL) {
		return nil, fmt.Errorf("unsupported url %s", rawURL)
	}
	if respectRobots {
		ok, delay, err := c.robots.Allowed(ctx, rawURL)
		if err == nil && !ok {
			return &Result{URL: rawURL, RobotsSkip: true, Status: 0}, nil
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	px := ""
	if c.pool != nil {
		if p := c.pool.Dial(); p != "" {
			px = p
		}
	}
	client, err := c.httpClient(rawURL, px)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.6")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if c.pool != nil && px != "" {
			c.pool.Report(px, false, err.Error())
		}
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if c.pool != nil && px != "" {
		c.pool.Report(px, resp.StatusCode < 500 && resp.StatusCode != 407, "")
	}
	final := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	blocked := resp.StatusCode == 429 || resp.StatusCode == 403
	return &Result{
		URL:      rawURL,
		Status:   resp.StatusCode,
		Body:     body,
		FinalURL: final,
		Duration: time.Since(start),
		ProxyURL: px,
		Blocked:  blocked,
	}, nil
}

func (c *Client) httpClient(pageURL, proxyURL string) (*http.Client, error) {
	tr := c.base.Clone()
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		tr.Proxy = http.ProxyURL(u)
	}
	jar := c.jarFor(urlx.Host(pageURL))
	return &http.Client{Timeout: c.timeout, Transport: tr, Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 8 {
			return fmt.Errorf("too many redirects")
		}
		*req = *req.WithContext(context.WithoutCancel(req.Context()))
		req.Header.Set("User-Agent", c.ua)
		return nil
	}}, nil
}

func (c *Client) jarFor(host string) http.CookieJar {
	if host == "" {
		host = "_"
	}
	if v, ok := c.jars.Load(host); ok {
		return v.(http.CookieJar)
	}
	j, _ := cookiejar.New(nil)
	actual, _ := c.jars.LoadOrStore(host, j)
	return actual.(http.CookieJar)
}

func IsHTML(body []byte, contentType string) bool {
	if strings.Contains(strings.ToLower(contentType), "html") {
		return true
	}
	s := strings.ToLower(string(body[:min(len(body), 256)]))
	return strings.Contains(s, "<html") || strings.Contains(s, "<!doctype")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
