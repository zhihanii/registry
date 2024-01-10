package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zhihanii/discovery"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdRegistry struct {
	etcdClient *clientv3.Client
	leaseTTL   int64
	metadata   *registerMetadata
}

type registerMetadata struct {
	leaseID clientv3.LeaseID
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewEtcdRegistry(endpoints []string, ttl int64, opts ...Option) (Registry, error) {
	cfg := clientv3.Config{
		Endpoints: endpoints,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	etcdClient, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &etcdRegistry{
		etcdClient: etcdClient,
		leaseTTL:   ttl,
	}, nil
}

func (e *etcdRegistry) Register(endpoint *Endpoint) error {
	if err := validateRegistryInfo(endpoint); err != nil {
		return err
	}
	leaseID, err := e.grant()
	if err != nil {
		return err
	}
	if err = e.register(endpoint, leaseID); err != nil {
		return err
	}
	metadata := registerMetadata{
		leaseID: leaseID,
	}
	metadata.ctx, metadata.cancel = context.WithCancel(context.Background())
	if err = e.keepAlive(&metadata); err != nil {
		return err
	}
	e.metadata = &metadata
	return nil
}

func (e *etcdRegistry) Deregister(endpoint *Endpoint) error {
	if endpoint.ServiceName == "" {
		return fmt.Errorf("missing service name in Deregister")
	}
	if err := e.deregister(endpoint); err != nil {
		return err
	}
	e.metadata.cancel()
	return nil
}

func (e *etcdRegistry) register(endpoint *Endpoint, leaseID clientv3.LeaseID) error {
	data, err := json.Marshal(discovery.NewInstance(endpoint.Network,
		endpoint.Address, endpoint.Weight, endpoint.Tags))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = e.etcdClient.Put(ctx, serviceKey(endpoint.ServiceName, endpoint.Address), string(data),
		clientv3.WithLease(leaseID))
	return err
}

func (e *etcdRegistry) deregister(endpoint *Endpoint) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err := e.etcdClient.Delete(ctx, serviceKey(endpoint.ServiceName, endpoint.Address))
	return err
}

func (e *etcdRegistry) keepAlive(metadata *registerMetadata) error {
	keepAlive, err := e.etcdClient.KeepAlive(metadata.ctx, metadata.leaseID)
	if err != nil {
		return err
	}
	go func() {
		// zap.Infof("start keepalive lease %x for etcd registry", metadata.leaseID)
		for range keepAlive {
			select {
			case <-metadata.ctx.Done():
				// klog.Infof("stop keepalive lease %x for etcd registry", meta.leaseID)
				return
			default:
			}
		}
	}()
	return nil
}

func (e *etcdRegistry) grant() (clientv3.LeaseID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resp, err := e.etcdClient.Grant(ctx, e.leaseTTL)
	if err != nil {
		return clientv3.NoLease, err
	}
	return resp.ID, nil
}

func validateRegistryInfo(endpoint *Endpoint) error {
	if endpoint.ServiceName == "" {
		return fmt.Errorf("missing service name in Register")
	}
	if strings.Contains(endpoint.ServiceName, "/") {
		return fmt.Errorf("service name registered with etcd should not include character '/'")
	}
	if endpoint.Network == "" {
		return fmt.Errorf("missing network in Register")
	}
	if endpoint.Address == "" {
		return fmt.Errorf("missing addr in Register")
	}
	return nil
}
