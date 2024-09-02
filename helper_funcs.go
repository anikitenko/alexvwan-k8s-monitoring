package main

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"math/rand"
	"time"
)

func IsConfigurationMode() {
	var k8sConfigs []Kubeconfig
	if err := DBHelper.FindAll(KubeconfigsCollection, bson.M{}, &k8sConfigs); err != nil {
		ConfigurationMode = true
	} else {
		if len(k8sConfigs) > 0 {
			ConfigurationMode = false
		} else {
			ConfigurationMode = true
		}
	}
}

func GenerateRandomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func ElapsedTimeShort(startTime time.Time) string {
	elapsed := time.Since(startTime)
	return DurationTimeShort(elapsed)
}

func DurationTimeShort(elapsed time.Duration) string {
	secs := int(elapsed.Seconds())
	mins := int(elapsed.Minutes())
	hours := int(elapsed.Hours())
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if mins > 0 {
		return fmt.Sprintf("%dm", mins)
	} else {
		return fmt.Sprintf("%ds", secs)
	}
}
