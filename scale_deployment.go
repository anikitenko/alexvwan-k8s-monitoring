package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"time"
)

type ScaleDeploymentHandler struct {
	ID         string
	Name       string
	NS         string
	Deployment string
}

func (h *ScaleDeploymentHandler) ServeHTTP(c echo.Context) error {
	type scaleType struct {
		Scale int `json:"scale"`
	}
	var scalePost scaleType
	if err := c.Bind(&scalePost); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	//clientset, errMsg, err := GetClientSet(h.ID, h.Name, "ScaleDeploymentHandler")
	//if err != nil {
	//	logger.Warnf("%s: %v", errMsg, err)
	//	return c.NoContent(http.StatusInternalServerError)
	//}

	//patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, scalePost.Scale))
	//_, err = clientset.AppsV1().Deployments(h.NS).Patch(context.TODO(), h.Deployment, types.StrategicMergePatchType, patch, v1.PatchOptions{})
	//if err != nil {
	//	panic(err.Error())
	//}

	go func() {
		for range 10 {
			LogActivityConsoleAdd("Deployment "+h.Deployment+" by "+strconv.Itoa(scalePost.Scale), "Scaling")

			time.Sleep(2 * time.Second)
		}
	}()

	return c.NoContent(http.StatusOK)
}
