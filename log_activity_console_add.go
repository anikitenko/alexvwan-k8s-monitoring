package main

import (
	logger "github.com/sirupsen/logrus"
	"time"
)

func LogActivityConsoleAdd(msg, msgType string) {
	data := LogActivity{
		Type:    msgType,
		Message: msg,
	}
	data.Time = time.Now()
	if err := DBHelper.InsertOne(ActivityConsole, data); err != nil {
		logger.Warnf("Failed to save log activity: %v", err)
	}
}
