package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

type StaticValidator struct {
	errors []ValidationError
}

func NewStaticValidator() *StaticValidator {
	return &StaticValidator{
		errors: make([]ValidationError, 0),
	}
}

// Validate performs all validation checks on the static configuration
func (v *StaticValidator) Validate(cfg *Static) error {
	v.errors = make([]ValidationError, 0) // Reset errors

	v.validateListeners(cfg.Listeners)
	v.validateProviders(cfg.Providers)
	v.validateDiscoveryConfigs(cfg.DiscoveryConfigs)

	if len(v.errors) > 0 {
		return v.ToError()
	}
	return nil
}

// validateListeners validates listener configuration
func (v *StaticValidator) validateListeners(listeners []ListenerConfig) {
	seenNames := make(map[string]bool)
	seenPorts := make(map[int]bool)

	for i, listener := range listeners {
		// Check for empty name
		if strings.TrimSpace(listener.Name) == "" {
			v.addError(fmt.Sprintf("listeners[%d].name", i), "listener name cannot be empty")
		}

		// Check for duplicate names
		if seenNames[listener.Name] {
			v.addError(fmt.Sprintf("listeners[%d].name", i),
				fmt.Sprintf("duplicate listener name: %s", listener.Name))
		}
		seenNames[listener.Name] = true

		// Check for valid port
		if listener.Port < 1 || listener.Port > 65535 {
			v.addError(fmt.Sprintf("listeners[%d].port", i),
				fmt.Sprintf("invalid port number: %d (must be between 1-65535)", listener.Port))
		}

		// Check for duplicate ports
		if seenPorts[listener.Port] {
			v.addError(fmt.Sprintf("listeners[%d].port", i),
				fmt.Sprintf("duplicate port: %d", listener.Port))
		}
		seenPorts[listener.Port] = true

		// StaticValidate TLS configuration if present
		if listener.TLS != nil {
			v.validateTLSConfig(i, listener.TLS)
		}
	}
}

// validateTLSConfig validates TLS configuration for a listener
func (v *StaticValidator) validateTLSConfig(listenerIndex int, tls *TLSConfig) {
	if strings.TrimSpace(tls.CertFile) == "" {
		v.addError(fmt.Sprintf("listeners[%d].tls.certFile", listenerIndex),
			"TLS certificate file path cannot be empty")
	}

	if strings.TrimSpace(tls.KeyFile) == "" {
		v.addError(fmt.Sprintf("listeners[%d].tls.keyFile", listenerIndex),
			"TLS key file path cannot be empty")
	}
}

func (v *StaticValidator) validateProviders(providers []ProviderConfig) {
	seenNames := make(map[string]bool)

	for i, provider := range providers {
		// Check for empty name
		if strings.TrimSpace(provider.Name) == "" {
			v.addError(fmt.Sprintf("providers[%d].name", i), "provider name cannot be empty")
		}

		// Check for duplicate names
		if seenNames[provider.Name] {
			v.addError(fmt.Sprintf("providers[%d].name", i),
				fmt.Sprintf("duplicate provider name: %s", provider.Name))
		}
		seenNames[provider.Name] = true

		// Check for empty type
		if strings.TrimSpace(provider.Type) == "" {
			v.addError(fmt.Sprintf("providers[%d].type", i), "provider type cannot be empty")
		}
	}
}

// validateDiscoveryConfigs validates discovery configuration
func (v *StaticValidator) validateDiscoveryConfigs(discoveries []DiscoveryConfig) {
	seenTypes := make(map[string]bool)

	for i, discovery := range discoveries {
		// Check for empty type
		if strings.TrimSpace(discovery.Type) == "" {
			v.addError(fmt.Sprintf("discovery[%d].type", i), "discovery type cannot be empty")
		}

		// Check for duplicate types
		if seenTypes[discovery.Type] {
			v.addError(fmt.Sprintf("discovery[%d].type", i),
				fmt.Sprintf("duplicate discovery type: %s", discovery.Type))
		}
		seenTypes[discovery.Type] = true
	}
}

func (v *StaticValidator) addError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (v *StaticValidator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *StaticValidator) GetErrors() []ValidationError {
	return v.errors
}

func (v *StaticValidator) ToError() error {
	if len(v.errors) == 0 {
		return nil
	}

	var errorMessages []string
	for _, err := range v.errors {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("configuration validation failed:\n%s",
		strings.Join(errorMessages, "\n"))
}
