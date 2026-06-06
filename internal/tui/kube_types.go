package tui

import (
	"fmt"
	"time"
)

type kubeResourceKind int

const (
	kubeResPods kubeResourceKind = iota
	kubeResDeployments
	kubeResServices
	kubeResIngress
	kubeResNamespaces
	kubeResNodes
	kubeResEvents
	kubeResContexts
)

var kubeResourceOrder = []kubeResourceKind{
	kubeResPods,
	kubeResDeployments,
	kubeResServices,
	kubeResIngress,
	kubeResNamespaces,
	kubeResNodes,
	kubeResEvents,
	kubeResContexts,
}

func kubeResourceLabel(kind kubeResourceKind) string {
	switch kind {
	case kubeResPods:
		return "Pods"
	case kubeResDeployments:
		return "Deployments"
	case kubeResServices:
		return "Services"
	case kubeResIngress:
		return "Ingress"
	case kubeResNamespaces:
		return "Namespaces"
	case kubeResNodes:
		return "Nodes"
	case kubeResEvents:
		return "Events"
	case kubeResContexts:
		return "Contexts"
	default:
		return "Kubernetes"
	}
}

func kubeResourceSingular(kind kubeResourceKind) string {
	switch kind {
	case kubeResPods:
		return "pod"
	case kubeResDeployments:
		return "deployment"
	case kubeResServices:
		return "service"
	case kubeResIngress:
		return "ingress"
	case kubeResNamespaces:
		return "namespace"
	case kubeResNodes:
		return "node"
	case kubeResEvents:
		return "event"
	case kubeResContexts:
		return "context"
	default:
		return "resource"
	}
}

func (k kubeResourceKind) next() kubeResourceKind {
	for i, item := range kubeResourceOrder {
		if item == k {
			return kubeResourceOrder[(i+1)%len(kubeResourceOrder)]
		}
	}
	return kubeResPods
}

func (k kubeResourceKind) prev() kubeResourceKind {
	for i, item := range kubeResourceOrder {
		if item == k {
			return kubeResourceOrder[(i+len(kubeResourceOrder)-1)%len(kubeResourceOrder)]
		}
	}
	return kubeResPods
}

type kubeRow struct {
	Resource  string
	Namespace string
	Name      string
	ColA      string
	ColB      string
	ColC      string
	ColD      string
	Status    string
	Detail    string
	Restarts  int
	Age       string
	Current   bool
}

func formatAge(created time.Time) string {
	if created.IsZero() {
		return "-"
	}
	delta := time.Since(created)
	switch {
	case delta < time.Minute:
		return fmt.Sprintf("%ds", int(delta.Seconds()))
	case delta < time.Hour:
		return fmt.Sprintf("%dm", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh", int(delta.Hours()))
	default:
		return fmt.Sprintf("%dd", int(delta.Hours()/24))
	}
}
