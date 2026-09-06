package agentcontrol

import (
	"sync"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"google.golang.org/grpc"
)

// sendControlEvent sends a single event over the stream with optional mutex locking.
// The mutex guards concurrent Send calls from multiple goroutines (e.g. main loop + forwarder).
func sendControlEvent(
	sendMutex *sync.Mutex,
	stream grpc.BidiStreamingServer[agentcontrolv1.ConnectRequest, agentcontrolv1.ConnectResponse],
	event *agentcontrolv1.ConnectResponse,
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
