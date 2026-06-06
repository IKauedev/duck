package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type kubeMeta struct {
	Namespace         string    `json:"namespace"`
	Name              string    `json:"name"`
	CreationTimestamp time.Time `json:"creationTimestamp"`
	DeletionTimestamp *string   `json:"deletionTimestamp"`
}

func parseKubeRows(kind kubeResourceKind, output string) ([]kubeRow, error) {
	output = normalizeCLIOutput(output)
	switch kind {
	case kubeResPods:
		return parseKubePods(output)
	case kubeResDeployments:
		return parseKubeDeployments(output)
	case kubeResServices:
		return parseKubeServices(output)
	case kubeResIngress:
		return parseKubeIngress(output)
	case kubeResNamespaces:
		return parseKubeNamespaces(output)
	case kubeResNodes:
		return parseKubeNodes(output)
	case kubeResEvents:
		return parseKubeEvents(output)
	case kubeResContexts:
		return parseKubeContexts(output)
	default:
		return nil, fmt.Errorf("recurso kubernetes nao suportado")
	}
}

func parseKubePods(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Status   struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready        bool `json:"ready"`
					RestartCount int  `json:"restartCount"`
					State        struct {
						Waiting *struct {
							Reason string `json:"reason"`
						} `json:"waiting"`
						Terminated *struct {
							Reason string `json:"reason"`
						} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		ready := 0
		total := len(item.Status.ContainerStatuses)
		restarts := 0
		detail := item.Status.Phase
		for _, status := range item.Status.ContainerStatuses {
			if status.Ready {
				ready++
			}
			restarts += status.RestartCount
			if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
				detail = status.State.Waiting.Reason
			}
			if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
				detail = status.State.Terminated.Reason
			}
		}
		if item.Metadata.DeletionTimestamp != nil {
			detail = "Terminating"
		}
		rows = append(rows, kubeRow{
			Resource:  "pod",
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			ColA:      fmt.Sprintf("%d/%d", ready, maxInt(total, 1)),
			Status:    item.Status.Phase,
			Detail:    detail,
			Restarts:  restarts,
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func parseKubeDeployments(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Spec     struct {
				Replicas *int `json:"replicas"`
			} `json:"spec"`
			Status struct {
				Replicas          int `json:"replicas"`
				ReadyReplicas     int `json:"readyReplicas"`
				UpdatedReplicas   int `json:"updatedReplicas"`
				AvailableReplicas int `json:"availableReplicas"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		desired := item.Status.Replicas
		if item.Spec.Replicas != nil {
			desired = *item.Spec.Replicas
		}
		rows = append(rows, kubeRow{
			Resource:  "deployment",
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			ColA:      fmt.Sprintf("%d/%d", item.Status.ReadyReplicas, desired),
			ColB:      fmt.Sprintf("%d", item.Status.UpdatedReplicas),
			ColC:      fmt.Sprintf("%d", item.Status.AvailableReplicas),
			Status:    deploymentStatus(item.Status.ReadyReplicas, desired),
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func deploymentStatus(ready, desired int) string {
	if desired == 0 {
		return "ScaledToZero"
	}
	if ready == desired {
		return "Available"
	}
	return "Progressing"
}

func parseKubeServices(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Spec struct {
				Type      string `json:"type"`
				ClusterIP string `json:"clusterIP"`
				Ports     []struct {
					Port       int    `json:"port"`
					TargetPort any    `json:"targetPort"`
					Protocol   string `json:"protocol"`
				} `json:"ports"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		ports := make([]string, 0, len(item.Spec.Ports))
		for _, port := range item.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
		}
		rows = append(rows, kubeRow{
			Resource:  "service",
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			ColA:      item.Spec.Type,
			ColB:      item.Spec.ClusterIP,
			ColC:      strings.Join(ports, ","),
			Status:    item.Spec.Type,
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func parseKubeIngress(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Spec     struct {
				IngressClassName *string `json:"ingressClassName"`
				Rules            []struct {
					Host string `json:"host"`
				} `json:"rules"`
			} `json:"spec"`
			Status struct {
				LoadBalancer struct {
					Ingress []struct {
						IP       string `json:"ip"`
						Hostname string `json:"hostname"`
					} `json:"ingress"`
				} `json:"loadBalancer"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		hosts := make([]string, 0, len(item.Spec.Rules))
		for _, rule := range item.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		address := "-"
		if len(item.Status.LoadBalancer.Ingress) > 0 {
			entry := item.Status.LoadBalancer.Ingress[0]
			if entry.IP != "" {
				address = entry.IP
			} else if entry.Hostname != "" {
				address = entry.Hostname
			}
		}
		class := "-"
		if item.Spec.IngressClassName != nil {
			class = *item.Spec.IngressClassName
		}
		rows = append(rows, kubeRow{
			Resource:  "ingress",
			Namespace: item.Metadata.Namespace,
			Name:      item.Metadata.Name,
			ColA:      class,
			ColB:      strings.Join(hosts, ","),
			ColC:      address,
			Status:    ingressStatus(address, hosts),
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func ingressStatus(address string, hosts []string) string {
	if address != "-" && len(hosts) > 0 {
		return "Ready"
	}
	if len(hosts) > 0 {
		return "Pending"
	}
	return "Unknown"
}

func parseKubeNamespaces(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Status   struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		rows = append(rows, kubeRow{
			Resource: "namespace",
			Name:     item.Metadata.Name,
			Status:   item.Status.Phase,
			Age:      formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func parseKubeNodes(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Status   struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
			Spec struct {
				Taints []struct {
					Effect string `json:"effect"`
				} `json:"taints"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for _, item := range list.Items {
		status := "NotReady"
		for _, cond := range item.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
				break
			}
		}
		rows = append(rows, kubeRow{
			Resource: "node",
			Name:     item.Metadata.Name,
			ColA:     nodeRoles(item.Metadata.Name),
			ColB:     item.Status.NodeInfo.KubeletVersion,
			Status:   status,
			Age:      formatAge(item.Metadata.CreationTimestamp),
		})
	}
	return rows, nil
}

func nodeRoles(name string) string {
	if strings.Contains(name, "control-plane") || strings.Contains(name, "master") {
		return "control-plane"
	}
	return "worker"
}

func parseKubeEvents(output string) ([]kubeRow, error) {
	var list struct {
		Items []struct {
			Metadata kubeMeta `json:"metadata"`
			Type     string   `json:"type"`
			Reason   string   `json:"reason"`
			Message  string   `json:"message"`
			InvolvedObject struct {
				Kind      string `json:"kind"`
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"involvedObject"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(list.Items))
	for i := len(list.Items) - 1; i >= 0; i-- {
		item := list.Items[i]
		rows = append(rows, kubeRow{
			Resource:  "event",
			Namespace: item.InvolvedObject.Namespace,
			Name:      item.InvolvedObject.Kind + "/" + item.InvolvedObject.Name,
			ColA:      item.Type,
			ColB:      item.Reason,
			ColC:      truncate(item.Message, 48),
			Status:    item.Type,
			Detail:    item.Reason,
			Age:       formatAge(item.Metadata.CreationTimestamp),
		})
		if len(rows) >= 80 {
			break
		}
	}
	return rows, nil
}

func parseKubeContexts(output string) ([]kubeRow, error) {
	lines := splitLines(output)
	current, _ := parseKubeContextsFromConfig(output)
	if len(lines) > 0 && !strings.Contains(lines[0], "{") {
		rows := make([]kubeRow, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			marker := ""
			isCurrent := line == current
			if isCurrent {
				marker = "*"
			}
			rows = append(rows, kubeRow{
				Resource: "context",
				Name:     line,
				ColA:     marker,
				Status:   "context",
				Current:  isCurrent,
			})
		}
		return rows, nil
	}

	var view struct {
		Contexts []struct {
			Name    string `json:"name"`
			Context struct {
				Cluster string `json:"cluster"`
			} `json:"context"`
		} `json:"contexts"`
		CurrentContext string `json:"current-context"`
	}
	if err := json.Unmarshal([]byte(output), &view); err != nil {
		return nil, err
	}
	rows := make([]kubeRow, 0, len(view.Contexts))
	for _, ctx := range view.Contexts {
		rows = append(rows, kubeRow{
			Resource: "context",
			Name:     ctx.Name,
			ColA:     ctx.Context.Cluster,
			Status:   "context",
			Current:  ctx.Name == view.CurrentContext,
		})
	}
	return rows, nil
}

func parseKubeContextsFromConfig(output string) (string, error) {
	var view struct {
		CurrentContext string `json:"current-context"`
	}
	if err := json.Unmarshal([]byte(output), &view); err != nil {
		return "", err
	}
	return view.CurrentContext, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

