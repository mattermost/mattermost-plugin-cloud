package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// RotateClusterRequest specifies the parameters for a new cluster rotation.
type RotateClusterRequest struct {
	ClusterID               string `json:"clusterID,omitempty"`
	MaxScaling              int    `json:"maxScaling,omitempty"`
	RotateMasters           bool   `json:"rotateMasters,omitempty"`
	RotateWorkers           bool   `json:"rotateWorkers,omitempty"`
	MaxDrainRetries         int    `json:"maxDrainRetries,omitempty"`
	EvictGracePeriod        int    `json:"evictGracePeriod,omitempty"`
	WaitBetweenRotations    int    `json:"waitBetweenRotations,omitempty"`
	WaitBetweenDrains       int    `json:"waitBetweenDrains,omitempty"`
	WaitBetweenPodEvictions int    `json:"waitBetweenPodEvictions,omitempty"`
}

// NewRotateClusterRequestFromReader decodes the request and returns after validation and setting the defaults.
func NewRotateClusterRequestFromReader(reader io.Reader) (*RotateClusterRequest, error) {
	var rotateClusterRequest RotateClusterRequest
	err := json.NewDecoder(reader).Decode(&rotateClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode rotate cluster request")
	}

	err = rotateClusterRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "rotate cluster request failed validation")
	}
	rotateClusterRequest.SetDefaults()

	return &rotateClusterRequest, nil
}

// Validate validates the values of a cluster rotate request.
func (request *RotateClusterRequest) Validate() error {
	if request.ClusterID == "" {
		return errors.New("Cluster ID cannot be empty")
	}

	if request.MaxScaling < 1 {
		return errors.New("Max scaling cannot be 0 or negative")
	}

	if request.MaxDrainRetries < 0 {
		return errors.New("Max drain retries cannot be negative")
	}

	if request.EvictGracePeriod < 0 {
		return errors.New("Evict grace period cannot be negative")
	}

	if request.WaitBetweenRotations < 0 {
		return errors.New("Wait between rotations cannot be negative")
	}

	if request.WaitBetweenDrains < 0 {
		return errors.New("Wait between drains cannot be negative")
	}

	if request.WaitBetweenPodEvictions < 0 {
		return errors.New("Wait between pod evictions cannot be negative")
	}

	return nil
}

// SetDefaults sets the default values for a cluster provision request.
func (request *RotateClusterRequest) SetDefaults() {}
