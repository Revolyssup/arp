package route

import (
	"net/http"
	"regexp"
	"strings"
)

// PathMatcher uses a prefix tree for efficient path matching
type PathMatcher struct {
	staticRoutes map[string][]*Route
	regexRoutes  []struct {
		pattern *regexp.Regexp
		routes  []*Route
	}
	prefixRoutes map[string][]*Route
}

func NewPathMatcher() *PathMatcher {
	return &PathMatcher{
		staticRoutes: make(map[string][]*Route),
		regexRoutes: make([]struct {
			pattern *regexp.Regexp
			routes  []*Route
		}, 0),
		prefixRoutes: make(map[string][]*Route),
	}
}

func (pm *PathMatcher) Add(pattern string, route *Route) {
	// Check if it's a regex pattern (contains special characters)
	if strings.ContainsAny(pattern, ".*+?()|[]{}^$") {
		regex, err := regexp.Compile(pattern)
		if err == nil {
			pm.regexRoutes = append(pm.regexRoutes, struct {
				pattern *regexp.Regexp
				routes  []*Route
			}{pattern: regex, routes: []*Route{route}})
		}
		return
	}

	// Check if it's a prefix pattern (ends with *)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		pm.prefixRoutes[prefix] = append(pm.prefixRoutes[prefix], route)
		return
	}

	// Static path
	pm.staticRoutes[pattern] = append(pm.staticRoutes[pattern], route)
}

func (pm *PathMatcher) Match(path string) []*Route {
	var matches []*Route

	// Check static routes first (exact match)
	if routes, exists := pm.staticRoutes[path]; exists {
		matches = append(matches, routes...)
	}

	// Check prefix routes
	for prefix, routes := range pm.prefixRoutes {
		if strings.HasPrefix(path, prefix) {
			matches = append(matches, routes...)
		}
	}

	// Check regex routes
	for _, regexRoute := range pm.regexRoutes {
		if regexRoute.pattern.MatchString(path) {
			matches = append(matches, regexRoute.routes...)
		}
	}

	return matches
}

func (pm *PathMatcher) Clear() {
	pm.staticRoutes = make(map[string][]*Route)
	pm.regexRoutes = make([]struct {
		pattern *regexp.Regexp
		routes  []*Route
	}, 0)
	pm.prefixRoutes = make(map[string][]*Route)
}

// MethodMatcher uses a simple map for method matching
type MethodMatcher struct {
	routes map[string][]*Route
}

func NewMethodMatcher() *MethodMatcher {
	return &MethodMatcher{
		routes: make(map[string][]*Route),
	}
}

func (mm *MethodMatcher) Add(method string, route *Route) {
	mm.routes[method] = append(mm.routes[method], route)
}

func (mm *MethodMatcher) Match(method string) []*Route {
	return mm.routes[method]
}

func (mm *MethodMatcher) Clear() {
	mm.routes = make(map[string][]*Route)
}

// HeaderMatcher handles header-based matching
type HeaderMatcher struct {
	headerRoutes map[string]map[string][]*Route // headerKey -> headerValue -> routes
}

func NewHeaderMatcher() *HeaderMatcher {
	return &HeaderMatcher{
		headerRoutes: make(map[string]map[string][]*Route),
	}
}

func (hm *HeaderMatcher) Add(headers map[string]string, route *Route) {
	for key, value := range headers {
		if _, exists := hm.headerRoutes[key]; !exists {
			hm.headerRoutes[key] = make(map[string][]*Route)
		}
		hm.headerRoutes[key][value] = append(hm.headerRoutes[key][value], route)
	}
}

func (hm *HeaderMatcher) Match(requestHeaders http.Header, candidateRoutes []*Route) []*Route {
	if len(hm.headerRoutes) == 0 {
		return candidateRoutes
	}

	candidateSet := make(map[*Route]bool)
	for _, route := range candidateRoutes {
		candidateSet[route] = true
	}

	// Filter routes that match all required headers
	var matchedRoutes []*Route
	for _, route := range candidateRoutes {
		matchesAll := true

		for headerKey, expectedValues := range hm.headerRoutes {
			actualValue := requestHeaders.Get(headerKey)
			if actualValue == "" {
				matchesAll = false
				break
			}

			// Check if this route requires this specific header
			routeMatches := false
			for expectedValue, routes := range expectedValues {
				if actualValue == expectedValue {
					// Check if this route is in the routes for this header value
					for _, r := range routes {
						if r == route {
							routeMatches = true
							break
						}
					}
					break
				}
			}

			if !routeMatches {
				matchesAll = false
				break
			}
		}

		if matchesAll {
			matchedRoutes = append(matchedRoutes, route)
		}
	}

	return matchedRoutes
}

func (hm *HeaderMatcher) Clear() {
	hm.headerRoutes = make(map[string]map[string][]*Route)
}
