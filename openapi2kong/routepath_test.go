package openapi2kong

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Valid route paths. Most are taken from the Kong Gateway test suite
// (spec/01-unit/01-db/01-schema/06-routes_spec.lua, spec/01-unit/08-router_spec.lua,
// and spec/02-integration/05-proxy/02-router_spec.lua), extended with commonly
// used PCRE constructs.
var validRoutePaths = map[string][]string{
	"plain paths": {
		"/",
		"/ovo/",                      // trailing slash
		"/abcd~user~2",               // RFC 3986 unreserved characters
		"/catalog/products/v1",       // plain prefix
		"/catalog/products/v1/",      // plain prefix, trailing slash
		"/v1.0/users",                // dots that are not dot-segments
		"/a.b/c..d/...",              // dots that are not dot-segments
		"/users/@me:profile",         // ':' and '@' are path characters
		"/a!$&'()*+,;=b",             // RFC 3986 sub-delims
		"/some%20words",              // NOTE: see below, %20 is not normalized
		"/encoded%2Fslash",           // percent-encoded reserved character
		"/encoded%3Fquestion%23hash", // percent-encoded reserved characters
		"/percent%25sign",            // percent-encoded '%'
		"/with space",                // non-RFC 3986 characters are accepted by Kong
		"/unicode/ä",                 // non-RFC 3986 characters are accepted by Kong
	},
	"Kong router test regexes": {
		`~/(path)`,
		`~/[0]?`,
		`~/delay/(?<delay>[^\/]+)$`,
		`~/reg%65x/\d+`,
		`~/regex-meta/%5Cd\+%2E`,
		`~/regex-reserved%2Fabc`,
		`~/route/(fixture)`,
		`~/some/thing/else`,
		`~/status/(?P<foo>re)`,
		`~/status/(ab)c?`,
		`~/(?i)webhook/foo/bar/?$`,
		`~/(users/)?1984/(?<subpath>profile)$`,
		`~/\"/*$`,
		`~/\"#/*$`,
		`~/\\/*$`,
		`~/\/*/user$`,
		`~/\d+"`,
		`~/admin$`,
		`~/enabled-service/\w+`,
		`~/route/persons/\d{3}`,
		`~/route/persons/\d{3}/following`,
		`~/route/persons/\d+/profile`,
		`~/users/(\d+)/profile`,
		`~/users/\d*/profile`,
		`~/v2/control-planes/resource-quotas(?<path>\S*)`,
	},
	"anchors and optional segments": {
		`~/catalog/products/v1/?$`,
		`~/catalog/products/v1(/.*)?$`,
		`~/api$`,
		`~/api/\z`,
		`~/api\b`,
		`~/`,
	},
	"quantifiers": {
		`~/a*/b+/c?$`,
		`~/x{2}/y{2,}/z{2,5}$`,
		`~/lazy/.*?/end$`,
		`~/lazy/.+?/end$`,
	},
	"character classes": {
		`~/items/[0-9]+$`,
		`~/items/[a-zA-Z0-9_-]+$`,
		`~/items/[^/]+$`,
		`~/items/[]a]$`,
		`~/items/[\d\w.-]+$`,
		`~/items/[[:alpha:]]+$`,
		`~/items/[[:digit:][:upper:]]+$`,
		`~/items/\p{L}+$`,
		`~/items/\P{N}+$`,
		`~/items/\s\S\d\D\w\W$`,
		`~/items/.$`,
	},
	"groups and alternation": {
		`~/(v1|v2|v3)/users$`,
		`~/api/(?:v1|v2)/users$`,
		`~/(?<version>v[0-9]+)/(?<resource>[a-z]+)/(?<id>[0-9a-f-]{36})$`,
		`~/(?P<version>v[0-9]+)/users$`,
		`~/(?i:users)/profile$`,
		`~/(?i)users/(?-i)profile$`,
		`~/(?s).*$`,
		`~/((a|b)(c|d))+$`,
		`~/a|/b`,
	},
	"escaping": {
		`~/files/.+\.json$`,
		`~/a\+b/c\*d/e\?f/g\(h\)/i\[j\]/k\{l\}/m\|n/o\$p/q\^r$`,
		`~/\Q1+1=2\E$`,
		`~/tab\tnewline\n$`,
		`~/hex\x41/unicode\x{263a}$`,
	},
	"PCRE only (not supported by Go RE2, left for Kong to validate)": {
		`~/users/(?=admin)\w+$`,             // positive look-ahead
		`~/users/(?!admin)\w+$`,             // negative look-ahead
		`~/users/\w+(?<=min)$`,              // positive look-behind
		`~/users/\w+(?<!admin)$`,            // negative look-behind
		`~/(a)/\1$`,                         // back-reference
		`~/(?<x>a)/\k<x>$`,                  // named back-reference
		`~/(?>atomic)/group$`,               // atomic group
		`~/possessive/a++b*+c?+$`,           // possessive quantifiers
		`~/keep/\K/out$`,                    // match reset
		`~/(?'name'value)$`,                 // named group, quoted syntax
		`~/(?#comment)users$`,               // inline comment
		`~/(a)?(?(1)b|c)$`,                  // conditional
		`~/nested/(\((?:[^()]|(?R))*\))$`,   // recursion
		`~/skip/(?:bad(*SKIP)(*FAIL)|\w+)$`, // backtracking verbs
		`~/h/\h\v$`,                         // horizontal/vertical white space
		`~/end\Z`,                           // end of subject or before final newline
	},
}

