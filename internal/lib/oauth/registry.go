package oauth

// Registry resolves a Provider by its lowercase name.
type Registry map[string]Provider

// NewRegistry constructs a Registry that contains only the non-nil providers.
func NewRegistry(providers ...Provider) Registry {
	registry := Registry{}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		registry[provider.Name()] = provider
	}
	return registry
}

// Get returns the registered Provider and a boolean for presence.
func (r Registry) Get(name string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r[name]
	return provider, ok
}
