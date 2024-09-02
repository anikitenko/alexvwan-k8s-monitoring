package main

import (
	"context"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strconv"
)

type GetK8sJobsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sJobsHandler) ServeHTTP(c echo.Context) error {
	type Condition struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	type Response struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Completions int32     `json:"completions"`
		Successful  int32     `json:"successful"`
		Age         string    `json:"age"`
		Labels      []string  `json:"labels"`
		Condition   Condition `json:"condition"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sJobsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	jobs, err := clientset.BatchV1().Jobs(h.NS).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get jobs when calling GetK8sJobsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, job := range jobs.Items {
		name := job.GenerateName + job.Name

		age := job.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range job.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		var latestCondition v1.JobCondition
		latestConditionOK := true
		if len(job.Status.Conditions) > 0 {
			latestCondition = job.Status.Conditions[0]
			for _, condition := range job.Status.Conditions {
				if condition.LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
					latestCondition = condition
				}
			}
			latestConditionOK, _ = strconv.ParseBool(string(latestCondition.Status))
		}
		latestConditionMessage := latestCondition.Message
		if latestCondition.Message == "" && latestConditionOK {
			startTime := job.Status.StartTime
			completionTime := job.Status.CompletionTime
			if startTime != nil && completionTime != nil {
				duration := completionTime.Time.Sub(startTime.Time)
				latestConditionMessage = "Job completed in " + DurationTimeShort(duration)
			} else {
				latestConditionMessage = "Job is still running"
			}
		}

		response = append(response, Response{
			ID:          GenerateRandomString(10),
			Name:        name,
			Completions: *job.Spec.Completions,
			Successful:  job.Status.Succeeded,
			Age:         ElapsedTimeShort(age.Time),
			Labels:      labels,
			Condition: Condition{
				OK:      latestConditionOK,
				Message: latestConditionMessage,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
