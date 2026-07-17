package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// degradedWindow is how recently the watch must have errored for the snapshot
// to be reported stale. While a stream is genuinely broken the reflector's
// retries keep the errors coming at least this often, so silence for longer
// than this window means the watch has recovered.
const degradedWindow = 30 * time.Second

// Store is a watch-based in-memory mirror of the cluster's nodes and pods.
// After the initial List, reads never touch the network: the informers'
// watch streams keep the cache current and Snapshot serves from RAM.
type Store struct {
	nodeLister corelisters.NodeLister
	podLister  corelisters.PodLister
	synced     []cache.InformerSynced

	mu          sync.Mutex
	streakStart time.Time // first watch error of the current failure streak
	lastErr     time.Time
	lastErrMsg  string
}

func NewStore(ctx context.Context, client kubernetes.Interface) (*Store, error) {
	factory := informers.NewSharedInformerFactory(client, 0)
	nodeInformer := factory.Core().V1().Nodes().Informer()
	podInformer := factory.Core().V1().Pods().Informer()

	s := &Store{
		nodeLister: factory.Core().V1().Nodes().Lister(),
		podLister:  factory.Core().V1().Pods().Lister(),
		synced:     []cache.InformerSynced{nodeInformer.HasSynced, podInformer.HasSynced},
	}
	for _, inf := range []cache.SharedIndexInformer{nodeInformer, podInformer} {
		if err := inf.SetWatchErrorHandler(s.recordWatchError); err != nil {
			return nil, err
		}
	}
	factory.Start(ctx.Done())
	return s, nil
}

// WaitForSync blocks until both caches hold a complete initial List, or the
// context expires. A sync that never completes is almost always RBAC or
// connectivity, so the error says so — with the watch's own words if we have
// them.
func (s *Store) WaitForSync(ctx context.Context) error {
	if !cache.WaitForCacheSync(ctx.Done(), s.synced...) {
		s.mu.Lock()
		msg := s.lastErrMsg
		s.mu.Unlock()
		if msg != "" {
			return fmt.Errorf("timed out syncing cluster state (last watch error: %s)", msg)
		}
		return fmt.Errorf("timed out syncing cluster state — check connectivity and RBAC (list/watch on nodes and pods)")
	}
	return nil
}

func (s *Store) Snapshot(kubeContext string) (*Snapshot, error) {
	nodes, err := s.nodeLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	pods, err := s.podLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return assembleSnapshot(kubeContext, nodes, pods, s.staleSince()), nil
}

func (s *Store) recordWatchError(_ *cache.Reflector, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.lastErr) > degradedWindow {
		s.streakStart = now
	}
	s.lastErr = now
	s.lastErrMsg = err.Error()
}

// staleSince reports the start of the current watch-failure streak, or nil
// when the watch is healthy — ADR-0002's honesty rule applied to our own
// cache rather than the kubelet's.
func (s *Store) staleSince() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastErr.IsZero() && time.Since(s.lastErr) < degradedWindow {
		t := s.streakStart
		return &t
	}
	return nil
}
