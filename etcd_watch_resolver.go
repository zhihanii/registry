package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/zhihanii/discovery"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdWatchResolverBuilder struct {
	c *clientv3.Client
}

func NewEtcdWatchResolverBuilder(c *clientv3.Client) (ResolverBuilder, error) {
	if c == nil {
		return nil, errors.New("invalid etcd client")
	}

	return &etcdWatchResolverBuilder{c: c}, nil
}

func (b *etcdWatchResolverBuilder) Build(target string, update UpdateFunc) (discovery.Resolver, error) {
	r := &etcdWatchResolver{
		c:      b.c,
		target: target,
		update: update,
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	w, err := newWatcher(r.c, r.target)
	if err != nil {
		return nil, fmt.Errorf("resolver: failed to new watcher: %s", err)
	}
	r.wch, err = w.watchChannel(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("resolver: failed to new watch channel: %s", err)
	}

	r.wg.Add(1)
	go r.watch()
	return r, nil
}

type Operation uint8

const (
	Add Operation = iota
	Delete
)

type Update struct {
	Op       Operation
	Key      string
	Instance discovery.Instance
}

type WatchChannel <-chan []*Update

type UpdateFunc func(discovery.Result) error

type etcdWatchResolver struct {
	c      *clientv3.Client
	target string
	update UpdateFunc
	wch    WatchChannel
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (r *etcdWatchResolver) Target(ctx context.Context) string {
	return r.target
}

func (r *etcdWatchResolver) Resolve(ctx context.Context, target string) (res discovery.Result, err error) {
	return
}

func (r *etcdWatchResolver) Close() {
	r.cancel()
	r.wg.Wait()
}

func (r *etcdWatchResolver) watch() {
	defer r.wg.Done()

	allUps := make(map[string]*Update)
	for {
		select {
		case <-r.ctx.Done():
			return
		case ups, ok := <-r.wch:
			if !ok {
				return
			}

			for _, up := range ups {
				switch up.Op {
				case Add:
					allUps[up.Key] = up
				case Delete:
					delete(allUps, up.Key)
				}
			}

			res := convertToDiscoveryResult(r.target, allUps)
			_ = r.update(res)
		}
	}
}

func convertToDiscoveryResult(target string, ups map[string]*Update) discovery.Result {
	instances := make([]discovery.Instance, 0, len(ups))
	for _, up := range ups {
		instances = append(instances, up.Instance)
	}
	return discovery.Result{
		Cacheable: true,
		CacheKey:  target,
		Instances: instances,
	}
}
