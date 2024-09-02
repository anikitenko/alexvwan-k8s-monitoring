package main

import (
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"mime/multipart"
	"net/http"
)

type AddKubeconfigHandler struct {
	name       string
	kubeconfig *multipart.FileHeader
}

func (h *AddKubeconfigHandler) ServeHTTP(c echo.Context) error {
	src, err := h.kubeconfig.Open()
	if err != nil {
		logger.Warnf("Failed to open kubeconfig after uploading: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			logger.Warnf("Failed to close kubeconfig after uploading: %v", err)
		}
	}(src)

	// Read the contents of the kubeconfig file
	kubeconfigBytes, err := io.ReadAll(src)
	if err != nil {
		logger.Warnf("Failed to read kubeconfig after uploading: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	parseConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		logger.Warnf("Failed to create client config from kubeconfig content: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if _, err = parseConfig.RawConfig(); err != nil {
		logger.Warnf("Failed to parse kubeconfig: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	kubeconfigString := string(kubeconfigBytes)
	data := Kubeconfig{
		Name:    h.name,
		Content: kubeconfigString,
		ID:      primitive.NewObjectID(),
	}
	if err := DBHelper.InsertOne(KubeconfigsCollection, data); err != nil {
		logger.Warnf("Failed to save kubeconfig after uploading: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