func Test_validateRoutePath_Valid(t *testing.T) {
	for group, paths := range validRoutePaths {
		for _, path := range paths {
			t.Run(group+": "+path, func(t *testing.T) {
				if path == "/some%20words" {
					// Kong considers '%20' non-normalized (decodes it to ' '), verify we do the same
					assert.ErrorContains(t, validateRoutePath(path), "consider using '/some words' instead")
					return
				}
				assert.NoError(t, validateRoutePath(path))
			})
		}
	}
}

func Test_validateRoutePath_Invalid(t *testing.T) {
	tests := []struct {
		path        string
		expectedErr string
	}{
		// prefix
		{"", "must start with '/' (plain path) or '~/' (regex path)"},
		{"foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"foo/bar", "must start with '/' (plain path) or '~/' (regex path)"},
		{"~", "must start with '/' (plain path) or '~/' (regex path)"},
		{"~foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"~^/foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"~(/foo)", "must start with '/' (plain path) or '~/' (regex path)"},
		{" /foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"^/foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"*/foo", "must start with '/' (plain path) or '~/' (regex path)"},
		{"http://example.com/foo", "must start with '/' (plain path) or '~/' (regex path)"},

		// empty segments (Kong: "must not have empty segments")
		{"//foo/bar", "must not have empty segments"},
		{"/foo//bar", "must not have empty segments"},
		{"/foo/bar//", "must not have empty segments"},
		{"~//foo", "must not have empty segments"},
		{"~/foo//bar$", "must not have empty segments"},

		// invalid regexes (Kong: "invalid regex: ...")
		{`~/users/(foo/profile`, "invalid regex '/users/(foo/profile': missing closing )"},
		{`~/[abc/*/user$`, "invalid regex '/[abc/*/user$': missing closing ]"},
		{`~/users/foo)/profile`, "unexpected )"},
		{`~/users/(?:foo/profile`, "missing closing )"},
		{`~/users/((foo)/profile`, "missing closing )"},
		{`~/items/[z-a]$`, "invalid character class range"},
		{`~/items/[9-0]$`, "invalid character class range"},
		{`~/items/[[:nope:]]$`, "invalid character class range"},
		{`~/trailing\`, "trailing backslash at end of expression"},
		{`~/a/(*b)$`, ""}, // PCRE verb syntax, accepted (see below)
		{`~/a/(+b)$`, "missing argument to repetition operator"},
		{`~/a/|*b$`, "missing argument to repetition operator"},

		// percent-encoding (Kong: "invalid url-encoded value: ...")
		{"/some%2words", "invalid url-encoded value: '%2w'"},
		{"/some%0Xwords", "invalid url-encoded value: '%0X'"},
		{"/some%2Gwords", "invalid url-encoded value: '%2G'"},
		{"/some%20words%", "invalid url-encoded value: '%'"},
		{"/some%20words%a", "invalid url-encoded value: '%a'"},
		{"/some%20words%ax", "invalid url-encoded value: '%ax'"},

		// not normalized (Kong: "non-normalized path, consider use '...' instead")
		{"/%c3%A4", "consider using '/ä' instead"},
		{"/%41bc", "consider using '/Abc' instead"},
		{"/a%2db", "consider using '/a-b' instead"},
		{"/a%7Eb", "consider using '/a~b' instead"},
		{"/a%2fb", "consider using '/a%2Fb' instead"},
		{"/a%3fb", "consider using '/a%3Fb' instead"},
		{"/a/./b", "consider using '/a/b' instead"},
		{"/a/../b", "consider using '/b' instead"},
		{"/a/b/..", "consider using '/a/' instead"},
		{"/a/b/.", "consider using '/a/b/' instead"},
		{"/./a", "consider using '/a' instead"},
		{"/../a", "consider using '/a' instead"},
		{"/a/%2e%2e/b", "consider using '/b' instead"},
		{"/a/%2E/b", "consider using '/a/b' instead"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			err := validateRoutePath(tc.path)
			if tc.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func Test_normalizeRoutePath(t *testing.T) {
	tests := map[string]string{
		"":                "",
		"/":               "/",
		"/a/b/c":          "/a/b/c",
		"/a/b/c/":         "/a/b/c/",
		"/a/./b":          "/a/b",
		"/a/../b":         "/b",
		"/a/b/../../c":    "/c",
		"/a/b/../../../c": "/c",
		"/..":             "/",
		"/.":              "/",
		"/a/b/.":          "/a/b/",
		"/a/b/..":         "/a/",
		"/a.b/c..d/...":   "/a.b/c..d/...",
		"/.hidden/x":      "/.hidden/x",
		"/%41%42%43":      "/ABC",
		"/%61%62%63":      "/abc",
		"/%2f%3a%40":      "/%2F%3A%40",
		"/%2F%3A%40":      "/%2F%3A%40",
		"/%25":            "/%25",
		"/%20":            "/ ",
		"/%c3%a4":         "/ä",
		"/%2e%2e/a":       "/a",
		"/%":              "/%",
		"/%zz":            "/%zz",
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, expected, normalizeRoutePath(input))
		})
	}
}

// Test_RoutePath_Matching documents the matching behaviour of some common
// x-kong-route-path regexes. Kong matches a regex path anchored at the start of
// the request path (but not at the end), which is emulated here with Go's regexp
// for the regexes that are compatible with both PCRE and RE2.
func Test_RoutePath_Matching(t *testing.T) {
	kongMatch := func(t *testing.T, routePath string) func(string) bool {
		t.Helper()
		assert.NoError(t, validateRoutePath(routePath))
		re := regexp.MustCompile(`^(?:` + routePath[1:] + `)`)
		return re.MatchString
	}

	tests := []struct {
		routePath  string
		matches    []string
		nonMatches []string
	}{
		{
			// APIO-I-70: optional trailing slash
			routePath:  `~/catalog/products/v1/?$`,
			matches:    []string{"/catalog/products/v1", "/catalog/products/v1/"},
			nonMatches: []string{"/catalog/products/v1x", "/catalog/products/v1/x", "/catalog/products/v2"},
		},
		{
			// what openapi2kong generates by default: trailing slash does NOT match
			routePath:  `~/catalog/products/v1$`,
			matches:    []string{"/catalog/products/v1"},
			nonMatches: []string{"/catalog/products/v1/"},
		},
		{
			// path and everything below it
			routePath:  `~/catalog/products/v1(/.*)?$`,
			matches:    []string{"/catalog/products/v1", "/catalog/products/v1/", "/catalog/products/v1/a/b"},
			nonMatches: []string{"/catalog/products/v10"},
		},
		{
			// no '$' anchor, so it is a regex prefix match
			routePath:  `~/users/\d+`,
			matches:    []string{"/users/1", "/users/123/profile", "/users/1abc"},
			nonMatches: []string{"/users/", "/users/abc", "/api/users/1"},
		},
		{
			routePath:  `~/items/(?<itemId>[0-9]+)/?$`,
			matches:    []string{"/items/1", "/items/42/"},
			nonMatches: []string{"/items/", "/items/abc", "/items/1/2"},
		},
		{
			routePath:  `~/(v1|v2)/users$`,
			matches:    []string{"/v1/users", "/v2/users"},
			nonMatches: []string{"/v3/users", "/v1/users/"},
		},
		{
			routePath:  `~/(?i)webhook/foo/bar/?$`,
			matches:    []string{"/webhook/foo/bar", "/WebHook/FOO/bar/"},
			nonMatches: []string{"/webhook/foo"},
		},
		{
			routePath:  `~/files/[^/]+\.json$`,
			matches:    []string{"/files/a.json", "/files/my-file.v2.json"},
			nonMatches: []string{"/files/a.yaml", "/files/ajson", "/files/a/b.json"},
		},
		{
			routePath:  `~/(?<version>v[0-9]+)/(?<id>[0-9a-f-]{36})$`,
			matches:    []string{"/v1/123e4567-e89b-12d3-a456-426614174000"},
			nonMatches: []string{"/v1/123", "/latest/123e4567-e89b-12d3-a456-426614174000"},
		},
		{
			routePath:  `~/api/(?:v1|v2)/users(/[0-9]+)?$`,
			matches:    []string{"/api/v1/users", "/api/v2/users/7"},
			nonMatches: []string{"/api/v1/users/x", "/api/v1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.routePath, func(t *testing.T) {
			match := kongMatch(t, tc.routePath)
			for _, uri := range tc.matches {
				assert.True(t, match(uri), "'%s' should match '%s'", tc.routePath, uri)
			}
			for _, uri := range tc.nonMatches {
				assert.False(t, match(uri), "'%s' should not match '%s'", tc.routePath, uri)
			}
		})
	}
}
