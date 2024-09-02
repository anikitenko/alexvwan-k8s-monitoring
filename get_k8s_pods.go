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

type GetK8sPodsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sPodsHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		ReadyActual  int       `json:"ready_actual"`
		ReadyDesired int       `json:"ready_desired"`
		Phase        string    `json:"phase"`
		Status       string    `json:"status"`
		Restarts     int       `json:"restarts"`
		Node         string    `json:"node"`
		Age          string    `json:"age"`
		Labels       []string  `json:"labels"`
		Condition    Condition `json:"condition"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sPodsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	pods, err := clientset.CoreV1().Pods(h.NS).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get pods when calling GetK8sPodsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, pod := range pods.Items {
		name := pod.GenerateName + pod.Name

		age := pod.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range pod.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		readyContainers := 0
		for _, status := range pod.Status.ContainerStatuses {
			if status.Ready {
				readyContainers++
			}
		}
		var latestCondition v1.PodCondition
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
		restartCount := 0
		for _, status := range pod.Status.ContainerStatuses {
			restartCount += int(status.RestartCount)
		}

		response = append(response, Response{
			ID:           GenerateRandomString(10),
			Name:         name,
			ReadyActual:  readyContainers,
			ReadyDesired: len(pod.Spec.Containers),
			Phase:        string(pod.Status.Phase),
			Status:       string(latestCondition.Type),
			Age:          ElapsedTimeShort(age.Time),
			Labels:       labels,
			Node:         pod.Spec.NodeName,
			Restarts:     restartCount,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
