package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClientSet(id, name, handler string) (*kubernetes.Clientset, string, error) {
	var k8sConfig Kubeconfig
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, "Failed to create ObjectID based on ID when calling " + handler, err
	}
	if err := DBHelper.FindOne(KubeconfigsCollection, BsonEquals("_id", objectID), &k8sConfig); err != nil {
		return nil, "Failed to get kubeconfig when calling " + handler, err
	}

	newConfig, err := SelectClusterContext([]byte(k8sConfig.Content), name)
	if err != nil {
		return nil, "Failed to select cluster when calling " + handler, err
	}

	parseConfig, err := clientcmd.NewClientConfigFromBytes(newConfig)
	if err != nil {
		return nil, "Failed to parse kubeconfig when calling " + handler, err
	}

	kubeConfig, err := parseConfig.ClientConfig()
	if err != nil {
		return nil, "Failed to parse kubeconfig when calling " + handler, err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, "Failed to create clientset when calling " + handler, err
	}

	return clientset, "", nil
}
