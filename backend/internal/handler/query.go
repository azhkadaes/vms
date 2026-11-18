package handler

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// maxQueryPairs bounds parsing work; real screens send far fewer pairs.
const maxQueryPairs = 100

// QueryValues is an allow-listed, typed view of the query string. Repeated keys keep
// the order they were received in, and empty values count as unset.
type QueryValues struct {
	keys   []string
	values map[string][]string
}

func invalidQuery() error { return failure.New(400, "invalid_query_parameter") }

// Query parses the raw query string. Keys outside allowed, malformed escapes, invalid
// UTF-8, semicolons and more than maxQueryPairs pairs give 400 invalid_query_parameter.
func Query(c fiber.Ctx, allowed ...string) (QueryValues, error) {
	out := QueryValues{keys: []string{}, values: map[string][]string{}}
	raw := string(c.Request().URI().QueryString())
	if raw == "" {
		return out, nil
	}
	pairs := strings.Split(raw, "&")
	if len(pairs) > maxQueryPairs {
		return QueryValues{}, invalidQuery()
	}
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		if strings.Contains(pair, ";") {
			return QueryValues{}, invalidQuery()
		}
		rawKey, rawValue, _ := strings.Cut(pair, "=")
		key, keyErr := url.QueryUnescape(rawKey)
		value, valueErr := url.QueryUnescape(rawValue)
		if keyErr != nil || valueErr != nil || !utf8.ValidString(key) || !utf8.ValidString(value) || !slices.Contains(allowed, key) {
			return QueryValues{}, invalidQuery()
		}
		if value == "" {
			continue
		}
		if _, seen := out.values[key]; !seen {
			out.keys = append(out.keys, key)
		}
		out.values[key] = append(out.values[key], value)
	}
	return out, nil
}

// Keys lists the keys that carry a value, in order of first appearance.
func (q QueryValues) Keys() []string { return append([]string{}, q.keys...) }

// Has reports whether key carries at least one non-empty value.
func (q QueryValues) Has(key string) bool { return len(q.values[key]) > 0 }

// Strings returns every value for key in received order; never nil.
func (q QueryValues) Strings(key string) []string { return append([]string{}, q.values[key]...) }

// String returns the single value for key ("" when unset); a repeated key is ambiguous (400).
func (q QueryValues) String(key string) (string, error) {
	values := q.values[key]
	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	}
	return "", invalidQuery()
}

// Int parses a canonical decimal within [min, max]; set is false when the key is unset.
func (q QueryValues) Int(key string, min, max int) (value int, set bool, err error) {
	raw, err := q.String(key)
	if err != nil || raw == "" {
		return 0, false, err
	}
	n, err := strconv.Atoi(raw)
	if err != nil || strconv.Itoa(n) != raw || n < min || n > max {
		return 0, false, invalidQuery()
	}
	return n, true, nil
}
