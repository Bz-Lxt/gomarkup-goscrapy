package fetcher

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"goscrapy/internal/urlx"
)

type robotsRule struct {
	allow    []string
	disallow []string
	delay    time.Duration
	fetched  time.Time
	ok       bool
}

type RobotsCache struct {
	mu      sync.Mutex
	entries map[string]*robotsRule
	ttl     time.Duration
	client  *http.Client
	ua      string
}

func NewRobotsCache(client *http.Client, ua string) *RobotsCache {
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	if ua == "" {
		ua = DefaultUA
	}
	return &RobotsCache{entries: map[string]*robotsRule{}, ttl: 10 * time.Minute, client: client, ua: ua}
}

func (c *RobotsCache) Allowed(ctx context.Context, rawURL string) (bool, time.Duration, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false, 0, err
	}
	host := strings.ToLower(u.Hostname())
	c.mu.Lock()
	ent := c.entries[host]
	if ent != nil && time.Since(ent.fetched) < c.ttl {
		c.mu.Unlock()
		return ent.allows(u.Path), ent.delay, nil
	}
	c.mu.Unlock()

	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodGet, robotsURL, nil)
	if err != nil {
		return true, 0, nil
	}
	req.Header.Set("User-Agent", c.ua)
	resp, err := c.client.Do(req)
	rule := &robotsRule{fetched: time.Now(), ok: true, allow: []string{"/"}}
	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			rule = parseRobots(string(body))
			rule.fetched = time.Now()
			rule.ok = true
		}
	}
	c.mu.Lock()
	c.entries[host] = rule
	c.mu.Unlock()
	return rule.allows(u.Path), rule.delay, nil
}

func (r *robotsRule) allows(path string) bool {
	if path == "" {
		path = "/"
	}
	matchedDisallow := ""
	matchedAllow := ""
	for _, d := range r.disallow {
		if matchPrefix(path, d) && len(d) >= len(matchedDisallow) {
			matchedDisallow = d
		}
	}
	for _, a := range r.allow {
		if matchPrefix(path, a) && len(a) >= len(matchedAllow) {
			matchedAllow = a
		}
	}
	if matchedDisallow == "" {
		return true
	}
	return len(matchedAllow) > len(matchedDisallow)
}

func matchPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	if prefix == "/" {
		return true
	}
	return strings.HasPrefix(path, prefix)
}

func parseRobots(body string) *robotsRule {
	r := &robotsRule{}
	sc := bufio.NewScanner(strings.NewReader(body))
	inStar := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		switch key {
		case "user-agent":
			inStar = val == "*" || strings.EqualFold(val, "goscrapy")
		case "disallow":
			if inStar {
				r.disallow = append(r.disallow, val)
			}
		case "allow":
			if inStar {
				r.allow = append(r.allow, val)
			}
		case "crawl-delay":
			if inStar {
				if d, err := time.ParseDuration(val + "s"); err == nil {
					r.delay = d
				}
			}
		}
	}
	return r
}

func RobotsURL(page string) string {
	host := urlx.Host(page)
	if host == "" {
		return ""
	}
	u, err := url.Parse(page)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/robots.txt"
}
