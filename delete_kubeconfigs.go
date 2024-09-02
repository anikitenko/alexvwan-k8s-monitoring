package main

import (
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type DeleteKubeconfigHandler struct {
	ID string
}

func (h *DeleteKubeconfigHandler) ServeHTTP(c echo.Context) error {
	objectID, err := primitive.ObjectIDFromHex(h.ID)
	if err != nil {
		logger.Warnf("Failed to create ObjectID based on ID when calling DeleteKubeconfigHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := DBHelper.DeleteOne(KubeconfigsCollection, BsonEquals("_id", objectID)); err != nil {
		logger.Warnf("Failed to delete Kubeconfig when calling DeleteKubeconfigHandler: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}
