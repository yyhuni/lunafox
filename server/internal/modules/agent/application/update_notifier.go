package application

import (
	"sync"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

type updateNotifier struct {
	messageBus           AgentMessagePublisher
	desiredAgentVersion  string
	agentImageRef        string
	workerImageRef       string
	desiredWorkerVersion string

	notifyMu sync.Mutex
	// notifiedVersion tracks whether update_required has been sent for each agent
	// while it is still reporting runtime versions different from desired versions.
	notifiedVersion map[int]bool
}

func newUpdateNotifier(messageBus AgentMessagePublisher, desiredAgentVersion, agentImageRef, workerImageRef, desiredWorkerVersion string) *updateNotifier {
	return &updateNotifier{
		messageBus:           messageBus,
		desiredAgentVersion:  desiredAgentVersion,
		agentImageRef:        agentImageRef,
		workerImageRef:       workerImageRef,
		desiredWorkerVersion: desiredWorkerVersion,
		notifiedVersion:      map[int]bool{},
	}
}

func (notifier *updateNotifier) maybeSendUpdateRequired(agentID int, reportedAgentVersion, reportedWorkerVersion string) {
	if notifier == nil || notifier.messageBus == nil || notifier.desiredAgentVersion == "" || reportedAgentVersion == "" {
		return
	}
	// Upgrade decision is strict string equality: once heartbeat agentVersion and
	// workerVersion both equal desired versions, clear dedupe state so future
	// mismatches can notify again.
	if reportedAgentVersion == notifier.desiredAgentVersion && reportedWorkerVersion == notifier.desiredWorkerVersion {
		notifier.setNotified(agentID, false)
		return
	}
	// Avoid repeated update_required pushes for the same mismatch window.
	if notifier.isNotified(agentID) {
		return
	}

	payload := agentdomain.UpdateRequiredPayload{
		AgentVersion:   notifier.desiredAgentVersion,
		AgentImageRef:  notifier.agentImageRef,
		WorkerImageRef: notifier.workerImageRef,
		WorkerVersion:  notifier.desiredWorkerVersion,
	}
	if notifier.messageBus.SendUpdateRequired(agentID, payload) {
		notifier.setNotified(agentID, true)
	}
}

func (notifier *updateNotifier) isNotified(agentID int) bool {
	notifier.notifyMu.Lock()
	defer notifier.notifyMu.Unlock()
	return notifier.notifiedVersion[agentID]
}

func (notifier *updateNotifier) setNotified(agentID int, value bool) {
	notifier.notifyMu.Lock()
	defer notifier.notifyMu.Unlock()
	if !value {
		// Remove state instead of storing false to keep map compact.
		delete(notifier.notifiedVersion, agentID)
		return
	}
	notifier.notifiedVersion[agentID] = true
}
