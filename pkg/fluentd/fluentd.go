package fluentd

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
)

func CreateAssets(client kubernetes.Interface, namespace string) error {
	if err := createMasterCfg(client, namespace); err != nil {
		return fmt.Errorf("error creating fluentd asset %v", err)
	}

	if err := createMasterDeploy(client, namespace); err != nil {
		return fmt.Errorf("error creating fluentd asset %v", err)
	}

	if err := createMasterSvc(client, namespace); err != nil {
		return fmt.Errorf("error creating fluentd asset %v", err)
	}

	if err := createWorkerCfg(client, namespace); err != nil {
		return fmt.Errorf("error creating fluentd asset %v", err)
	}

	if err := createWorkerDs(client, namespace); err != nil {
		return fmt.Errorf("error creating fluentd asset %v", err)
	}

	return nil
}

func DeleteAssets(client kubernetes.Interface, namespace string) error {
	if err := deleteMasterCfg(client, namespace); err != nil {
		return fmt.Errorf("error deleting fluentd asset %v", err)
	}

	if err := deleteMasterDeploy(client, namespace); err != nil {
		return fmt.Errorf("error deleting fluentd asset %v", err)
	}

	if err := deleteMasterSvc(client, namespace); err != nil {
		return fmt.Errorf("error deleting fluentd asset %v", err)
	}

	if err := deleteWorkerCfg(client, namespace); err != nil {
		return fmt.Errorf("error deleting fluentd asset %v", err)
	}

	if err := deleteWorkerDs(client, namespace); err != nil {
		return fmt.Errorf("error deleting fluentd asset %v", err)
	}

	return nil
}
