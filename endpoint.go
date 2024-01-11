package registry

type Endpoint struct {
	ServiceName string
	Network     string
	Address     string //host
	Port        int
	Weight      int
	Tags        map[string]string
}
