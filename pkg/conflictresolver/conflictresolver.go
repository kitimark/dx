package conflictresolver

type ConflictResolver interface {
	Name() string
	Detect(fileNames []string) bool
	Resolve(fileNames []string) error
}

var ConflictResolvers = []ConflictResolver{
	&GoModResolver{},
	&YarnLockResolver{},
}
