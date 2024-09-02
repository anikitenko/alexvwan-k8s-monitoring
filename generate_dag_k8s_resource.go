package main

import (
	"context"
	"fmt"
	v1 "k8s.io/api/apps/v1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"strings"
)

type KubernetesResource struct {
	ResourceName      string
	ResourceType      string
	ResourceNamespace string
}

type StatusInterface interface {
	OK() bool
}

type DeploymentStatus struct {
	Ok                 bool     `json:"ok"`
	Namespace          string   `json:"namespace"`
	Created            string   `json:"created"`
	DeploymentStrategy string   `json:"deployment_strategy"`
	Selectors          []string `json:"selectors"`
	Message            string   `json:"message"`
}

func (s DeploymentStatus) OK() bool {
	return s.Ok
}

type ReplicaSetStatus struct {
	Ok              bool   `json:"ok"`
	Namespace       string `json:"namespace"`
	Created         string `json:"created"`
	CurrentReplicas string `json:"current_replicas"`
	DesiredReplicas string `json:"desired_replicas"`
	Message         string `json:"message"`
}

func (s ReplicaSetStatus) OK() bool {
	return s.Ok
}

type PodStatus struct {
	Ok             bool     `json:"ok"`
	Namespace      string   `json:"namespace"`
	Created        string   `json:"created"`
	ServiceAccount string   `json:"service_account"`
	Node           string   `json:"node"`
	ControlledBy   []string `json:"controlled_by"`
	Message        string   `json:"message"`
}

func (s PodStatus) OK() bool {
	return s.Ok
}

type ServiceStatus struct {
	Ok              bool   `json:"ok"`
	Namespace       string `json:"namespace"`
	Created         string `json:"created"`
	SessionAffinity string `json:"session_affinity"`
	Message         string `json:"message"`
}

func (s ServiceStatus) OK() bool {
	return s.Ok
}

type ServiceAccountStatus struct {
	Ok        bool   `json:"ok"`
	Namespace string `json:"namespace"`
	Created   string `json:"created"`
	Message   string `json:"message"`
}

func (s ServiceAccountStatus) OK() bool {
	return s.Ok
}

type IngressStatus struct {
	Ok             bool   `json:"ok"`
	Namespace      string `json:"namespace"`
	Created        string `json:"created"`
	DefaultBackend string `json:"default_backend"`
	Message        string `json:"message"`
}

func (s IngressStatus) OK() bool {
	return s.Ok
}

type PersistentVolumeClaimStatus struct {
	Ok        bool   `json:"ok"`
	Namespace string `json:"namespace"`
	Created   string `json:"created"`
	Message   string `json:"message"`
}

func (s PersistentVolumeClaimStatus) OK() bool {
	return s.Ok
}

type SecretStatus struct {
	Ok        bool   `json:"ok"`
	Namespace string `json:"namespace"`
	Created   string `json:"created"`
	Message   string `json:"message"`
}

func (s SecretStatus) OK() bool {
	return s.Ok
}

type ConfigMapStatus struct {
	Ok        bool   `json:"ok"`
	Namespace string `json:"namespace"`
	Created   string `json:"created"`
	Message   string `json:"message"`
}

func (s ConfigMapStatus) OK() bool {
	return s.Ok
}

type HPAStatus struct {
	Ok        bool   `json:"ok"`
	Namespace string `json:"namespace"`
	Created   string `json:"created"`
	Message   string `json:"message"`
}

func (s HPAStatus) OK() bool {
	return s.Ok
}

type ResourceViewerNodes struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Color       string      `json:"color"`
	Status      interface{} `json:"status"`
}
type ResourceViewerEdges struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type Graph struct {
	Nodes []ResourceViewerNodes `json:"nodes"`
	Edges []ResourceViewerEdges `json:"edges"`
}

type DAGTraverser struct {
	Visited map[string]bool
	Graph   *Graph
}

