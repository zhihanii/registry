package registry

import (
	"context"
	"fmt"
	"github.com/zhihanii/zlog"
	clientv3 "go.etcd.io/etcd/client/v3"
	gresolver "google.golang.org/grpc/resolver"
	"sync"
)

var _ gresolver.Builder = (*grpcEtcdResolverBuilder)(nil)

type grpcEtcdResolverBuilder struct {
	c *clientv3.Client
}

func NewGRPCEtcdResolverBuilder(c *clientv3.Client) (gresolver.Builder, error) {
	return &grpcEtcdResolverBuilder{c: c}, nil
}

func (b *grpcEtcdResolverBuilder) Build(target gresolver.Target, cc gresolver.ClientConn, opts gresolver.BuildOptions) (gresolver.Resolver, error) {
	r := &grpcEtcdResolver{
		c:      b.c,
		target: target.Endpoint(),
		cc:     cc,
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

func (b *grpcEtcdResolverBuilder) Scheme() string {
	return "etcd"
}

type grpcEtcdResolver struct {
	c      *clientv3.Client
	target string
	cc     gresolver.ClientConn
	wch    WatchChannel
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (r *grpcEtcdResolver) watch() {
	defer r.wg.Done()

	allUps := make(map[string]*Update)
	for {
		select {
		case <-r.ctx.Done():
			return
		case ups, ok := <-r.wch:
			if !ok {
				zlog.Infof("watch channel closed")
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

			res := convertToGRPCAddresses(allUps)
			_ = r.cc.UpdateState(gresolver.State{Addresses: res})
		}
	}
}

func (r *grpcEtcdResolver) ResolveNow(gresolver.ResolveNowOptions) {}

func (r *grpcEtcdResolver) Close() {
	r.cancel()
	r.wg.Wait()
}

func convertToGRPCAddresses(ups map[string]*Update) []gresolver.Address {
	var res []gresolver.Address
	for _, up := range ups {
		addr := gresolver.Address{
			Addr: fmt.Sprintf("%s:%d", up.Instance.Address(), up.Instance.Port()),
		}
		res = append(res, addr)
	}
	return res
}
