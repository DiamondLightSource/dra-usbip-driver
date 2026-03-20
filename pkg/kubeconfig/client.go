package kubeconfig

import (
	"fmt"

	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetCoreClient(kubeConfigFile string) (*coreclientset.Clientset, error) {
	var csconfig *rest.Config
	var err error

	if kubeConfigFile == "" {
		csconfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config failed: %w", err)
		}
	} else {
		csconfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return nil, fmt.Errorf("config file failed: %w", err)
		}
	}

	coreclient, err := coreclientset.NewForConfig(csconfig)
	if err != nil {
		return nil, fmt.Errorf("error creating new client for config: %s", err)
	}

	return coreclient, nil
}
