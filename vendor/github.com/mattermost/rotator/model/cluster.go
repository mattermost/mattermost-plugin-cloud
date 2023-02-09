package model

import (
	"encoding/json"
	"io"

	"k8s.io/client-go/kubernetes"
)

// Cluster represents a K8s cluster.
type Cluster struct {
	ClusterID               string
	MaxScaling              int
	RotateMasters           bool
	RotateWorkers           bool
	MaxDrainRetries         int
	EvictGracePeriod        int
	WaitBetweenRotations    int
	WaitBetweenDrains       int
	WaitBetweenPodEvictions int
	ClientSet               *kubernetes.Clientset
}

// ClusterFromReader decodes a json-encoded cluster from the given io.Reader.
func ClusterFromReader(reader io.Reader) (*Cluster, error) {
	cluster := Cluster{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&cluster)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &cluster, nil
}
