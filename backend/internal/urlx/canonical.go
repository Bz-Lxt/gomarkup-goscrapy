package urlx

import (
	"hash/fnv"
	"net/url"
	"path"
	"sort"
	"strings"
)

func Canonical(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	} else {
		cleaned := path.Clean(u.Path)
		if strings.HasSuffix(u.Path, "/") && !strings.HasSuffix(cleaned, "/") && cleaned != "/" {
			cleaned += "/"
		}
		u.Path = cleaned
	}
	if u.RawQuery != "" {
		q := u.Query()
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		vals := url.Values{}
		for _, k := range keys {
			vs := q[k]
			sort.Strings(vs)
			for _, v := range vs {
				vals.Add(k, v)
			}
		}
		u.RawQuery = vals.Encode()
	}
	return u.String()
}

func Resolve(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
		return ""
	}
	if strings.HasPrefix(href, "#") {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if base == "" {
		return Canonical(href)
	}
	bu, err := url.Parse(base)
	if err != nil {
		return Canonical(href)
	}
	return Canonical(bu.ResolveReference(ref).String())
}

func Host(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func DomainKey(raw string) string {
	h := Host(raw)
	if h == "" {
		return "unknown"
	}
	return h
}

// AffinityShard maps a host onto a worker bucket. Same host always lands
// in the same shard so per-domain rate limits and cookies stay local.
func AffinityShard(host string, workers int) int {
	if workers <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(host)))
	return int(h.Sum32() % uint32(workers))
}

func SameHost(a, b string) bool {
	return Host(a) != "" && Host(a) == Host(b)
}

func IsHTTP(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	s := strings.ToLower(u.Scheme)
	return s == "http" || s == "https"
}
