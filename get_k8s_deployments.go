package main

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strconv"
)

type GetK8sDeploymentsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sDeploymentsHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		TotalReplicas int32     `json:"total_replicas"`
		Replicas      int32     `json:"replicas"`
		Age           string    `json:"age"`
		Containers    []string  `json:"containers"`
		Labels        []string  `json:"labels"`
		Selectors     []string  `json:"selectors"`
		Condition     Condition `json:"condition"`
	}
	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sDeploymentsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}
	deployments, err := clientset.AppsV1().Deployments(h.NS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get deployments when calling GetK8sDeploymentsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	var response = make([]Response, 0)
	for _, deployment := range deployments.Items {
		name := deployment.GenerateName + deployment.Name

		age := deployment.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range deployment.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		deploymentContainers := deployment.Spec.Template.Spec.Containers
		var deploymentContainersNames []string
		for _, container := range deploymentContainers {
			deploymentContainersNames = append(deploymentContainersNames, container.Name)
		}

		deploymentSelectors := deployment.Spec.Selector.MatchLabels
		var deploymentSelectorsFormatted []string
		for key, value := range deploymentSelectors {
			deploymentSelectorsFormatted = append(deploymentSelectorsFormatted, fmt.Sprintf("%s:%s", key, value))
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

		//for _, managedFields := range deployment.ManagedFields {
		//	var fields map[string]interface{}
		//	err := json.Unmarshal(managedFields.FieldsV1.Raw, &fields)
		//	if err != nil {
		//		fmt.Println(err)
		//		return nil
		//	}
		//	prettyJSON, _ := json.MarshalIndent(fields, "", "    ")
		//	fmt.Println("Fields:", string(prettyJSON))
		//}

		response = append(response, Response{
			ID:            GenerateRandomString(10),
			Name:          name,
			TotalReplicas: deployment.Status.Replicas,
			Replicas:      deployment.Status.AvailableReplicas,
			Age:           ElapsedTimeShort(age.Time),
			Containers:    deploymentContainersNames,
			Labels:        labels,
			Selectors:     deploymentSelectorsFormatted,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}
	return c.JSON(http.StatusOK, response)
}
