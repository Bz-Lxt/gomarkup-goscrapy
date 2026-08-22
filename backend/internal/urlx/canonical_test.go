package urlx

import "testing"

func TestCanonical(t *testing.T) {
	got := Canonical("HTTP://Mock-Target/list.html#top")
	if got != "http://mock-target/list.html" {
		t.Fatalf("got %s", got)
	}
	if Host("http://A.example/x") != "a.example" {
		t.Fatal("host")
	}
	abs := Resolve("http://mock-target/list.html", "/products/p-1.html")
	if abs != "http://mock-target/products/p-1.html" {
		t.Fatalf("resolve %s", abs)
	}
	if Resolve("http://x/", "javascript:alert(1)") != "" {
		t.Fatal("js href")
	}
	if AffinityShard("a.com", 2) == AffinityShard("a.com", 2) && AffinityShard("a.com", 1) != 0 {
		t.Fatal("affinity")
	}
	if !IsHTTP("https://x") || IsHTTP("ftp://x") {
		t.Fatal("scheme")
	}
}

func TestQuerySort(t *testing.T) {
	a := Canonical("http://x/q?b=2&a=1")
	b := Canonical("http://x/q?a=1&b=2")
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
