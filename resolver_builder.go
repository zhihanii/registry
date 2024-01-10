package registry

import (
	"github.com/zhihanii/discovery"
)

type ResolverBuilder interface {
	Build(target string, update UpdateFunc) (discovery.Resolver, error)
}
