package registry

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/zhihanii/discovery"
	"github.com/zhihanii/zlog"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type watcher struct {
	etcdClient *clientv3.Client
	target     string
}

func newWatcher(c *clientv3.Client, target string) (*watcher, error) {
	if c == nil {
		return nil, errors.New("invalid etcd client")
	}

	w := &watcher{
		etcdClient: c,
		target:     target,
	}
	return w, nil
}

func (w *watcher) watchChannel(ctx context.Context) (WatchChannel, error) {
	ctx1, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	resp, err := w.etcdClient.Get(ctx1, w.target, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	initUpdates := make([]*Update, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var ins instance
		if err1 := json.Unmarshal(kv.Value, &ins); err1 != nil {
			zlog.Errorf("unmarshal:%v", err1)
			continue
		}
		up := &Update{
			Op:  Add,
			Key: string(kv.Key),
			Instance: discovery.NewInstance(
				ins.Network,
				ins.Address,
				ins.Port,
				ins.Weight,
				ins.Tags,
			),
		}
		initUpdates = append(initUpdates, up)
	}

	upch := make(chan []*Update, 1)
	if len(initUpdates) > 0 {
		upch <- initUpdates
	}
	go w.watch(ctx, resp.Header.Revision+1, upch)
	return upch, nil
}

func (w *watcher) watch(ctx context.Context, rev int64, upch chan []*Update) {
	defer close(upch)

	opts := []clientv3.OpOption{clientv3.WithRev(rev), clientv3.WithPrefix()}
	wch := w.etcdClient.Watch(ctx, w.target, opts...)
	for {
		select {
		case <-ctx.Done():
			return
		case wresp, ok := <-wch:
			if !ok {
				zlog.Infof("watch channel closed")
				return
			}
			if wresp.Err() != nil {
				zlog.Errorf("watch response err:%v", wresp.Err())
				return
			}

			deltaUps := make([]*Update, 0, len(wresp.Events))
			for _, e := range wresp.Events {
				var (
					ins instance
					err error
					op  Operation
				)
				switch e.Type {
				case clientv3.EventTypePut:
					err = json.Unmarshal(e.Kv.Value, &ins)
					op = Add
					if err != nil {
						zlog.Errorf("unmarshal:%v", err)
						continue
					}
				case clientv3.EventTypeDelete:
					op = Delete
				default:
					continue
				}
				up := &Update{
					Op:  op,
					Key: string(e.Kv.Key),
					Instance: discovery.NewInstance(
						ins.Network,
						ins.Address,
						ins.Port,
						ins.Weight,
						ins.Tags,
					),
				}
				deltaUps = append(deltaUps, up)
			}
			if len(deltaUps) > 0 {
				upch <- deltaUps
			}
		}
	}
}
