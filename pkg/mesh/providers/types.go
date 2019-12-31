package providers

import "errors"

type Provider int

// The goal of the enums below is to categorize compute providers.
// Knowing the kind of provider would allow us to invoke appropriate
// mesh module to retrieve networking context: routing, IP addresses etc.
const (
	// Azure compute provider
	Azure Provider = iota + 1

	// Kubernetes compute provider
	Kubernetes
)

var friendlyNames = map[Provider]string{
	Azure:      "Azure",
	Kubernetes: "Kubernetes",
}

var ErrFriendlyNameNotFound = errors.New("friendly name not found")

func GetFriendlyName(provider Provider) (string, error) {
	if name, ok := friendlyNames[provider]; ok {
		return name, nil
	}
	return "", ErrFriendlyNameNotFound
}
