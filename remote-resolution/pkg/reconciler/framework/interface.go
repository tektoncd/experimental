package framework

import "context"

type Resolver interface {
	// Initialize is called at the moment the resolver controller is
	// instantiated and is a good place to setup things like
	// resource listers.
	Initialize(context.Context) error

	// GetName should give back the name of the resolver. E.g. "Git"
	GetName() string

	// These are the labels that are used to direct resolution
	// requests to the right resolver.
	GetSelector() map[string]string

	// ValidateParams is given the parameters from a ResourceRequest
	// and should return an error if any are missing or invalid.
	ValidateParams(map[string]string) error

	// Resolve receives the parameters passed with a ResourceRequest
	// and returns the resolved data as a string and any annotations
	// to include in the response or an error. If a resolution.Error
	// is returned then its Reason and Message are utilized in the
	// failed Condition of the ResourceRequest.
	Resolve(map[string]string) (string, map[string]string, error)
}
