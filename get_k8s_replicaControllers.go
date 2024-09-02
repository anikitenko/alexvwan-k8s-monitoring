package main

import (
	"context"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strconv"
)

type GetK8sReplicaControllersHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sReplicaControllersHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		StatusActual  int32     `json:"ready_actual"`
		StatusDesired int32     `json:"ready_desired"`
		Containers    []string  `json:"containers"`
		Selector      []string  `json:"selectors"`
		Age           string    `json:"age"`
		Labels        []string  `json:"labels"`
		Condition     Condition `json:"condition"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sCronJobsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	replicaControllers, err := clientset.CoreV1().ReplicationControllers(h.NS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get replica controllers: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, rc := range replicaControllers.Items {
		name := rc.GenerateName + rc.Name

		age := rc.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range rc.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		containers := make([]string, 0)
		for _, container := range rc.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
		}

		matchLabels := make([]string, 0)
		for key, value := range rc.Spec.Selector {
			matchLabels = append(matchLabels, key+":"+value)
		}

		var latestCondition v1.ReplicationControllerCondition
		latestConditionOK := true
		if len(rc.Status.Conditions) > 0 {
			latestCondition = rc.Status.Conditions[0]
			for _, condition := range rc.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
			latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
		}
		latestConditionMessage := latestCondition.Message
		if latestCondition.Message == "" && latestConditionOK {
			latestConditionMessage = "Replication Controller is OK"
		}

		response = append(response, Response{
			ID:            GenerateRandomString(10),
			Name:          name,
			StatusActual:  rc.Status.ReadyReplicas,
			StatusDesired: rc.Status.Replicas,
			Containers:    containers,
			Selector:      matchLabels,
			Age:           ElapsedTimeShort(age.Time),
			Labels:        labels,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
