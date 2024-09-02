package main

import (
	"k8s.io/client-go/tools/clientcmd"
)

func SelectClusterContext(config []byte, name string) ([]byte, error) {
	var result []byte
	loadConfig, err := clientcmd.Load(config)
	if err != nil {
		return result, err
	}
	loadConfig.CurrentContext = name
	for _, cluster := range loadConfig.Clusters {
		cluster.InsecureSkipTLSVerify = true
		cluster.CertificateAuthority = ""
		cluster.CertificateAuthorityData = nil
	}
	result, err = clientcmd.Write(*loadConfig)
	if err != nil {
		return result, err
	}
	return result, nil
}
