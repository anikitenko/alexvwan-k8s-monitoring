package main

import (
	"context"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

type GetK8sClusterNSsHandler struct {
	ID   string
	Name string
}

func (h *GetK8sClusterNSsHandler) ServeHTTP(c echo.Context) error {
	clientset, errMsg, err := GetClientSet(h.ID, h.Name, "GetK8sClusterNSsHandler")
	if err != nil {
		logger.Warnf("%s: %v", errMsg, err)
		return c.NoContent(http.StatusInternalServerError)
	}
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Warnf("Failed to get namespaces when calling GetK8sClusterNSsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	var response = make([]string, 0)
	for _, namespace := range namespaces.Items {
		name := namespace.Name
		response = append(response, name)
	}
	return c.JSON(http.StatusOK, response)
}
