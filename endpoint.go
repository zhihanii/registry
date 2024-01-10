package registry

type Endpoint struct {
	ServiceName string
	Network     string
	Address     string
	Port        int
	Weight      int
	Tags        map[string]string
}
