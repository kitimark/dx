package conflictresolver

type ConflictResolver interface {
	Detect(fileNames []string) bool
	Resolve(fileNames []string) error
}

var ConflictResolvers = []ConflictResolver{
	&GoModResolver{},
}
