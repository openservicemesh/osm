package identity

// ServiceIdentity is the type used to represent the identity for a service
type ServiceIdentity string

// String returns the ServiceIdentity as a string
func (si ServiceIdentity) String() string {
	return string(si)
}
