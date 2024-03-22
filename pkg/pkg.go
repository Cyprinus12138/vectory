package pkg

import "sync"

// ClusterName to be registered in the service discovery.
var (
	ClusterName    = "default"
	Status         = Start
	statusMutex    = sync.RWMutex{}
	StatusUpdating = make(chan NodeStatus)
)

func GetStatus() NodeStatus {
	statusMutex.RLock()
	defer statusMutex.RUnlock()

	return Status
}

func SetStatus(status NodeStatus) {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	Status = status
	StatusUpdating <- status
}

type NodeStatus string

func (n *NodeStatus) ToString() string {
	return string(*n)
}

const (
	Start       NodeStatus = "start"
	Init        NodeStatus = "init"
	Healthy     NodeStatus = "healthy"
	Unhealthy   NodeStatus = "unhealthy"
	Rebalancing NodeStatus = "rebalancing"
	Inactive    NodeStatus = "inactive"
)
