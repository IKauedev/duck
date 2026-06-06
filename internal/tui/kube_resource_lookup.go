package tui

import "strings"

func kubeResourceFromName(name string) kubeResourceKind {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "pod", "pods":
		return kubeResPods
	case "deployment", "deployments":
		return kubeResDeployments
	case "service", "services":
		return kubeResServices
	case "ingress":
		return kubeResIngress
	case "namespace", "namespaces":
		return kubeResNamespaces
	case "node", "nodes":
		return kubeResNodes
	case "event", "events":
		return kubeResEvents
	case "context", "contexts":
		return kubeResContexts
	default:
		return kubeResPods
	}
}
