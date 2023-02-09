package rotator

import (
	"time"

	awsTools "github.com/mattermost/rotator/aws"
	k8sTools "github.com/mattermost/rotator/k8s"
	"github.com/mattermost/rotator/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// AutoscalingGroup creates a autoscaling group object.
type AutoscalingGroup struct {
	Name            string
	DesiredCapacity int
	Nodes           []string
}

// RotatorMetadata is a container struct for any metadata related to cluster rotator.
type RotatorMetadata struct {
	MasterGroups []AutoscalingGroup `json:"MasterGroups,omitempty"`
	WorkerGroups []AutoscalingGroup `json:"WorkerGroups,omitempty"`
}

// InitRotateCluster is used to call the RotateCluster function.
func InitRotateCluster(cluster *model.Cluster, rotatorMetadata *RotatorMetadata, logger *logrus.Entry) (*RotatorMetadata, error) {
	rotatorMetadata, err := RotateCluster(cluster, logger, rotatorMetadata)
	if err != nil {
		logger.WithError(err).Error("failed to rotate cluster")
		return rotatorMetadata, err
	}

	return rotatorMetadata, nil
}

// RotateCluster is used to rotate the Cluster nodes.
func RotateCluster(cluster *model.Cluster, logger *logrus.Entry, rotatorMetadata *RotatorMetadata) (*RotatorMetadata, error) {
	clientset, err := getk8sClientset(cluster)
	if err != nil {
		return rotatorMetadata, err
	}

	if rotatorMetadata.MasterGroups == nil && rotatorMetadata.WorkerGroups == nil {
		err = rotatorMetadata.GetSetAutoscalingGroups(cluster)
		if err != nil {
			return rotatorMetadata, err
		}
	}

	for index, masterASG := range rotatorMetadata.MasterGroups {
		logger.Infof("The autoscaling group %s has %d instance(s)", masterASG.Name, masterASG.DesiredCapacity)

		err = MasterNodeRotation(cluster, &masterASG, clientset, logger)
		if err != nil {
			rotatorMetadata.MasterGroups[index] = masterASG
			return rotatorMetadata, err
		}

		rotatorMetadata.MasterGroups[index] = masterASG

		logger.Infof("Checking that all %d nodes are running...", masterASG.DesiredCapacity)
		err = FinalCheck(&masterASG, clientset, logger)
		if err != nil {
			return rotatorMetadata, err
		}

		logger.Infof("ASG %s rotated successfully.", masterASG.Name)
	}

	for index, workerASG := range rotatorMetadata.WorkerGroups {
		logger.Infof("The autoscaling group %s has %d instance(s)", workerASG.Name, workerASG.DesiredCapacity)

		err = WorkerNodeRotation(cluster, &workerASG, clientset, logger)
		if err != nil {
			rotatorMetadata.WorkerGroups[index] = workerASG
			return rotatorMetadata, err
		}

		rotatorMetadata.WorkerGroups[index] = workerASG

		logger.Infof("Checking that all %d nodes are running...", workerASG.DesiredCapacity)
		err = FinalCheck(&workerASG, clientset, logger)
		if err != nil {
			return rotatorMetadata, err
		}

		logger.Infof("ASG %s rotated successfully.", workerASG.Name)
	}

	logger.Info("All ASGs rotated successfully")
	return rotatorMetadata, nil
}

// FinalCheck checks that rotation is complete.
func FinalCheck(autoscalingGroup *AutoscalingGroup, clientset *kubernetes.Clientset, logger *logrus.Entry) error {
	asg, err := awsTools.AutoScalingGroupReady(autoscalingGroup.Name, autoscalingGroup.DesiredCapacity, logger)
	if err != nil {
		return errors.Wrap(err, "Failed to get AutoscalingGroup ready")
	}

	asgNodes, err := awsTools.GetNodeHostnames(asg.Instances)
	if err != nil {
		return errors.Wrap(err, "Failed to get node hostnames")
	}

	err = k8sTools.NodesReady(asgNodes, clientset, logger)
	if err != nil {
		return errors.Wrap(err, "Failed to get cluster nodes ready")
	}

	return nil
}

// MasterNodeRotation handles rotation of master nodes.
func MasterNodeRotation(cluster *model.Cluster, autoscalingGroup *AutoscalingGroup, clientset *kubernetes.Clientset, logger *logrus.Entry) error {

	for len(autoscalingGroup.Nodes) > 0 {
		logger.Infof("The number of nodes in the ASG to be rotated is %d", len(autoscalingGroup.Nodes))

		nodesToRotate := []string{autoscalingGroup.Nodes[0]}

		err := autoscalingGroup.DrainNodes(nodesToRotate, 10, int(cluster.EvictGracePeriod), cluster.WaitBetweenDrains, cluster.WaitBetweenPodEvictions, clientset, logger, "master")
		if err != nil {
			return err
		}

		err = awsTools.DetachNodes(false, nodesToRotate, autoscalingGroup.Name, logger)
		if err != nil {
			return err
		}

		err = awsTools.TerminateNodes(nodesToRotate, logger)
		if err != nil {
			return err
		}

		logger.Info("Sleeping 60 seconds for autoscaling group to balance...")
		time.Sleep(60 * time.Second)

		autoscalingGroupReady, err := awsTools.AutoScalingGroupReady(autoscalingGroup.Name, autoscalingGroup.DesiredCapacity, logger)
		if err != nil {
			return err
		}

		nodeHostnames, err := awsTools.GetNodeHostnames(autoscalingGroupReady.Instances)
		if err != nil {
			return err
		}

		newNodes := newNodes(nodeHostnames, autoscalingGroup.Nodes)

		err = k8sTools.NodesReady(newNodes, clientset, logger)
		if err != nil {
			return err
		}

		logger.Info("Removing nodes from rotation list")
		autoscalingGroup.popNodes(nodesToRotate)

	}
	return nil
}

// WorkerNodeRotation handles rotation of worker nodes.
func WorkerNodeRotation(cluster *model.Cluster, autoscalingGroup *AutoscalingGroup, clientset *kubernetes.Clientset, logger *logrus.Entry) error {

	for len(autoscalingGroup.Nodes) > 0 {
		logger.Infof("The number of nodes in the ASG to be rotated is %d", len(autoscalingGroup.Nodes))

		var nodesToRotate []string

		if len(autoscalingGroup.Nodes) < int(cluster.MaxScaling) {
			nodesToRotate = autoscalingGroup.Nodes
		} else {
			nodesToRotate = autoscalingGroup.Nodes[:int(cluster.MaxScaling)]
		}

		err := awsTools.DetachNodes(false, nodesToRotate, autoscalingGroup.Name, logger)
		if err != nil {
			return err
		}

		logger.Info("Sleeping 60 seconds for autoscaling group to balance...")
		time.Sleep(60 * time.Second)

		autoscalingGroupReady, err := awsTools.AutoScalingGroupReady(autoscalingGroup.Name, autoscalingGroup.DesiredCapacity, logger)
		if err != nil {
			return err
		}

		nodeHostnames, err := awsTools.GetNodeHostnames(autoscalingGroupReady.Instances)
		if err != nil {
			return err
		}

		newNodes := newNodes(nodeHostnames, autoscalingGroup.Nodes)

		err = k8sTools.NodesReady(newNodes, clientset, logger)
		if err != nil {
			return err
		}

		err = autoscalingGroup.DrainNodes(nodesToRotate, 10, int(cluster.EvictGracePeriod), cluster.WaitBetweenDrains, cluster.WaitBetweenPodEvictions, clientset, logger, "worker")
		if err != nil {
			return err
		}

		if len(autoscalingGroup.Nodes) > 0 {
			logger.Infof("Waiting for %d seconds before next node rotation", cluster.WaitBetweenRotations)
			time.Sleep(time.Duration(cluster.WaitBetweenRotations) * time.Second)
		}
	}

	return nil
}
