package openapi2kong

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

func Test_parseServerUris(t *testing.T) {
	// basics

	servers := []*v3.Server{
		{
			URL: "http://cookiemonster.com/chocolate/cookie",
		},
		{
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
		t.Errorf(diff) //nolint:staticcheck
	}

	variables := orderedmap.New[string, *v3.ServerVariable]()
	variables.Set("var1", &v3.ServerVariable{
		Default: "hello",
		Enum:    []string{"hello", "world"},
	})
	variables.Set("var2", &v3.ServerVariable{
		Default: "Welt",
		Enum:    []string{"hallo", "Welt"},
	})

	// replaces variables with defaults
	servers = []*v3.Server{
		{
			URL:       "http://{var1}-{var2}.com/chocolate/cookie",
			Variables: variables,
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
		t.Errorf(diff) //nolint:staticcheck
	}

	// returns error on a bad URL

	servers = []*v3.Server{
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
	targets, err = parseServerUris([]*v3.Server{})
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	if diff := cmp.Diff(targets, expected); diff != "" {
		t.Errorf(diff) //nolint:staticcheck
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
		t.Errorf(diff) //nolint:staticcheck
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
