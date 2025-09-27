package config

import (
	"fmt"
	"net/url"
	"strings"
)

// TODO: Make dynamic validation's validator registerable by each component (e.g., plugins, discovery, etc.). (maybe jsonschema validation?)
type DynamicValidator struct {
	errors []ValidationError
}

func NewDynamicValidator() *DynamicValidator {
	return &DynamicValidator{
		errors: make([]ValidationError, 0),
	}
}

func (v *DynamicValidator) Validate(cfg *Dynamic) error {
	v.errors = make([]ValidationError, 0) // Reset errors

	v.validateUniqueNames(cfg)
	v.validateRoutes(cfg.Routes)
	v.validateUpstreams(cfg.Upstreams)
	v.validatePlugins(cfg.Plugins)
	v.validateStreamRoutes(cfg.StreamRoute)

	if len(v.errors) > 0 {
		return v.ToError()
	}
	return nil
}

func (v *DynamicValidator) validateUniqueNames(cfg *Dynamic) {
	seenNames := make(map[string]string) // map[name]type

	// Check routes
	for i, route := range cfg.Routes {
		if existingType, exists := seenNames[route.Name]; exists {
			v.addError(fmt.Sprintf("routes[%d].name", i),
				fmt.Sprintf("duplicate name '%s' already used by %s", route.Name, existingType))
		} else {
			seenNames[route.Name] = "route"
		}
	}

	// Check upstreams
	for i, upstream := range cfg.Upstreams {
		if existingType, exists := seenNames[upstream.Name]; exists {
			v.addError(fmt.Sprintf("upstreams[%d].name", i),
				fmt.Sprintf("duplicate name '%s' already used by %s", upstream.Name, existingType))
		} else {
			seenNames[upstream.Name] = "upstream"
		}
	}

	// Check plugins
	for i, plugin := range cfg.Plugins {
		if existingType, exists := seenNames[plugin.Name]; exists {
			v.addError(fmt.Sprintf("plugins[%d].name", i),
				fmt.Sprintf("duplicate name '%s' already used by %s", plugin.Name, existingType))
		} else {
			seenNames[plugin.Name] = "plugin"
		}
	}

	// Check stream routes
	for i, streamRoute := range cfg.StreamRoute {
		if existingType, exists := seenNames[streamRoute.Name]; exists {
			v.addError(fmt.Sprintf("streamRoutes[%d].name", i),
				fmt.Sprintf("duplicate name '%s' already used by %s", streamRoute.Name, existingType))
		} else {
			seenNames[streamRoute.Name] = "streamRoute"
		}
	}
}

func (v *DynamicValidator) validateRoutes(routes []RouteConfig) {
	for i, route := range routes {
		// Check for empty name
		if strings.TrimSpace(route.Name) == "" {
			v.addError(fmt.Sprintf("routes[%d].name", i), "route name cannot be empty")
		}

		// Check for empty listener
		if strings.TrimSpace(route.Listener) == "" {
			v.addError(fmt.Sprintf("routes[%d].listener", i), "route listener cannot be empty")
		}

		// Validate matches
		if len(route.Matches) == 0 {
			v.addError(fmt.Sprintf("routes[%d].matches", i), "route must have at least one match condition")
		}

		// Validate each match
		for j, match := range route.Matches {
			matchPrefix := fmt.Sprintf("routes[%d].matches[%d]", i, j)

			// At least one of path, headers, or method should be specified
			if strings.TrimSpace(match.Path) == "" && len(match.Headers) == 0 && strings.TrimSpace(match.Method) == "" {
				v.addError(matchPrefix, "match must specify at least one of: path, headers, or method")
			}

			// Validate path if present
			if match.Path != "" && !strings.HasPrefix(match.Path, "/") {
				v.addError(matchPrefix+".path", "path must start with '/'")
			}

			// Validate method if present
			if match.Method != "" {
				validMethods := map[string]bool{
					"GET": true, "POST": true, "PUT": true, "DELETE": true,
					"PATCH": true, "HEAD": true, "OPTIONS": true, "CONNECT": true, "TRACE": true,
				}
				if !validMethods[strings.ToUpper(match.Method)] {
					v.addError(matchPrefix+".method",
						fmt.Sprintf("invalid HTTP method: %s", match.Method))
				}
			}
		}

		// Validate upstream reference
		if route.Upstream != nil {
			v.validateUpstreamReference(fmt.Sprintf("routes[%d].upstream", i), *route.Upstream)
		} else {
			v.addError(fmt.Sprintf("routes[%d].upstream", i), "route must have an upstream configuration")
		}

		// Validate plugins
		for j, plugin := range route.Plugins {
			v.validatePluginReference(fmt.Sprintf("routes[%d].plugins[%d]", i, j), plugin)
		}
	}
}

