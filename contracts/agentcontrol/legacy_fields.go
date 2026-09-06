// Package agentcontrol owns validation shared by both ends of the Agent
// control-plane protocol.
package agentcontrol

import (
	"fmt"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// RejectLegacyHeartbeatFields rejects removed Worker identity fields while
// preserving protobuf's normal forward-compatible treatment of other unknowns.
func RejectLegacyHeartbeatFields(message *agentcontrolv1.Heartbeat) error {
	return rejectUnknownFieldNumbers(message, map[protowire.Number]string{
		11: "worker_version",
	})
}

// RejectLegacyRegisterSessionFields rejects removed Worker identity fields.
func RejectLegacyRegisterSessionFields(message *agentcontrolv1.RegisterSession) error {
	return rejectUnknownFieldNumbers(message, map[protowire.Number]string{
		5: "worker_version",
	})
}

// RejectLegacyUpdateRequiredFields rejects removed Worker update targets.
func RejectLegacyUpdateRequiredFields(message *agentcontrolv1.UpdateRequired) error {
	return rejectUnknownFieldNumbers(message, map[protowire.Number]string{
		4: "worker_version",
		6: "worker_image",
	})
}

func rejectUnknownFieldNumbers(message proto.Message, forbidden map[protowire.Number]string) error {
	if message == nil {
		return nil
	}
	raw := message.ProtoReflect().GetUnknown()
	for len(raw) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(raw)
		if tagLength < 0 {
			return fmt.Errorf("decode control-plane unknown field tag: %v", protowire.ParseError(tagLength))
		}
		raw = raw[tagLength:]
		valueLength := protowire.ConsumeFieldValue(number, wireType, raw)
		if valueLength < 0 {
			return fmt.Errorf("decode control-plane unknown field %d: %v", number, protowire.ParseError(valueLength))
		}
		if name, removed := forbidden[number]; removed {
			return fmt.Errorf("legacy control-plane field %s is not supported", name)
		}
		raw = raw[valueLength:]
	}
	return nil
}
