package openapi2kong

import (
	"net/url"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp"
)

func Test_parseServerUris(t *testing.T) {
	// basics

	servers := &openapi3.Servers{
		{
			URL: "http://cookiemonster.com/chocolate/cookie",
		}, {
			URL: "https://konghq.com/bitter/sweet",
		},
	}
	expected := []*url.URL{
		{
			Scheme: "http",
			Host:   "cookiemonster.com",
			Path:   "/chocolate/cookie",
		}, {
			Scheme: httpsScheme,
			Host:   "konghq.com",
			Path:   "/bitter/sweet",
		},
	}
	targets, err := parseServerUris(servers)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	if diff := cmp.Diff(targets, expected); diff != "" {
		t.Errorf(diff)
	}

	// replaces variables with defaults

	servers = &openapi3.Servers{
		{
			URL: "http://{var1}-{var2}.com/chocolate/cookie",
			Variables: map[string]*openapi3.ServerVariable{
				"var1": {
					Default: "hello",
					Enum:    []string{"hello", "world"},
				},
				"var2": {
					Default: "Welt",
					Enum:    []string{"hallo", "Welt"},
				},
			},
		},
	}
	expected = []*url.URL{
		{
			Scheme: "http",
			Host:   "hello-Welt.com",
			Path:   "/chocolate/cookie",
		},
	}
	targets, err = parseServerUris(servers)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	if diff := cmp.Diff(targets, expected); diff != "" {
		t.Errorf(diff)
	}

	// returns error on a bad URL

	servers = &openapi3.Servers{
		{
			URL: "http://cookiemonster.com/chocolate/cookie",
		}, {
			URL: "not really a url...",
		},
	}
	_, err = parseServerUris(servers)
	if err == nil {
		t.Error("expected an error")
	}

	// returns no error if servers is empty

	expected = []*url.URL{
		{
			Path: "/",
		},
	}
	targets, err = parseServerUris(&openapi3.Servers{})
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	if diff := cmp.Diff(targets, expected); diff != "" {
		t.Errorf(diff)
	}

	// returns no error if servers is nil

	expected = []*url.URL{
		{
			Path: "/",
		},
	}
	targets, err = parseServerUris(nil)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	if diff := cmp.Diff(targets, expected); diff != "" {
		t.Errorf(diff)
	}
}

func Test_setServerDefaults(t *testing.T) {
	defaultTests := []struct {
		name      string
		inURL     string
		outPort   string
		outScheme string
	}{
		{"adds default scheme", "//host/path", "443", "https"},
		{"adds port 80 for http", "http://host/path", "80", "http"},
		{"adds port 443 for https", "https://host/path", "443", "https"},
	}

	for _, tst := range defaultTests {
		inURL, _ := url.Parse(tst.inURL)
		urls := []*url.URL{inURL}
		setServerDefaults(urls, "https")
		if urls[0].Port() != tst.outPort {
			t.Errorf("%s: expected port to be '%s', but got '%s'", tst.name, tst.outPort, urls[0].Port())
		}
		if urls[0].Scheme != tst.outScheme {
			t.Errorf("%s: expected scheme to be '%s', but got '%s'", tst.name, tst.outScheme, urls[0].Scheme)
		}
	}
}
