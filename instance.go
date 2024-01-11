package registry

type instance struct {
	Network string            `json:"network"`
	Address string            `json:"address"` //host
	Port    int               `json:"port"`
	Weight  int               `json:"weight"`
	Tags    map[string]string `json:"tags"`
}
