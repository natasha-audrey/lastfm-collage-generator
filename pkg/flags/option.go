package flags

// Struct for adding a new command line option to the CLI
// The Option function returns a cli flag, and the parse function parses the
// provided input.
type Option[F any, P any] struct {
	Option func() *F
	Parse  func(t F) (P, error)
}
