package registry

import "fmt"

func serviceKeyPrefix(serviceName string) string {
	return fmt.Sprintf("%s/", serviceName)
}

func serviceKey(serviceName, addr string, port int) string {
	//return serviceKeyPrefix(serviceName) + addr
	return fmt.Sprintf("%s/%s:%d", serviceName, addr, port)
}