func (dt *DAGTraverser) CreateGraph() *Graph {
	// Assuming nodes and edges are created inside this method
	nodes := make([]ResourceViewerNodes, 0)
	edges := make([]ResourceViewerEdges, 0)

	return &Graph{
		Nodes: nodes,
		Edges: edges,
	}
}

func (dt *Graph) addNode(kr KubernetesResource, status interface{}) error {
	node := ResourceViewerNodes{
		ID:          kr.ResourceName + "-" + kr.ResourceType,
		Title:       kr.ResourceName,
		Description: kr.ResourceType,
		Status:      status,
		Color:       "#63f828",
	}
	if statusObj, ok := status.(StatusInterface); ok {
		if !statusObj.OK() {
			node.Color = "#f10909"
		}
	} else {
		return fmt.Errorf("failed to assert StatusInterface: %v", ok)
	}
	dt.Nodes = append(dt.Nodes, node)
	return nil
}

func (dt *Graph) addEdge(resource, relatedResource string) {
	dt.Edges = append(dt.Edges, ResourceViewerEdges{
		Source: resource,
		Target: relatedResource,
	})
}

func (dt *DAGTraverser) GenerateDAGForResource(clientSet *kubernetes.Clientset, kr KubernetesResource) error {
	if _, visited := dt.Visited[kr.ResourceType+"_"+kr.ResourceName]; visited {
		return nil
	}

	var relatedResources []KubernetesResource
	var status interface{}
	var err error

	switch kr.ResourceType {
	case "Deployment":
		relatedResources, status, err = dt.getRelatedResourcesDeployments(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for Deployment: %v", err)
		}
	case "ReplicaSet":
		relatedResources, status, err = dt.getRelatedResourcesReplicaSet(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for ReplicaSet: %v", err)
		}
	case "Pod":
		relatedResources, status, err = dt.getRelatedResourcesPod(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for Pod: %v", err)
		}
	case "Service":
		relatedResources, status, err = dt.getRelatedResourcesService(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for Service: %v", err)
		}
	case "ServiceAccount":
		relatedResources, status, err = dt.getRelatedResourcesServiceAccount(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for ServiceAccount: %v", err)
		}
	case "Ingress":
		relatedResources, status, err = dt.getRelatedResourcesIngress(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for Ingress: %v", err)
		}
	case "PersistentVolumeClaim":
		relatedResources, status, err = dt.getRelatedResourcesPersistentVolumeClaim(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for PersistentVolumeClaim: %v", err)
		}
	case "ConfigMap":
		relatedResources, status, err = dt.getRelatedResourcesConfigMap(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for ConfigMap: %v", err)
		}
	case "Secret":
		relatedResources, status, err = dt.getRelatedResourcesSecret(clientSet, kr)
		if err != nil {
			return fmt.Errorf("unable to get related resources for Secret: %v", err)
		}
	default:
		return nil
	}

	hpaRelatedResources, err := dt.getHPAForController(clientSet, kr)
	if err != nil {
		return fmt.Errorf("unable to get related resources for Service: %v", err)
	}

	relatedResources = append(relatedResources, hpaRelatedResources...)

	if err := dt.Graph.addNode(kr, status); err != nil {
		return fmt.Errorf("failed to add node to graph: %v", err)
	}
	for _, relatedResource := range relatedResources {
		dt.Graph.addEdge(kr.ResourceName+"-"+kr.ResourceType, relatedResource.ResourceName+"-"+relatedResource.ResourceType)
	}

	// Mark the resource as visited.
	dt.Visited[kr.ResourceType+"_"+kr.ResourceName] = true

	// Recurse for child resources/nodes
	for _, childResource := range relatedResources {
		err := dt.GenerateDAGForResource(clientSet, childResource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dt *DAGTraverser) getRelatedResourcesDeployments(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status DeploymentStatus, err error) {
	deployment, err := clientSet.AppsV1().Deployments(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, DeploymentStatus{}, fmt.Errorf("unable to get deployment: %v", err)
	}
	var latestCondition v1.DeploymentCondition
	latestConditionOK := true
	if len(deployment.Status.Conditions) > 0 {
		latestCondition = deployment.Status.Conditions[0]
		for _, condition := range deployment.Status.Conditions {
			if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = condition
			}
		}
		latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
	}
	latestConditionMessage := latestCondition.Message
	if latestCondition.Message == "" && latestConditionOK {
		latestConditionMessage = "Deployment is OK"
	}
	deploymentSelectors := deployment.Spec.Selector.MatchLabels
	var deploymentSelectorsFormatted []string
	for key, value := range deploymentSelectors {
		deploymentSelectorsFormatted = append(deploymentSelectorsFormatted, fmt.Sprintf("%s:%s", key, value))
	}
	status = DeploymentStatus{
		Ok:                 latestConditionOK,
		Namespace:          deployment.Namespace,
		Created:            ElapsedTimeShort(deployment.CreationTimestamp.Time),
		DeploymentStrategy: string(deployment.Spec.Strategy.Type),
		Selectors:          deploymentSelectorsFormatted,
		Message:            latestConditionMessage,
	}

	replicaSets, err := clientSet.AppsV1().ReplicaSets(kr.ResourceNamespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, DeploymentStatus{}, fmt.Errorf("unable to list replica sets: %v", err)
	}

	replicaSetVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "ReplicaSet" {
			replicaSetVisited = true
		}
	}

	if !replicaSetVisited {
		for _, replica := range replicaSets.Items {
			for _, ref := range replica.ObjectMeta.OwnerReferences {
				if ref.Name == kr.ResourceName && *replica.Spec.Replicas > 0 {
					relatedResources = append(relatedResources, KubernetesResource{
						ResourceName:      replica.Name,
						ResourceType:      "ReplicaSet",
						ResourceNamespace: replica.Namespace,
					})
				}
			}
		}
	}

	if deployment.Spec.Template.Spec.ServiceAccountName != "" {
		serviceAccount, err := clientSet.CoreV1().ServiceAccounts(kr.ResourceNamespace).Get(context.TODO(), deployment.Spec.Template.Spec.ServiceAccountName, metav1.GetOptions{})
		if err != nil {
			return nil, DeploymentStatus{}, fmt.Errorf("unable to get service account: %v", err)
		}
		relatedResources = append(relatedResources, KubernetesResource{
			ResourceName:      serviceAccount.Name,
			ResourceType:      "ServiceAccount",
			ResourceNamespace: serviceAccount.Namespace,
		})
	}

	services, err := clientSet.CoreV1().Services(deployment.Namespace).List(context.TODO(), metav1.ListOptions{})
	for _, service := range services.Items {
		selector := labels.SelectorFromSet(service.Spec.Selector)
		if selector.Matches(labels.Set(deployment.Labels)) {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      service.Name,
				ResourceType:      "Service",
				ResourceNamespace: service.Namespace,
			})
		}
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesReplicaSet(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status ReplicaSetStatus, err error) {
	replicaSet, err := clientSet.AppsV1().ReplicaSets(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, ReplicaSetStatus{}, fmt.Errorf("unable to get replica set: %v", err)
	}
	var latestCondition v1.ReplicaSetCondition
	latestConditionOK := true
	if len(replicaSet.Status.Conditions) > 0 {
		latestCondition = replicaSet.Status.Conditions[0]
		for _, condition := range replicaSet.Status.Conditions {
			if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = condition
			}
		}
		latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
	}
	latestConditionMessage := latestCondition.Message
	if latestCondition.Message == "" && latestConditionOK {
		latestConditionMessage = "ReplicaSet is OK"
	}
	status = ReplicaSetStatus{
		Ok:              latestConditionOK,
		Namespace:       replicaSet.Namespace,
		Created:         ElapsedTimeShort(replicaSet.CreationTimestamp.Time),
		CurrentReplicas: strconv.Itoa(int(replicaSet.Status.Replicas)),
		DesiredReplicas: strconv.Itoa(int(replicaSet.Status.Replicas)),
		Message:         latestConditionMessage,
	}

	deploymentVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Deployment" {
			deploymentVisited = true
		}
	}
	if !deploymentVisited {
		for _, owner := range replicaSet.GetObjectMeta().GetOwnerReferences() {
			if owner.Kind == "Deployment" {
				deployment, err := clientSet.AppsV1().Deployments(kr.ResourceNamespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
				if err != nil {
					return nil, ReplicaSetStatus{}, fmt.Errorf("unable to get deployment: %v", err)
				}
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      deployment.Name,
					ResourceType:      "Deployment",
					ResourceNamespace: deployment.Namespace,
				})
			}
		}
	}

	pods, err := clientSet.CoreV1().Pods(kr.ResourceNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, ReplicaSetStatus{}, fmt.Errorf("unable to list pods: %v", err)
	}

	for _, pod := range pods.Items {
		for _, ref := range pod.ObjectMeta.OwnerReferences {
			if ref.Name == kr.ResourceName {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      pod.Name,
					ResourceType:      "Pod",
					ResourceNamespace: pod.Namespace,
				})
			}
		}
	}

	if replicaSet.Spec.Template.Spec.ServiceAccountName != "" {
		serviceAccount, err := clientSet.CoreV1().ServiceAccounts(kr.ResourceNamespace).Get(context.TODO(), replicaSet.Spec.Template.Spec.ServiceAccountName, metav1.GetOptions{})
		if err != nil {
			return nil, ReplicaSetStatus{}, fmt.Errorf("unable to get service account: %v", err)
		}
		relatedResources = append(relatedResources, KubernetesResource{
			ResourceName:      serviceAccount.Name,
			ResourceType:      "ServiceAccount",
			ResourceNamespace: serviceAccount.Namespace,
		})
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesPod(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status PodStatus, err error) {
	pod, err := clientSet.CoreV1().Pods(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, PodStatus{}, fmt.Errorf("unable to get pod: %v", err)
	}
	var latestCondition v1core.PodCondition
	latestConditionOK := true
	if len(pod.Status.Conditions) > 0 {
		latestCondition = pod.Status.Conditions[0]
		for _, condition := range pod.Status.Conditions {
			if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = condition
			}
		}
		latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
	}
	latestConditionMessage := latestCondition.Message
	if latestCondition.Message == "" && latestConditionOK {
		latestConditionMessage = "Pod is OK"
	}
	controlledBy := make([]string, 0)
	for _, control := range pod.ObjectMeta.OwnerReferences {
		controlledBy = append(controlledBy, control.Name)
	}
	status = PodStatus{
		Ok:             latestConditionOK,
		Namespace:      pod.Namespace,
		Created:        ElapsedTimeShort(pod.CreationTimestamp.Time),
		ServiceAccount: pod.Spec.ServiceAccountName,
		Node:           pod.Spec.NodeName,
		ControlledBy:   controlledBy,
		Message:        latestConditionMessage,
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      volume.PersistentVolumeClaim.ClaimName,
				ResourceType:      "PersistentVolumeClaim",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
		if volume.Secret != nil {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      volume.Secret.SecretName,
				ResourceType:      "Secret",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
		if volume.ConfigMap != nil {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      volume.ConfigMap.Name,
				ResourceType:      "ConfigMap",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
	}

	for _, container := range pod.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      envFrom.ConfigMapRef.Name,
					ResourceType:      "ConfigMap",
					ResourceNamespace: kr.ResourceNamespace,
				})
			}
			if envFrom.SecretRef != nil {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      envFrom.SecretRef.Name,
					ResourceType:      "Secret",
					ResourceNamespace: kr.ResourceNamespace,
				})
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil {
				if env.ValueFrom.ConfigMapKeyRef != nil {
					relatedResources = append(relatedResources, KubernetesResource{
						ResourceName:      env.ValueFrom.ConfigMapKeyRef.Name,
						ResourceType:      "ConfigMap",
						ResourceNamespace: kr.ResourceNamespace,
					})
				}
				if env.ValueFrom.SecretKeyRef != nil {
					relatedResources = append(relatedResources, KubernetesResource{
						ResourceName:      env.ValueFrom.SecretKeyRef.Name,
						ResourceType:      "Secret",
						ResourceNamespace: kr.ResourceNamespace,
					})
				}
			}
		}
	}
	for _, container := range pod.Spec.InitContainers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      envFrom.ConfigMapRef.Name,
					ResourceType:      "ConfigMap",
					ResourceNamespace: kr.ResourceNamespace,
				})
			}
			if envFrom.SecretRef != nil {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      envFrom.SecretRef.Name,
					ResourceType:      "Secret",
					ResourceNamespace: kr.ResourceNamespace,
				})
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil {
				if env.ValueFrom.ConfigMapKeyRef != nil {
					relatedResources = append(relatedResources, KubernetesResource{
						ResourceName:      env.ValueFrom.ConfigMapKeyRef.Name,
						ResourceType:      "ConfigMap",
						ResourceNamespace: kr.ResourceNamespace,
					})
				}
				if env.ValueFrom.SecretKeyRef != nil {
					relatedResources = append(relatedResources, KubernetesResource{
						ResourceName:      env.ValueFrom.SecretKeyRef.Name,
						ResourceType:      "Secret",
						ResourceNamespace: kr.ResourceNamespace,
					})
				}
			}
		}
	}

	if pod.Spec.ServiceAccountName != "" {
		relatedResources = append(relatedResources, KubernetesResource{
			ResourceName:      pod.Spec.ServiceAccountName,
			ResourceType:      "ServiceAccount",
			ResourceNamespace: kr.ResourceNamespace,
		})
	}

	svcList, err := clientSet.CoreV1().Services(pod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, PodStatus{}, fmt.Errorf("unable to get services: %v", err)
	}

	for _, svc := range svcList.Items {
		if svc.Spec.Selector == nil {
			continue
		}
		selector := labels.SelectorFromSet(svc.Spec.Selector)
		if selector.Matches(labels.Set(pod.Labels)) {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      svc.Name,
				ResourceType:      "Service",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
	}

	pdbs, err := clientSet.PolicyV1().PodDisruptionBudgets(kr.ResourceNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, PodStatus{}, fmt.Errorf("unable to get PDBs: %v", err)
	}

	for _, pdb := range pdbs.Items {
		selector, _ := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
		if selector.Matches(labels.Set(pod.Labels)) {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      pdb.Name,
				ResourceType:      "PodDisruptionBudget",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
	}

	netpols, err := clientSet.NetworkingV1().NetworkPolicies(kr.ResourceNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, PodStatus{}, fmt.Errorf("unable to get NetworkPolicies: %v", err)
	}
	// Filter NetworkPolicies that select the Pod
	for _, netpol := range netpols.Items {
		podSelector, _ := metav1.LabelSelectorAsSelector(&netpol.Spec.PodSelector)
		if podSelector.Matches(labels.Set(pod.Labels)) {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      netpol.Name,
				ResourceType:      "NetworkPolicy",
				ResourceNamespace: kr.ResourceNamespace,
			})
		}
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesService(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status ServiceStatus, err error) {
	service, err := clientSet.CoreV1().Services(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, ServiceStatus{}, fmt.Errorf("unable to get service: %v", err)
	}
	var latestCondition metav1.Condition
	latestConditionOK := true
	if len(service.Status.Conditions) > 0 {
		latestCondition = service.Status.Conditions[0]
		for _, condition := range service.Status.Conditions {
			if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = condition
			}
		}
		latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
	}
	latestConditionMessage := latestCondition.Message
	if latestCondition.Message == "" && latestConditionOK {
		latestConditionMessage = "Service is OK"
	}
	status = ServiceStatus{
		Ok:              latestConditionOK,
		Namespace:       service.Namespace,
		Created:         ElapsedTimeShort(service.CreationTimestamp.Time),
		SessionAffinity: string(service.Spec.SessionAffinity),
		Message:         latestConditionMessage,
	}

	ingressVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Ingress" {
			ingressVisited = true
		}
	}

	if !ingressVisited {
		ingresses, err := clientSet.NetworkingV1().Ingresses(kr.ResourceNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, ServiceStatus{}, fmt.Errorf("unable to get ingress: %v", err)
		}
		for _, ingress := range ingresses.Items {
			for _, rule := range ingress.Spec.Rules {
				for _, path := range rule.IngressRuleValue.HTTP.Paths {
					if path.Backend.Service.Name == kr.ResourceName {
						relatedResources = append(relatedResources, KubernetesResource{
							ResourceName:      ingress.Name,
							ResourceType:      "Ingress",
							ResourceNamespace: ingress.Namespace,
						})
						break
					}
				}
			}
		}
	}

	podVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Pod" {
			podVisited = true
		}
	}

	if !podVisited {
		podList, err := clientSet.CoreV1().Pods(service.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.Set(service.Spec.Selector).AsSelectorPreValidated().String(),
		})

		if err != nil {
			return nil, ServiceStatus{}, fmt.Errorf("unable to get pods: %v", err)
		}

		for _, pod := range podList.Items {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      pod.Name,
				ResourceType:      "Pod",
				ResourceNamespace: pod.Namespace,
			})
		}
	}

	deploymentVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Deployment" {
			deploymentVisited = true
		}
	}

	if !deploymentVisited {
		deployments, err := clientSet.AppsV1().Deployments(kr.ResourceNamespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.Set(service.Spec.Selector).AsSelectorPreValidated().String(),
		})
		if err != nil {
			return nil, ServiceStatus{}, fmt.Errorf("unable to get deployments: %v", err)
		}
		for _, deployment := range deployments.Items {
			selector := labels.SelectorFromSet(service.Spec.Selector)
			if selector.Matches(labels.Set(deployment.Labels)) {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      deployment.Name,
					ResourceType:      "Deployment",
					ResourceNamespace: deployment.Namespace,
				})
			}
		}
	}
	return
}

