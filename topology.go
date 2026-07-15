package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeInfo struct {
	Name       string    `json:"name"`
	Ready      bool      `json:"ready"`
	Roles      []string  `json:"roles"`
	Version    string    `json:"version"`
	InternalIP string    `json:"internalIP"`
	CPU        string    `json:"cpu"`
	Memory     string    `json:"memory"`
	Pressures  []string  `json:"pressures"`
	CreatedAt  time.Time `json:"createdAt"`
}

type PodInfo struct {
	Namespace       string    `json:"namespace"`
	Name            string    `json:"name"`
	Node            string    `json:"node"`
	Phase           string    `json:"phase"`
	Terminating     bool      `json:"terminating"`
	Stale           bool      `json:"stale"`
	Restarts        int32     `json:"restarts"`
	ReadyContainers int       `json:"readyContainers"`
	TotalContainers int       `json:"totalContainers"`
	Owner           string    `json:"owner"`
	Images          []string  `json:"images"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Snapshot struct {
	Context   string     `json:"context"`
	FetchedAt time.Time  `json:"fetchedAt"`
	Nodes     []NodeInfo `json:"nodes"`
	Pods      []PodInfo  `json:"pods"`
}

func buildSnapshot(ctx context.Context, client kubernetes.Interface, kubeContext string) (*Snapshot, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	snap := &Snapshot{Context: kubeContext, FetchedAt: time.Now()}

	nodeReady := make(map[string]bool, len(nodes.Items))
	for _, n := range nodes.Items {
		ready := false
		var pressures []string
		for _, cond := range n.Status.Conditions {
			switch cond.Type {
			case corev1.NodeReady:
				ready = cond.Status == corev1.ConditionTrue
			case corev1.NodeMemoryPressure, corev1.NodeDiskPressure, corev1.NodePIDPressure:
				if cond.Status == corev1.ConditionTrue {
					pressures = append(pressures, string(cond.Type))
				}
			}
		}
		internalIP := ""
		for _, addr := range n.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
			}
		}
		nodeReady[n.Name] = ready
		snap.Nodes = append(snap.Nodes, NodeInfo{
			Name:       n.Name,
			Ready:      ready,
			Roles:      nodeRoles(&n),
			Version:    n.Status.NodeInfo.KubeletVersion,
			InternalIP: internalIP,
			CPU:        n.Status.Allocatable.Cpu().String(),
			Memory:     fmt.Sprintf("%.1fGi", float64(n.Status.Allocatable.Memory().Value())/(1<<30)),
			Pressures:  pressures,
			CreatedAt:  n.CreationTimestamp.Time,
		})
	}

	for _, p := range pods.Items {
		ready, onKnownNode := nodeReady[p.Spec.NodeName]
		var restarts int32
		readyContainers := 0
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
			if cs.Ready {
				readyContainers++
			}
		}
		owner := ""
		if len(p.OwnerReferences) > 0 {
			owner = p.OwnerReferences[0].Kind
		}
		images := make([]string, 0, len(p.Spec.Containers))
		for _, c := range p.Spec.Containers {
			images = append(images, c.Image)
		}
		snap.Pods = append(snap.Pods, PodInfo{
			Namespace:       p.Namespace,
			Name:            p.Name,
			Node:            p.Spec.NodeName,
			Phase:           string(p.Status.Phase),
			Terminating:     p.DeletionTimestamp != nil,
			Restarts:        restarts,
			ReadyContainers: readyContainers,
			TotalContainers: len(p.Spec.Containers),
			Owner:           owner,
			Images:          images,
			CreatedAt:       p.CreationTimestamp.Time,
			// Phase is the kubelet's last report. If the pod's node has stopped
			// reporting (NotReady), that phase is frozen and cannot be trusted —
			// render it as unknown rather than repeating a possibly dead pod's
			// last words (ADR-002).
			Stale: onKnownNode && !ready,
		})
	}

	sort.Slice(snap.Nodes, func(i, j int) bool { return snap.Nodes[i].Name < snap.Nodes[j].Name })
	sort.Slice(snap.Pods, func(i, j int) bool {
		if snap.Pods[i].Namespace != snap.Pods[j].Namespace {
			return snap.Pods[i].Namespace < snap.Pods[j].Namespace
		}
		return snap.Pods[i].Name < snap.Pods[j].Name
	})
	return snap, nil
}

func nodeRoles(n *corev1.Node) []string {
	var roles []string
	for label := range n.Labels {
		if r, ok := strings.CutPrefix(label, "node-role.kubernetes.io/"); ok && r != "" {
			roles = append(roles, r)
		}
	}
	sort.Strings(roles)
	return roles
}
