package registry

type ResolverBuilder interface {
	Build(target string, update UpdateFunc) (Resolver, error)
}

type WatchResolverBuilder interface {
	Build(target string, update UpdateFunc) (WatchResolver, error)
}
