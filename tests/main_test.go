package tests

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	collector "github.com/abhinavdahiya/log-collector"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// global clients for use by all tests
var (
	client kubernetes.Interface
	cr     *collector.Collector
)

// non-configurable for now
const namespace = "testing"

// TestMain handles setup before all tests
func TestMain(m *testing.M) {
	var kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// create the clientset
	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := ready(client); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// createNamespace
	if _, err := createNamespace(client, namespace); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cr = collector.New(&collector.Config{
		K8sClient: client,
		Namespace: namespace,
	})

	if err := cr.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// run tests
	exitCode := m.Run()

	if err := cr.Cleanup(); err != nil {
		fmt.Println(err)
	}

	if err := deleteNamespace(client, namespace); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func createNamespace(c kubernetes.Interface, name string) (*v1.Namespace, error) {
	ns, err := c.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})
	if errors.IsAlreadyExists(err) {
		log.Println("ns already exists")
	} else if err != nil {
		return nil, fmt.Errorf("failed to create namespace with name %v %v", name, namespace)
	}

	return ns, nil
}

func deleteNamespace(c kubernetes.Interface, name string) error {
	return c.CoreV1().Namespaces().Delete(name, nil)
}

// Ready blocks until the cluster is considered available. The current
// implementation checks that 1 schedulable node is ready.
func ready(c kubernetes.Interface) error {
	f := func() error {
		list, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		if len(list.Items) < 1 {
			return fmt.Errorf("cluster is not ready, waiting for 1 or more worker nodes: %v", len(list.Items))
		}

		// check for 1 or more ready nodes by ignoring nodes marked
		// unschedulable or containing taints
		var oneReady bool
		for _, node := range list.Items {
			if node.Spec.Unschedulable {
				log.Println("no worker nodes checked in yet")
				continue
			}

			if len(node.Spec.Taints) != 0 {
				log.Println("no worker nodes checked in yet")
				continue
			}

			for _, condition := range node.Status.Conditions {
				if condition.Type == v1.NodeReady {
					if condition.Status == v1.ConditionTrue {
						oneReady = true
					}
					log.Println("waiting for first worker to be ready")
					break
				}
			}
		}
		if !oneReady {
			return fmt.Errorf("waiting for one worker node to be ready")
		}

		return nil
	}

	if err := retry(50, 10*time.Second, f); err != nil {
		return err
	}
	return nil
}

func retry(attempts int, delay time.Duration, f func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil {
			break
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}

	return err
}

func checkmaster() error {
	d, err := client.ExtensionsV1beta1().Deployments(namespace).Get("fluentd-master", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("fluentd-master not found %v", err)
	}
	if d.Status.Replicas != d.Status.AvailableReplicas {
		return fmt.Errorf("fluentd-master has not succeded: replicas running %d required %d", d.Status.AvailableReplicas, d.Status.Replicas)
	}
	return nil
}

func checkworker() error {
	ds, err := client.ExtensionsV1beta1().DaemonSets(namespace).Get("fluentd-worker", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("fluentd-worker not found %v", err)
	}
	if ds.Status.DesiredNumberScheduled != ds.Status.NumberReady {
		return fmt.Errorf("fluentd-worker has not succeded: replicas running %d required %d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	}
	return nil
}
