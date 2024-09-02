package main

import (
	"context"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

type GetK8sCronJobsHandler struct {
	ID   string
	Name string
	NS   string
}

func (h *GetK8sCronJobsHandler) ServeHTTP(c echo.Context) error {
	type Response struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Schedule string   `json:"schedule"`
		Age      string   `json:"age"`
		Labels   []string `json:"labels"`
	}

	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sCronJobsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}

	cronJobs, err := clientset.BatchV1().CronJobs(h.NS).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get cron jobs when calling GetK8sCronJobsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]Response, 0)
	for _, cronJob := range cronJobs.Items {
		name := cronJob.GenerateName + cronJob.Name

		age := cronJob.GetObjectMeta().GetCreationTimestamp()

		labels := make([]string, 0)
		for key, value := range cronJob.GetLabels() {
			labels = append(labels, key+":"+value)
		}

		response = append(response, Response{
			ID:       GenerateRandomString(10),
			Name:     name,
			Schedule: cronJob.Spec.Schedule,
			Age:      ElapsedTimeShort(age.Time),
			Labels:   labels,
		})
	}

	return c.JSON(http.StatusOK, response)
}
