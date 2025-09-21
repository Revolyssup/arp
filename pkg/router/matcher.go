package router

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/Revolyssup/arp/pkg/config"
)

type Matcher interface {
	Match(*http.Request) bool
}

type compositeMatcher struct {
	matchers []Matcher
}

func NewCompositeMatcher(matchConfigs []config.Match) (Matcher, error) {
	var matchers []Matcher

	for _, mc := range matchConfigs {
		if mc.Path != "" {
			pathMatcher, err := newPathMatcher(mc.Path)
			if err != nil {
				return nil, err
			}
			matchers = append(matchers, pathMatcher)
		}

		if len(mc.Headers) > 0 {
			headerMatcher := newHeaderMatcher(mc.Headers)
			matchers = append(matchers, headerMatcher)
		}

		if mc.Method != "" {
			methodMatcher := newMethodMatcher(mc.Method)
			matchers = append(matchers, methodMatcher)
		}
	}

	return &compositeMatcher{matchers: matchers}, nil
}

func (m *compositeMatcher) Match(r *http.Request) bool {
	for _, matcher := range m.matchers {
		if !matcher.Match(r) {
			return false
		}
	}
	return true
}

type pathMatcher struct {
	pattern *regexp.Regexp
}

func newPathMatcher(pattern string) (Matcher, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &pathMatcher{pattern: regex}, nil
}

func (m *pathMatcher) Match(r *http.Request) bool {
	return m.pattern.MatchString(r.URL.Path)
}

type headerMatcher struct {
	headers map[string]string
}

func newHeaderMatcher(headers map[string]string) Matcher {
	return &headerMatcher{headers: headers}
}

func (m *headerMatcher) Match(r *http.Request) bool {
	for key, value := range m.headers {
		if r.Header.Get(key) != value {
			return false
		}
	}
	return true
}

type methodMatcher struct {
	method string
}

func newMethodMatcher(method string) Matcher {
	return &methodMatcher{method: strings.ToUpper(method)}
}

func (m *methodMatcher) Match(r *http.Request) bool {
	return r.Method == m.method
}
