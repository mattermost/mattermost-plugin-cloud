package rotator

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	awsTools "github.com/mattermost/rotator/aws"
	k8sTools "github.com/mattermost/rotator/k8s"
	"github.com/mattermost/rotator/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// newNodes separates old nodes from new in a provided slice and returns new.
func newNodes(allNodes, oldNodes []string) []string {
	mb := make(map[string]struct{}, len(oldNodes))
	for _, x := range oldNodes {
		mb[x] = struct{}{}
	}
	var newNodes []string
	for _, x := range allNodes {
		if _, found := mb[x]; !found {
			newNodes = append(newNodes, x)
		}
	}
	return newNodes
}

// SetObject sets each AutoscalingGroup object.
func (autoscalingGroup *AutoscalingGroup) SetObject(asg *autoscaling.Group) error {
	autoscalingGroup.Name = *asg.AutoScalingGroupName
	autoscalingGroup.DesiredCapacity = int(*asg.DesiredCapacity)
	nodeHostNames, err := awsTools.GetNodeHostnames(asg.Instances)
	if err != nil {
		return errors.Wrap(err, "Failed to get asg instance node names and set asg object")
	}
	autoscalingGroup.Nodes = nodeHostNames
	return nil
}

// popNodes removes a node that completed rotation from the AutoscalingGroup object node list.
func (autoscalingGroup *AutoscalingGroup) popNodes(popNodes []string) {
	var updatedList []string
	for _, node := range autoscalingGroup.Nodes {
		nodeFound := false
		for _, popNode := range popNodes {
			if popNode == node {
				nodeFound = true
				break
			}
		}
		if !nodeFound {
			updatedList = append(updatedList, node)
		}
	}
	autoscalingGroup.Nodes = updatedList
	return
}

// DrainNodes covers all node drain actions.
func (autoscalingGroup *AutoscalingGroup) DrainNodes(nodesToDrain []string, attempts, gracePeriod, wait, waitBetweenPodEvictions int, clientset *kubernetes.Clientset, logger *logrus.Entry, nodeType string) error {
	ctx := context.TODO()

	drainOptions := &DrainOptions{
		DeleteLocalData:    true,
		IgnoreDaemonsets:   true,
		Timeout:            600,
		GracePeriodSeconds: gracePeriod,
	}

	logger.Infof("Draining %d nodes", len(nodesToDrain))

	remaining := len(nodesToDrain)

	for _, nodeToDrain := range nodesToDrain {
		logger.Infof("Draining node %s", nodeToDrain)

		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeToDrain, metav1.GetOptions{})
		if k8sErrors.IsNotFound(err) {
			logger.Warnf("Node %s not found, assuming already drained", nodeToDrain)
		} else if err != nil {
			return errors.Wrapf(err, "Failed to get node %s", nodeToDrain)
		} else {
			err = Drain(clientset, []*corev1.Node{node}, drainOptions, waitBetweenPodEvictions)
			for i := 1; i < attempts && err != nil; i++ {
				logger.Warnf("Failed to drain node %q on attempt %d, retrying up to %d times", nodesToDrain, i, attempts)
				err = Drain(clientset, []*corev1.Node{node}, drainOptions, waitBetweenPodEvictions)
			}
			if err != nil {
				return errors.Wrapf(err, "Failed to drain node %s", nodeToDrain)
			}
			logger.Infof("Node %s drained successfully", nodeToDrain)
		}

		//Terminating nodes after each drain rotation ensures that nodes do not hang and create alerts.
		if nodeType == "worker" {
			err = awsTools.TerminateNodes([]string{nodeToDrain}, logger)
			if err != nil {
				return err
			}

			err = k8sTools.DeleteClusterNodes([]string{nodeToDrain}, clientset, logger)
			if err != nil {
				return err
			}

			logger.Info("Removing node from rotation list")
			autoscalingGroup.popNodes([]string{nodeToDrain})

		}
		remaining--
		if remaining > 0 {
			logger.Infof("Waiting for %d seconds before next node drain", wait)
			time.Sleep(time.Duration(wait) * time.Second)
		}

	}

	return nil
}

// getk8sClientset returns the k8s clientset. Uses local config if no client is provided.
func getk8sClientset(cluster *model.Cluster) (*kubernetes.Clientset, error) {
	if cluster.ClientSet != nil {
		return cluster.ClientSet, nil
	}

	clientSet, err := k8sTools.GetClientset()
	if err != nil {
		return nil, err
	}
	return clientSet, nil

}

// GetSetAutoscalingGroups separates master from worker Autoscaling Groups and prepares the respective objects.
func (metadata *RotatorMetadata) GetSetAutoscalingGroups(cluster *model.Cluster) error {
	asgs, err := awsTools.GetAutoscalingGroups(cluster.ClusterID)
	if err != nil {
		return err
	}
	logger.Infof("Cluster with cluster ID %s is consisted of %d Autoscaling Groups", cluster.ClusterID, len(asgs))

	for _, asg := range asgs {
		autoscalingGroup := AutoscalingGroup{}
		err := autoscalingGroup.SetObject(asg)
		if err != nil {
			return err
		}

		if strings.Contains(autoscalingGroup.Name, "master") && cluster.RotateMasters {
			metadata.MasterGroups = append(metadata.MasterGroups, autoscalingGroup)
		} else if !strings.Contains(autoscalingGroup.Name, "master") && cluster.RotateWorkers {
			metadata.WorkerGroups = append(metadata.WorkerGroups, autoscalingGroup)
		}
	}
	return nil
}
