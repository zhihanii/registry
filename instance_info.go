package registry

type instanceInfo struct {
	Network string `json:"network"`
	Address string `json:"address"`
	Weight int `json:"weight"`
	Tags map[string]string `json:"tags"`
}
