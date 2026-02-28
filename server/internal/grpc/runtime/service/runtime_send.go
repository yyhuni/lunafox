package service

import (
	"sync"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	"google.golang.org/grpc"
)

// sendRuntimeEvent sends a single event over the stream with optional mutex locking.
// The mutex guards concurrent Send calls from multiple goroutines (e.g. main loop + forwarder).
func sendRuntimeEvent(
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[runtimev1.AgentRuntimeRequest, runtimev1.AgentRuntimeEvent],
	event *runtimev1.AgentRuntimeEvent,
) error {
	if event == nil {
		return nil
	}
	if sendMutex != nil {
		sendMutex.Lock()
		defer sendMutex.Unlock()
	}
	return stream.Send(event)
}
