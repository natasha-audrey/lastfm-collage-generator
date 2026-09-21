package flags

// Option pairs registration of a flag value F with conversion to a parsed value P.
type Option[F any, P any] struct {
	// Option registers the flag and returns a pointer to its value.
	Option func() *F
	// Parse converts and validates the flag value.
	Parse func(t F) (P, error)
}
