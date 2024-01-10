package registry

type Endpoint struct {
	ServiceName string
	Network     string
	Addr        string
	Weight      int
	Tags        map[string]string
}
