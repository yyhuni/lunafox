package handler

import (
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	"github.com/yyhuni/lunafox/server/internal/modules/blacklist/dto"
)

func toBlacklistPolicyOutput(policy *blacklistapp.BlacklistPolicy) dto.BlacklistPolicyResponse {
	patterns := []string{}
	if policy == nil {
		return dto.BlacklistPolicyResponse{Patterns: patterns}
	}
	patterns = append(patterns, policy.Patterns...)
	return dto.BlacklistPolicyResponse{
		Name:       policy.Name,
		Patterns:   patterns,
		ETag:       policy.ETag,
		UpdateTime: policy.UpdateTime,
	}
}

func toReplaceBlacklistPolicyInput(request dto.UpdateBlacklistPolicyRequest) blacklistapp.ReplaceBlacklistPolicyInput {
	return blacklistapp.ReplaceBlacklistPolicyInput{
		ETag:     request.ETag,
		Patterns: append([]string{}, request.Patterns...),
	}
}
