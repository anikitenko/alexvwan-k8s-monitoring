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

type GetK8sStateFulSetsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sStateFulSetsHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID              string    `json:"id"`
		Name            string    `json:"name"`
		DesiredReplicas int32     `json:"desired_replicas"`
		CurrentReplicas int32     `json:"current_replicas"`
		Age             string    `json:"age"`
		Labels          []string  `json:"labels"`
		Selectors       []string  `json:"selectors"`
		Condition       Condition `json:"condition"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sStateFulSetsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	statefulsets, err := clientset.AppsV1().StatefulSets(h.NS).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get statefulsets when calling GetK8sStateFulSetsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, ss := range statefulsets.Items {
		name := ss.GenerateName + ss.Name
		age := ss.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range ss.GetLabels() {
			labels = append(labels, key+":"+value)
		}
		selectors := make([]string, 0)
		for key, value := range ss.Spec.Selector.MatchLabels {
			selectors = append(selectors, key+":"+value)
		}

		var latestCondition v1.StatefulSetCondition
		latestConditionOK := true
		if len(ss.Status.Conditions) > 0 {
			latestCondition = ss.Status.Conditions[0]
			for _, condition := range ss.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
			latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
		}
		latestConditionMessage := latestCondition.Message
		if latestCondition.Message == "" && latestConditionOK {
			latestConditionMessage = "Stateful Set is OK"
		}

		response = append(response, Response{
			ID:              GenerateRandomString(10),
			Name:            name,
			DesiredReplicas: *ss.Spec.Replicas,
			CurrentReplicas: ss.Status.CurrentReplicas,
			Age:             ElapsedTimeShort(age.Time),
			Labels:          labels,
			Selectors:       selectors,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
