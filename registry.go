package registry

type Registry interface {
	Register(endpoint *Endpoint) error
	Deregister(endpoint *Endpoint) error
}
