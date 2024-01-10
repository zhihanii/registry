package registry

import "fmt"

func serviceKeyPrefix(serviceName string) string {
	return fmt.Sprintf("%s/", serviceName)
}

func serviceKey(serviceName, addr string) string {
	return serviceKeyPrefix(serviceName) + addr
}
