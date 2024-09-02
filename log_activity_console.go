package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

func LogActivityConsole(c echo.Context) error {
	// Set the headers for SSE
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// Flush to ensure the headers are sent
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Flush()

	// Create a change stream to watch the collection
	pipeline := mongo.Pipeline{}
	stream, err := DBHelper.db.Collection(ActivityConsole).Watch(context.Background(), pipeline)
	if err != nil {
		logger.Warnf("Failed to start watching when calling LogActivityConsole: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer func() {
		if err := stream.Close(context.Background()); err != nil {
			logger.Warnf("Failed to close stream of watching when calling LogActivityConsole: %v", err)
		}
	}()

	// Continuously send data
	for {
		select {
		case <-c.Request().Context().Done():
			// Client has disconnected
			return nil
		default:
			if stream.Next(context.Background()) {
				var response = make([]LogActivity, 0)
				if err := DBHelper.FindAll(ActivityConsole, bson.M{}, &response); err != nil {
					logger.Warnf("Failed to get activity log when calling LogActivityConsole: %v", err)
					return c.NoContent(http.StatusInternalServerError)
				}
				// Convert the response to JSON
				data, err := json.Marshal(response)
				if err != nil {
					logger.Warnf("Failed to marshal activity log: %v", err)
					return c.NoContent(http.StatusInternalServerError)
				}

				// Write the SSE event to the response
				_, err = fmt.Fprintf(c.Response(), "data: %s\n\n", data)
				if err != nil {
					logger.Warnf("Failed to write to response for activity log: %v", err)
					return nil
				}

				// Flush to ensure the event is sent
				c.Response().Flush()
			} else if err := stream.Err(); err != nil {
				logger.Warnf("Failed to watch stream when calling LogActivityConsole: %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
