package model

import (
	"encoding/json"
	"io"
)

// Node represents a K8s node.
type Node struct {
	NodeName string
}

// NodeFromReader decodes a json-encoded node from the given io.Reader.
func NodeFromReader(reader io.Reader) (*Node, error) {
	node := Node{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&node)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &node, nil
}
