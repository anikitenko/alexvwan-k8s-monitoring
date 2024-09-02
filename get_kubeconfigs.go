package main

import (
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
)

type GetKubeconfigsHandler struct {
}

func (h *GetKubeconfigsHandler) ServeHTTP(c echo.Context) error {
	var k8sConfigs []Kubeconfig
	var k8sConfigParsed = make([]KubeConfigParsed, 0)
	if err := DBHelper.FindAll(KubeconfigsCollection, bson.M{}, &k8sConfigs); err != nil {
		logger.Warnf("Failed to get kubeconfigs when calling GetKubeconfigsHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, k8sConfig := range k8sConfigs {
		parseConfig, err := clientcmd.NewClientConfigFromBytes([]byte(k8sConfig.Content))
		if err != nil {
			logger.Warnf("Failed to create client config from kubeconfig content when calling GetKubeconfigsHandler: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		config, err := parseConfig.RawConfig()
		if err != nil {
			logger.Warnf("Failed to parse kubeconfig when calling GetKubeconfigsHandler: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		var kubeClusters []KubeConfigClustersParsed
		for name, cluster := range config.Clusters {
			var kubeCluster KubeConfigClustersParsed
			kubeCluster.ID = GenerateRandomString(10)
			kubeCluster.Name = name
			kubeCluster.Server = cluster.Server
			kubeClusters = append(kubeClusters, kubeCluster)
		}
		k8sConfigParsed = append(k8sConfigParsed, KubeConfigParsed{
			ID:       k8sConfig.ID.Hex(),
			Name:     k8sConfig.Name,
			Clusters: kubeClusters,
		})
	}
	return c.JSON(http.StatusOK, k8sConfigParsed)
}
