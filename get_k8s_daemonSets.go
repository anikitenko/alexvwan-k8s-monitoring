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

type GetK8sDaemonSetsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sDaemonSetsHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		DesiredReplicas  int32     `json:"desired_replicas"`
		CurrentReplicas  int32     `json:"current_replicas"`
		ReadyReplicas    int32     `json:"ready_replicas"`
		UpToDateReplicas int32     `json:"up_to_date_replicas"`
		Age              string    `json:"age"`
		Labels           []string  `json:"labels"`
		NodeSelector     []string  `json:"selectors"`
		Condition        Condition `json:"condition"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sDaemonSetsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	daemonSets, err := clientset.AppsV1().DaemonSets(h.NS).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get daemon sets when calling GetK8sDaemonSetsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, ds := range daemonSets.Items {
		name := ds.GenerateName + ds.Name

		age := ds.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range ds.GetLabels() {
			labels = append(labels, key+":"+value)
		}
		nodeSelector := make([]string, 0)
		for key, value := range ds.Spec.Template.Spec.NodeSelector {
			nodeSelector = append(nodeSelector, key+":"+value)
		}

		var latestCondition v1.DaemonSetCondition
		latestConditionOK := true
		if len(ds.Status.Conditions) > 0 {
			latestCondition = ds.Status.Conditions[0]
			for _, condition := range ds.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
			latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
		}
		latestConditionMessage := latestCondition.Message
		if latestCondition.Message == "" && latestConditionOK {
			latestConditionMessage = "Daemon Set is OK"
		}

		response = append(response, Response{
			ID:               GenerateRandomString(10),
			Name:             name,
			DesiredReplicas:  ds.Status.DesiredNumberScheduled,
			CurrentReplicas:  ds.Status.CurrentNumberScheduled,
			ReadyReplicas:    ds.Status.NumberReady,
			UpToDateReplicas: ds.Status.UpdatedNumberScheduled,
			Age:              ElapsedTimeShort(age.Time),
			Labels:           labels,
			NodeSelector:     nodeSelector,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
