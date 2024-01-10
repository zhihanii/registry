package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zhihanii/discovery"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const defaultWeight = 10

type etcdResolver struct {
	etcdClient *clientv3.Client
}

func (e *etcdResolver) Resolve(ctx context.Context, serviceName string) (res discovery.Result, err error) {
	prefix := serviceKeyPrefix(serviceName)
	resp, err := e.etcdClient.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return
	}
	var instances []discovery.Instance
	for _, kv := range resp.Kvs {
		var info instanceInfo
		err1 := json.Unmarshal(kv.Value, &info)
		if err1 != nil {
			// klog.Warnf("fail to unmarshal with err: %v, ignore key: %v", err, string(kv.Key))
			continue
		}
		weight := info.Weight
		if weight <= 0 {
			weight = defaultWeight
		}
		instances = append(instances, discovery.NewInstance(info.Network, info.Address, weight, info.Tags))
	}
	if len(instances) == 0 {
		err = fmt.Errorf("no instance remains for %v", serviceName)
		return
	}
	return discovery.Result{
		Cacheable: true,
		CacheKey:  serviceName,
		Instances: instances,
	}, nil
}
