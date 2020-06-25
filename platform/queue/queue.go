// Package queue implements a boldDB backed queue for MDM Commands.
package queue

import "fmt"

const (
	DeviceCommandBucket = "mdm.DeviceCommands"
	CommandQueuedTopic  = "mdm.CommandQueued"
)

type NotFound struct {
	ResourceType string
	Message      string
}

func (e *NotFound) Error() string {
	return fmt.Sprintf("not found: %s %s", e.ResourceType, e.Message)
}

func IsNotFound(err error) bool {
	if _, ok := err.(*NotFound); ok {
		return true
	}
	return false
}
