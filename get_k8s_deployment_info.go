package main

import (
	"context"
	"encoding/json"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

type GetK8sDeploymentInfoHandler struct {
	ID         string
	Name       string
	NS         string
	Deployment string
}

func (h *GetK8sDeploymentInfoHandler) ServeHTTP(c echo.Context) error {
	type Configuration struct {
		DS       string `json:"ds"`
		Replicas int32  `json:"replicas"`
	}
	type Status struct {
		AvailableReplicas   int32 `json:"available_replicas"`
		ReadyReplicas       int32 `json:"ready_replicas"`
		TotalReplicas       int32 `json:"total_replicas"`
		UnavailableReplicas int32 `json:"unavailable_replicas"`
		UpdatedReplicas     int32 `json:"updated_replicas"`
	}
	type Pod struct {
		Name         string `json:"name"`
		Ready        int    `json:"ready"`
		ReadyDesired int    `json:"ready_desired"`
		Phase        string `json:"phase"`
		Status       string `json:"status"`
		Restarts     int32  `json:"restarts"`
		Node         string `json:"node"`
		Age          string `json:"age"`
	}
	type PodTemplateEnvironment struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Source string `json:"source"`
	}
	type PodTemplateVolume struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Propagation string `json:"propagation"`
	}
	type PodTemplatePort struct {
		Port     int32  `json:"port"`
		Protocol string `json:"protocol"`
	}
	type PodTemplate struct {
		ContainerName string                   `json:"container_name"`
		Image         string                   `json:"image"`
		Ports         []PodTemplatePort        `json:"ports"`
		Environment   []PodTemplateEnvironment `json:"environment"`
		Volume        []PodTemplateVolume      `json:"volume"`
	}
	type Volume struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
	}
	type Condition struct {
		Type           string `json:"type"`
		Reason         string `json:"reason"`
		Status         string `json:"status"`
		Message        string `json:"message"`
		LastUpdate     string `json:"last_update"`
		LastTransition string `json:"last_transition"`
	}
	type Event struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	type Summary struct {
		Configuration Configuration `json:"configuration"`
		Status        Status        `json:"status"`
		PodSelectors  []string      `json:"pod_selectors"`
		Pods          []Pod         `json:"pods"`
		Template      []PodTemplate `json:"template"`
		Volumes       []Volume      `json:"volumes"`
		Conditions    []Condition   `json:"conditions"`
		Events        []Event       `json:"events"`
	}
	type MetadataCustom struct {
		Name      string `json:"name"`
		Operation string `json:"operation"`
		Updated   string `json:"updated"`
		Fields    string `json:"fields"`
	}
	type Metadata struct {
		Age         string           `json:"age"`
		Labels      []string         `json:"labels"`
		Annotations []string         `json:"annotations"`
		Custom      []MetadataCustom `json:"custom"`
	}
	type Response struct {
		Summary  Summary  `json:"summary"`
		Metadata Metadata `json:"metadata"`
		Graph    *Graph   `json:"graph"`
		YAML     string   `json:"yaml"`
	}

	traverser := DAGTraverser{
		Visited: make(map[string]bool),
		Graph:   new(DAGTraverser).CreateGraph(),
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sDeploymentInfoHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	deploy, _ := clientset.AppsV1().Deployments(h.NS).Get(context.TODO(), h.Deployment, v1.GetOptions{})
	d, _ := json.Marshal(&deploy)
	var m map[string]interface{}
	if err := json.Unmarshal(d, &m); err != nil {
		logger.Fatalf("Error unmarshaling json when calling GetK8sDeploymentInfoHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	nestedMap, ok := m["metadata"].(map[string]interface{})
	if ok {
		delete(nestedMap, "managedFields")
	}
	m["apiVersion"] = "apps/v1"
	m["kind"] = "Deployment"

	yamlStr, err := yaml.Marshal(&m)
	if err != nil {
		logger.Fatalf("Error marshaling to YAML when calling GetK8sDeploymentInfoHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if err := traverser.GenerateDAGForResource(clientset, KubernetesResource{ResourceName: h.Deployment, ResourceType: "Deployment", ResourceNamespace: h.NS}); err != nil {
		logger.Warnf("Failed to get K8S resource traverse when calling GetK8sDeploymentInfoHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	summary := Summary{
		Configuration: Configuration{
			DS:       string(deploy.Spec.Strategy.Type),
			Replicas: *deploy.Spec.Replicas,
		},
		Status: Status{
			AvailableReplicas:   deploy.Status.AvailableReplicas,
			ReadyReplicas:       deploy.Status.ReadyReplicas,
			TotalReplicas:       deploy.Status.Replicas,
			UnavailableReplicas: deploy.Status.UnavailableReplicas,
			UpdatedReplicas:     deploy.Status.UpdatedReplicas,
		},
	}

	podSelectors := make([]string, 0)
	for i, s := range deploy.Spec.Selector.MatchLabels {
		podSelectors = append(podSelectors, i+":"+s)
	}

	summary.PodSelectors = podSelectors

	selector := v1.FormatLabelSelector(deploy.Spec.Selector)
	pods, err := clientset.CoreV1().Pods(h.NS).List(context.Background(), v1.ListOptions{LabelSelector: selector})
	if err != nil {
		logger.Warnf("Failed to get pods when calling GetK8sDeploymentInfoHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, pod := range pods.Items {
		var latestCondition v1core.PodCondition
		if len(pod.Status.Conditions) > 0 {
			latestCondition = pod.Status.Conditions[0]
			for _, condition := range pod.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
		}
		readyContainers := 0
		for _, status := range pod.Status.ContainerStatuses {
			if status.Ready {
				readyContainers++
			}
		}
		p := Pod{
			Name:         pod.Name,
			Ready:        readyContainers,
			ReadyDesired: len(pod.Spec.Containers),
			Phase:        string(pod.Status.Phase),
			Status:       string(latestCondition.Status),
			Restarts:     pod.Status.ContainerStatuses[0].RestartCount,
			Node:         pod.Spec.NodeName,
			Age:          ElapsedTimeShort(pod.CreationTimestamp.Time),
		}
		summary.Pods = append(summary.Pods, p)
	}

	for _, template := range deploy.Spec.Template.Spec.Containers {
		podTemplate := PodTemplate{
			ContainerName: template.Name,
			Image:         template.Image,
			Ports:         make([]PodTemplatePort, 0),
			Environment:   make([]PodTemplateEnvironment, 0),
			Volume:        make([]PodTemplateVolume, 0),
		}

		for _, port := range template.Ports {
			podTemplate.Ports = append(podTemplate.Ports, PodTemplatePort{
				Port:     port.ContainerPort,
				Protocol: string(port.Protocol),
			})
		}

		for _, env := range template.Env {
			pEnv := PodTemplateEnvironment{
				Name:  env.Name,
				Value: env.Value,
			}
			if env.ValueFrom != nil {
				if env.ValueFrom.FieldRef != nil {
					pEnv.Source = env.ValueFrom.FieldRef.FieldPath
				}
				if env.ValueFrom.ResourceFieldRef != nil {
					pEnv.Source = env.ValueFrom.ResourceFieldRef.Resource
				}
				if env.ValueFrom.ConfigMapKeyRef != nil {
					pEnv.Source = env.ValueFrom.ConfigMapKeyRef.Name
				}
				if env.ValueFrom.SecretKeyRef != nil {
					pEnv.Source = env.ValueFrom.SecretKeyRef.Name
				}
			}
			podTemplate.Environment = append(podTemplate.Environment, pEnv)
		}

		for _, volumeMount := range template.VolumeMounts {
			volume := PodTemplateVolume{
				Name: volumeMount.Name,
				Path: volumeMount.MountPath,
			}
			if volumeMount.MountPropagation != nil {
				volume.Propagation = string(*volumeMount.MountPropagation)
			} else {
				volume.Propagation = "None"
			}
			podTemplate.Volume = append(podTemplate.Volume, volume)
		}

		summary.Template = append(summary.Template, podTemplate)
	}

	for _, volume := range deploy.Spec.Template.Spec.Volumes {
		v := Volume{
			Name: volume.Name,
		}
		if volume.PersistentVolumeClaim != nil {
			v.Kind = "PersistentVolumeClaim"
			v.Description = volume.PersistentVolumeClaim.ClaimName
		}
		if volume.Secret != nil {
			v.Kind = "Secret"
			v.Description = volume.Secret.SecretName
		}
		if volume.ConfigMap != nil {
			v.Kind = "ConfigMap"
			v.Description = volume.ConfigMap.Name
		}
		if volume.EmptyDir != nil {
			v.Kind = "EmptyDir"
		}
		if volume.HostPath != nil {
			v.Kind = "HostPath"
			v.Description = volume.HostPath.Path
		}
		summary.Volumes = append(summary.Volumes, v)
	}

	for _, condition := range deploy.Status.Conditions {
		c := Condition{
			Type:           string(condition.Type),
			Reason:         condition.Reason,
			Status:         string(condition.Status),
			Message:        condition.Message,
			LastUpdate:     ElapsedTimeShort(condition.LastUpdateTime.Time),
			LastTransition: ElapsedTimeShort(condition.LastTransitionTime.Time),
		}
		summary.Conditions = append(summary.Conditions, c)
	}

	events, err := clientset.CoreV1().Events(h.NS).List(context.Background(), v1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get events when calling GetK8sDeploymentInfoHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	summary.Events = make([]Event, 0)
	for _, event := range events.Items {
		if event.InvolvedObject.Kind == "Deployment" && event.InvolvedObject.Name == h.Name {
			summary.Events = append(summary.Events, Event{
				Reason:  event.Reason,
				Message: event.Message,
			})
		}
	}

	deployLabels := make([]string, 0)
	for key, value := range deploy.GetLabels() {
		deployLabels = append(deployLabels, key+":"+value)
	}

	deployAnnotations := make([]string, 0)
	for key, value := range deploy.GetAnnotations() {
		deployAnnotations = append(deployAnnotations, key+":"+value)
	}

	deploymentMetadata := Metadata{
		Age:         ElapsedTimeShort(deploy.ObjectMeta.CreationTimestamp.Time),
		Labels:      deployLabels,
		Annotations: deployAnnotations,
		Custom:      make([]MetadataCustom, 0),
	}

	for _, mField := range deploy.ManagedFields {
		fieldsJson, err := json.Marshal(mField.FieldsV1)
		if err != nil {
			continue
		}
		deploymentMetadata.Custom = append(deploymentMetadata.Custom, MetadataCustom{
			Name:      mField.Manager,
			Operation: string(mField.Operation),
			Updated:   ElapsedTimeShort(mField.Time.Time),
			Fields:    string(fieldsJson),
		})
	}

	response := Response{
		Summary:  summary,
		Metadata: deploymentMetadata,
		Graph:    traverser.Graph,
		YAML:     string(yamlStr),
	}
	return c.JSON(http.StatusOK, response)
}
