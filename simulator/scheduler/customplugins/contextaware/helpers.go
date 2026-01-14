package contextaware

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

func getNodeReadyDuration(node *v1.Node) time.Duration {
	if node == nil {
		return 0
	}
	for _, condition := range node.Status.Conditions {
		// Check for all contitions to find NodeReady
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			// Return the time since the last transition to Ready
			return time.Since(condition.LastTransitionTime.Time)
		}
	}
	return 0
}
