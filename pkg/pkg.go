package pkg

// ClusterName to be registered in the service discovery.
var ClusterName = "default"
var Status = Start

type NodeStatus string

const (
	Start       NodeStatus = "start"
	Init        NodeStatus = "init"
	Healthy     NodeStatus = "healthy"
	Unhealthy   NodeStatus = "unhealthy"
	Rebalancing NodeStatus = "rebalancing"
	Inactive    NodeStatus = "inactive"
)
