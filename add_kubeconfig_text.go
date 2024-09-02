package main

import (
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
)

type AddKubeconfigTextHandler struct {
	name       string
	kubeconfig string
}

func (h *AddKubeconfigTextHandler) ServeHTTP(c echo.Context) error {
	parseConfig, err := clientcmd.NewClientConfigFromBytes([]byte(h.kubeconfig))
	if err != nil {
		logger.Warnf("Failed to create client config from kubeconfig text content: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if _, err = parseConfig.RawConfig(); err != nil {
		logger.Warnf("Failed to parse kubeconfig text: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	data := Kubeconfig{
		Name:    h.name,
		Content: h.kubeconfig,
		ID:      primitive.NewObjectID(),
	}
	if err := DBHelper.InsertOne(KubeconfigsCollection, data); err != nil {
		logger.Warnf("Failed to save kubeconfig text after uploading: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
