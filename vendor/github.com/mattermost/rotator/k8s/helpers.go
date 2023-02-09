package k8s

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NodesReady(nodes []string, clientset *kubernetes.Clientset, logger *logrus.Entry) error {
	wait := 600
	logger.Infof("Waiting up to %d seconds for all nodes to become ready...", wait)
	for _, node := range nodes {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
		defer cancel()
		node, err := WaitForNodeRunning(ctx, node, clientset, logger)
		if err != nil {
			return errors.Wrapf(err, "Node %s failed to get ready", node)
		}
	}
	logger.Info("All nodes in Ready state")

	return nil
}

// WaitForNodeRunning will poll a given kubernetes node at a regular interval for
// it to enter the 'Ready' state. If the node fails to become ready before
// the provided timeout then an error will be returned.
func WaitForNodeRunning(ctx context.Context, nodeName string, clientset *kubernetes.Clientset, logger *logrus.Entry) (*corev1.Node, error) {
	for {
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err == nil {
			for _, condition := range node.Status.Conditions {
				if condition.Reason == "KubeletReady" && condition.Status == corev1.ConditionTrue {
					return node, nil
				} else if condition.Reason == "KubeletReady" && condition.Status == corev1.ConditionFalse {
					logger.Infof("Node %s found but not ready, waiting...", nodeName)
				}
			}
		}
		if k8sErrors.IsNotFound(err) {
			logger.Infof("Node %s not found, waiting...", nodeName)
		} else if err != nil {
			logger.WithError(err).Errorf("Error while waiting for node %s to become ready...", nodeName)
		}

		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "timed out waiting for node to become ready")
		case <-time.After(20 * time.Second):
		}
	}
}

func DeleteClusterNodes(nodes []string, clientset *kubernetes.Clientset, logger *logrus.Entry) error {
	ctx := context.TODO()

	for _, node := range nodes {
		err := clientset.CoreV1().Nodes().Delete(ctx, node, metav1.DeleteOptions{})
		if k8sErrors.IsNotFound(err) {
			logger.Warnf("Node %s not found, assuming already removed from cluster", node)
		} else if err != nil {
			return err
		}
	}
	return nil
}

// getClientSet gets the k8s clientset
func GetClientset() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
