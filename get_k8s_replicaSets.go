package main

import (
	"context"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strconv"
)

type GetK8sReplicaSetsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sReplicaSetsHandler) ServeHTTP(c echo.Context) error {
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

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sReplicaSetsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	replicaSetList, err := clientset.AppsV1().ReplicaSets(h.NS).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get ReplicaSets when calling GetK8sReplicaSetsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, rs := range replicaSetList.Items {
		name := rs.GenerateName + rs.Name

		age := rs.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range rs.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		containers := make([]string, 0)
		for _, container := range rs.Spec.Template.Spec.Containers {
			containers = append(containers, container.Name)
		}

		matchLabels := make([]string, 0)
		for key, value := range rs.Spec.Selector.MatchLabels {
			matchLabels = append(matchLabels, key+":"+value)
		}

		var latestCondition v1.ReplicaSetCondition
		latestConditionOK := true
		if len(rs.Status.Conditions) > 0 {
			latestCondition = rs.Status.Conditions[0]
			for _, condition := range rs.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
			latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
		}
		latestConditionMessage := latestCondition.Message
		if latestCondition.Message == "" && latestConditionOK {
			latestConditionMessage = "Replica Set is OK"
		}

		response = append(response, Response{
			ID:            GenerateRandomString(10),
			Name:          name,
			StatusActual:  rs.Status.ReadyReplicas,
			StatusDesired: rs.Status.Replicas,
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