func (dt *DAGTraverser) getRelatedResourcesServiceAccount(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status ServiceAccountStatus, err error) {
	serviceAccount, err := clientSet.CoreV1().ServiceAccounts(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, ServiceAccountStatus{}, fmt.Errorf("unable to list service accounts: %v", err)
	}

	status = ServiceAccountStatus{
		Ok:        true,
		Namespace: serviceAccount.Namespace,
		Created:   ElapsedTimeShort(serviceAccount.CreationTimestamp.Time),
		Message:   "ServiceAccount is OK",
	}

	for _, secret := range serviceAccount.Secrets {
		relatedResources = append(relatedResources, KubernetesResource{
			ResourceName:      secret.Name,
			ResourceType:      "Secret",
			ResourceNamespace: secret.Namespace,
		})
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesIngress(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status IngressStatus, err error) {
	ingress, err := clientSet.NetworkingV1().Ingresses(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, IngressStatus{}, fmt.Errorf("unable to get ingress: %v", err)
	}

	status = IngressStatus{
		Ok:        true,
		Namespace: ingress.Namespace,
		Created:   ElapsedTimeShort(ingress.CreationTimestamp.Time),
		Message:   "Ingress is OK",
	}
	if ingress.Spec.DefaultBackend != nil {
		status.DefaultBackend = ingress.Spec.DefaultBackend.Service.Name
	}

	serviceVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Service" {
			serviceVisited = true
		}
	}

	if !serviceVisited {
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				relatedResources = append(relatedResources, KubernetesResource{
					ResourceName:      path.Backend.Service.Name,
					ResourceType:      "Service",
					ResourceNamespace: ingress.Namespace,
				})
			}
		}
	}

	secretVisited := false
	for kr := range dt.Visited {
		kr = strings.Split(kr, "_")[0]
		if kr == "Secret" {
			secretVisited = true
		}
	}

	if !secretVisited {
		for _, tls := range ingress.Spec.TLS {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      tls.SecretName,
				ResourceType:      "Secret",
				ResourceNamespace: ingress.Namespace,
			})
		}
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesPersistentVolumeClaim(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status PersistentVolumeClaimStatus, err error) {
	pvc, err := clientSet.CoreV1().PersistentVolumeClaims(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, PersistentVolumeClaimStatus{}, fmt.Errorf("unable to get PersistentVolumeClaim: %v", err)
	}

	var latestCondition v1core.PersistentVolumeClaimCondition
	latestConditionOK := true
	if len(pvc.Status.Conditions) > 0 {
		latestCondition = pvc.Status.Conditions[0]
		for _, condition := range pvc.Status.Conditions {
			if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = condition
			}
		}
		latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
	}
	latestConditionMessage := latestCondition.Message
	if latestCondition.Message == "" && latestConditionOK {
		latestConditionMessage = "PersistentVolumeClaim is OK"
	}

	status = PersistentVolumeClaimStatus{
		Ok:        latestConditionOK,
		Namespace: pvc.Namespace,
		Created:   ElapsedTimeShort(pvc.CreationTimestamp.Time),
		Message:   latestConditionMessage,
	}

	return
}

