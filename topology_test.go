package main

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func makeNode(name string, ready bool) *corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}},
		},
	}
}

func makePod(ns, name, node string, phase corev1.PodPhase, terminating bool) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			NodeName:   node,
			Containers: []corev1.Container{{Name: "c", Image: "example.test/img:1"}},
		},
		Status: corev1.PodStatus{Phase: phase},
	}
	if terminating {
		now := metav1.Now()
		p.DeletionTimestamp = &now
	}
	return p
}

func podByName(t *testing.T, snap *Snapshot, name string) PodInfo {
	t.Helper()
	for _, p := range snap.Pods {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("pod %q not in snapshot", name)
	return PodInfo{}
}

// The zombie-coredns regression test: a Running pod on a node whose kubelet
// has stopped reporting must be marked stale, never presented as healthy.
func TestStalePodsOnNotReadyNodes(t *testing.T) {
	snap := assembleSnapshot("test",
		[]*corev1.Node{makeNode("dead", false), makeNode("alive", true)},
		[]*corev1.Pod{
			makePod("kube-system", "zombie", "dead", corev1.PodRunning, false),
			makePod("kube-system", "healthy", "alive", corev1.PodRunning, false),
		}, nil)

	if !podByName(t, snap, "zombie").Stale {
		t.Error("Running pod on NotReady node must be stale")
	}
	if podByName(t, snap, "healthy").Stale {
		t.Error("Running pod on Ready node must not be stale")
	}
}

func TestTerminatingPods(t *testing.T) {
	snap := assembleSnapshot("test",
		[]*corev1.Node{makeNode("n1", true)},
		[]*corev1.Pod{makePod("default", "dying", "n1", corev1.PodRunning, true)}, nil)

	p := podByName(t, snap, "dying")
	if !p.Terminating {
		t.Error("pod with deletionTimestamp must be terminating")
	}
	if p.Stale {
		t.Error("terminating pod on a healthy node is not stale")
	}
}

func TestUnscheduledPods(t *testing.T) {
	snap := assembleSnapshot("test",
		[]*corev1.Node{makeNode("n1", true)},
		[]*corev1.Pod{makePod("default", "waiting", "", corev1.PodPending, false)}, nil)

	p := podByName(t, snap, "waiting")
	if p.Stale {
		t.Error("unscheduled pod must not be stale — it has no node to distrust")
	}
	if p.Node != "" {
		t.Errorf("unscheduled pod node = %q, want empty", p.Node)
	}
}

func TestSnapshotOrderIsDeterministic(t *testing.T) {
	pods := []*corev1.Pod{
		makePod("zz", "a", "n1", corev1.PodRunning, false),
		makePod("aa", "b", "n1", corev1.PodRunning, false),
		makePod("aa", "a", "n1", corev1.PodRunning, false),
	}
	snap := assembleSnapshot("test", []*corev1.Node{makeNode("n1", true)}, pods, nil)
	got := []string{}
	for _, p := range snap.Pods {
		got = append(got, p.Namespace+"/"+p.Name)
	}
	want := []string{"aa/a", "aa/b", "zz/a"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pod order = %v, want %v", got, want)
		}
	}
}

// End-to-end through the informer machinery against the fake clientset:
// the initial List populates the cache, and a pod created afterwards must
// arrive in the snapshot via the watch stream — no further Lists involved.
func TestStoreServesFromWatchCache(t *testing.T) {
	client := fake.NewClientset(
		makeNode("n1", true),
		makePod("default", "existing", "n1", corev1.PodRunning, false),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := NewStore(ctx, client)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	syncCtx, syncCancel := context.WithTimeout(ctx, 5*time.Second)
	defer syncCancel()
	if err := store.WaitForSync(syncCtx); err != nil {
		t.Fatalf("WaitForSync: %v", err)
	}

	snap, err := store.Snapshot("test")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Nodes) != 1 || len(snap.Pods) != 1 {
		t.Fatalf("initial snapshot: %d nodes, %d pods; want 1, 1", len(snap.Nodes), len(snap.Pods))
	}

	_, err = client.CoreV1().Pods("default").Create(ctx,
		makePod("default", "arrived-by-watch", "n1", corev1.PodPending, false),
		metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("creating pod: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		snap, err := store.Snapshot("test")
		if err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
		if len(snap.Pods) == 2 {
			podByName(t, snap, "arrived-by-watch")
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pod never arrived via watch; snapshot has %d pods", len(snap.Pods))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestStaleSinceTracksWatchErrorStreaks(t *testing.T) {
	s := &Store{}
	if s.staleSince() != nil {
		t.Fatal("new store must not report stale")
	}
	s.recordWatchError(nil, context.DeadlineExceeded)
	first := s.staleSince()
	if first == nil {
		t.Fatal("store must report stale immediately after a watch error")
	}
	s.recordWatchError(nil, context.DeadlineExceeded)
	second := s.staleSince()
	if second == nil || !second.Equal(*first) {
		t.Error("repeated errors in one streak must keep the original staleSince")
	}
}
