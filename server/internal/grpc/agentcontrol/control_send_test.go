package agentcontrol

import (
	"context"
	"errors"
	"sync"
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

func TestSendRuntimeEventHandlesNilEventAndSendError(t *testing.T) {
	stream := &fakeConnectStream{ctx: context.Background()}
	if err := sendControlEvent(nil, stream, nil); err != nil {
		t.Fatalf("expected nil event to be ignored, got %v", err)
	}
	if len(stream.sent) != 0 {
		t.Fatalf("expected nil event not to be sent, got %+v", stream.sent)
	}

	sendErr := errors.New("send failed")
	errStream := &erroringConnectStream{ctx: context.Background(), sendErr: sendErr}
	err := sendControlEvent(&sync.Mutex{}, errStream, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{Task: resourcenames.Task(1, 1)},
		},
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected send error to be returned, got %v", err)
	}
}