func (dt *DAGTraverser) getRelatedResourcesConfigMap(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status ConfigMapStatus, err error) {
	configMap, err := clientSet.CoreV1().ConfigMaps(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		status.Ok = false
		status.Message = err.Error()
		err = nil
		return
	}
	status.Ok = true
	status.Message = "ConfigMap is OK"
	status.Namespace = configMap.Namespace
	status.Created = ElapsedTimeShort(configMap.CreationTimestamp.Time)
	return
}

func (dt *DAGTraverser) getRelatedResourcesSecret(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, status SecretStatus, err error) {
	secret, err := clientSet.CoreV1().Secrets(kr.ResourceNamespace).Get(context.TODO(), kr.ResourceName, metav1.GetOptions{})
	if err != nil {
		status = SecretStatus{
			Ok:      false,
			Message: err.Error(),
		}
		err = nil
		return
	}
	status = SecretStatus{
		Ok:        true,
		Message:   "Secret is OK",
		Namespace: secret.Namespace,
		Created:   ElapsedTimeShort(secret.CreationTimestamp.Time),
	}
	return
}

func (dt *DAGTraverser) getHPAForController(clientSet *kubernetes.Clientset, kr KubernetesResource) (relatedResources []KubernetesResource, err error) {
	hpas, err := clientSet.AutoscalingV1().HorizontalPodAutoscalers(kr.ResourceNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get HPAs: %v", err)
	}

	for _, hpa := range hpas.Items {
		// Compare the apiVersion, kind and name between hpa's reference and the target controller
		// Note that you might need to normalize API versions at first
		if hpa.Spec.ScaleTargetRef.Kind == kr.ResourceType &&
			hpa.Spec.ScaleTargetRef.Name == kr.ResourceName {
			relatedResources = append(relatedResources, KubernetesResource{
				ResourceName:      hpa.Name,
				ResourceType:      "HorizontalPodAutoscaler",
				ResourceNamespace: hpa.Namespace,
			})
		}
	}

	return
}
