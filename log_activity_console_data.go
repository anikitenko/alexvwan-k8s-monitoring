package main

import (
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"time"
)

func LogActivityConsoleData(c echo.Context) error {
	queryTime, err := time.Parse(time.RFC3339, c.QueryParam("time"))
	if err != nil {
		logger.Warnf("Failed to parse time to get activity log when calling LogActivityConsoleData: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response = make([]LogActivity, 0)
	if err := DBHelper.FindGtLtLimit(ActivityConsole, "time", queryTime, false, true, 10, &response); err != nil {
		logger.Warnf("Failed to get activity log when calling LogActivityConsoleData: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	sort.Slice(response, func(i, j int) bool {
		return response[i].Time.Before(response[j].Time)
	})

	return c.JSON(http.StatusOK, response)
}
