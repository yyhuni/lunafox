package agentcontrol

import (
	"strings"
	"testing"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestRejectLegacyWorkerFields(t *testing.T) {
	tests := []struct {
		name    string
		reject  func() error
	}{
		{
			name: "heartbeat worker version",
			reject: func() error {
				message := &agentcontrolv1.Heartbeat{}
				message.ProtoReflect().SetUnknown(legacyStringField(11, "1.2.3"))
				return RejectLegacyHeartbeatFields(message)
			},
		},
		{
			name: "register session worker version",
			reject: func() error {
				message := &agentcontrolv1.RegisterSession{}
				message.ProtoReflect().SetUnknown(legacyStringField(5, "1.2.3"))
				return RejectLegacyRegisterSessionFields(message)
			},
		},
		{
			name: "update worker version",
			reject: func() error {
				message := &agentcontrolv1.UpdateRequired{}
				message.ProtoReflect().SetUnknown(legacyStringField(4, "1.2.3"))
				return RejectLegacyUpdateRequiredFields(message)
			},
		},
		{
			name: "update worker image",
			reject: func() error {
				message := &agentcontrolv1.UpdateRequired{}
				message.ProtoReflect().SetUnknown(legacyStringField(6, "registry.example/worker:1.2.3"))
				return RejectLegacyUpdateRequiredFields(message)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.reject(); err == nil || !strings.Contains(err.Error(), "legacy control-plane field") {
				t.Fatalf("legacy field rejection error = %v", err)
			}
		})
	}
}

func TestLegacyFieldGuardAllowsUnrelatedUnknownField(t *testing.T) {
	message := &agentcontrolv1.UpdateRequired{}
	message.ProtoReflect().SetUnknown(legacyStringField(99, "future"))
	if err := RejectLegacyUpdateRequiredFields(message); err != nil {
		t.Fatalf("unrelated additive unknown field rejected: %v", err)
	}
}

func legacyStringField(number protowire.Number, value string) []byte {
	result := protowire.AppendTag(nil, number, protowire.BytesType)
	return protowire.AppendString(result, value)
}
