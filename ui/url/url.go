//go:build js && wasm

// Package url replaces the standard net/url package for basic url operations
package url

import (
	"errors"
	"strings"
	"syscall/js"
)

type (
	// URL is the address of the form
	URL struct {
		// Scheme is the type of request.
		Scheme string
		// Authority is the host and port of the url without a leading //.
		Authority string
		// Path is the location on the host.
		Path string
		// RawQuery contains additional options
		RawQuery string
	}

	// Values is the query params contain options for the url.
	Values map[string]string
)

// Parse creates a URL out of the text.
func Parse(text string) (*URL, error) {
	schemeStartIndex := 0
	schemeEndIndex := strings.Index(text, "://")
	if schemeEndIndex <= 0 {
		return nil, errors.New("url has no scheme or authority: " + text)
	}
	scheme := text[schemeStartIndex:schemeEndIndex]
	authorityStartIndex := schemeEndIndex + 3                                                     // ignore ://
	authorityEndIndex := authorityStartIndex + strings.IndexAny(text[authorityStartIndex:], "/?") // ends at path/query start
	if authorityStartIndex > authorityEndIndex {                                                  // no path/query
		authorityEndIndex = len(text)
	}
	authority := text[authorityStartIndex:authorityEndIndex]
	pathStartIndex := authorityEndIndex
	pathEndIndex := pathStartIndex + strings.Index(text[pathStartIndex:], "?")
	if pathStartIndex > pathEndIndex { // no path
		pathEndIndex = len(text)
	}
	path := text[pathStartIndex:pathEndIndex]
	queryStartIndex := pathEndIndex + 1 // ignore ?
	if queryStartIndex > len(text) {    // no query
		queryStartIndex = len(text)
	}
	rawQuery := text[queryStartIndex:]
	fragmentIndex := strings.Index(rawQuery, "#")
	if fragmentIndex >= 0 {
		return nil, errors.New("url fragment not allowed: " + text)
	}
	u := URL{
		Scheme:    scheme,
		Authority: authority,
		Path:      path,
		RawQuery:  rawQuery,
	}
	return &u, nil
}

// String concatenates the scheme, authority, path, and raw query of the url
func (u URL) String() string {
	s := u.Scheme + "://" + u.Authority + u.Path
	if len(u.RawQuery) > 0 {
		s += "?" + u.RawQuery
	}
	return s
}

// Get gets the value associated with the key, or an empty string if the value is not present.
func (v Values) Get(key string) string {
	return v[key]
}

// Add stores the value for the key.
func (v Values) Add(key, value string) {
	v[key] = value
}

// Encode concatenates the values together
func (v Values) Encode() string {
	queries := make([]string, 0, len(v))
	for key, value := range v {
		value = encodeURIComponent(value)
		queries = append(queries, key+"="+value)
	}
	return strings.Join(queries, "&")
}

// encodeURIComponent escapes special characters for safe use in URIs.
func encodeURIComponent(str string) string {
	global := js.Global()
	fn := global.Get("encodeURIComponent")
	encodedURIValue := fn.Invoke(str)
	return encodedURIValue.String()
}