// validateUpstreams validates upstream configurations
func (v *DynamicValidator) validateUpstreams(upstreams []UpstreamConfig) {
	for i, upstream := range upstreams {
		if strings.TrimSpace(upstream.Name) == "" {
			v.addError(fmt.Sprintf("upstreams[%d].name", i), "upstream name cannot be empty")
		}

		v.validateUpstreamConfig(fmt.Sprintf("upstreams[%d]", i), upstream)
	}
}

func (v *DynamicValidator) validateUpstreamConfig(prefix string, upstream UpstreamConfig) {
	if upstream.Discovery.Type != "" {
		if strings.TrimSpace(upstream.Service) == "" {
			v.addError(prefix+".service",
				"service cannot be empty when discovery is configured")
		}
	} else {
		// If no discovery, must have nodes
		if len(upstream.Nodes) == 0 {
			v.addError(prefix+".nodes",
				"upstream must have either discovery or static nodes")
		}

		// Validate nodes
		for j, node := range upstream.Nodes {
			v.validateNode(prefix+fmt.Sprintf(".nodes[%d]", j), node)
		}
	}
}

// validateUpstreamReference validates an upstream reference (used in routes)
func (v *DynamicValidator) validateUpstreamReference(prefix string, upstream UpstreamConfig) {
	if strings.TrimSpace(upstream.Name) == "" && upstream.Discovery.Type == "" {
		v.addError(prefix, "upstream reference must have either name or discovery")
	}

	if upstream.Discovery.Type != "" && strings.TrimSpace(upstream.Service) == "" {
		v.addError(prefix+".service",
			"service cannot be empty when discovery is configured")
	}
}

func (v *DynamicValidator) validateNode(prefix string, node Node) {
	if strings.TrimSpace(node.URL) == "" {
		v.addError(prefix+".url", "node URL cannot be empty")
		return
	}

	// Validate URL format
	parsedURL, err := url.Parse(node.URL)
	if err != nil {
		v.addError(prefix+".url", fmt.Sprintf("invalid URL: %s", err.Error()))
		return
	}

	if parsedURL.Scheme == "" {
		v.addError(prefix+".url", "URL must include scheme (http:// or https://)")
	}

	if parsedURL.Host == "" {
		v.addError(prefix+".url", "URL must include host")
	}
}

func (v *DynamicValidator) validatePlugins(plugins []PluginConfig) {
	for i, plugin := range plugins {
		// Check for empty name
		if strings.TrimSpace(plugin.Name) == "" {
			v.addError(fmt.Sprintf("plugins[%d].name", i), "plugin name cannot be empty")
		}

		// Check for empty type
		if strings.TrimSpace(plugin.Type) == "" {
			v.addError(fmt.Sprintf("plugins[%d].type", i), "plugin type cannot be empty")
		}
	}
}

func (v *DynamicValidator) validatePluginReference(prefix string, plugin PluginConfig) {
	if strings.TrimSpace(plugin.Name) == "" {
		v.addError(prefix+".name", "plugin name cannot be empty")
	}
}

func (v *DynamicValidator) validateStreamRoutes(streamRoutes []StreamRouteConfig) {
	for i, streamRoute := range streamRoutes {
		// Check for empty name
		if strings.TrimSpace(streamRoute.Name) == "" {
			v.addError(fmt.Sprintf("streamRoutes[%d].name", i), "stream route name cannot be empty")
		}

		// Check for empty listener
		if strings.TrimSpace(streamRoute.Listener) == "" {
			v.addError(fmt.Sprintf("streamRoutes[%d].listener", i), "stream route listener cannot be empty")
		}

		// Validate upstream if present
		if streamRoute.Upstream != nil {
			v.validateUpstreamReference(fmt.Sprintf("streamRoutes[%d].upstream", i), *streamRoute.Upstream)
		}

		// Validate plugins
		for j, plugin := range streamRoute.Plugins {
			v.validatePluginReference(fmt.Sprintf("streamRoutes[%d].plugins[%d]", i, j), plugin)
		}
	}
}

func (v *DynamicValidator) addError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *DynamicValidator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *DynamicValidator) GetErrors() []ValidationError {
	return v.errors
}

func (v *DynamicValidator) ToError() error {
	if len(v.errors) == 0 {
		return nil
	}

	var errorMessages []string
	for _, err := range v.errors {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("dynamic configuration validation failed:\n%s",
		strings.Join(errorMessages, "\n"))
}
