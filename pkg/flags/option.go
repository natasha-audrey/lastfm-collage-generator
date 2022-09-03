package flags

type Option[F any, P any] struct {
	Option func() *F
	Parse  func(t F) (P, error)
}
