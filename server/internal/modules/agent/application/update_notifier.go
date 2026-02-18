package application

import (
	"sync"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
)

type updateNotifier struct {
	messageBus    AgentMessagePublisher
	serverVersion string
	agentImageRef string

	notifyMu sync.Mutex
	// notifiedVersion tracks whether update_required has been sent for each agent
	// while it is still reporting a version different from serverVersion.
	notifiedVersion map[int]bool
}

func newUpdateNotifier(messageBus AgentMessagePublisher, serverVersion, agentImageRef string) *updateNotifier {
	return &updateNotifier{
		messageBus:      messageBus,
		serverVersion:   serverVersion,
		agentImageRef:   agentImageRef,
		notifiedVersion: map[int]bool{},
	}
}

func (notifier *updateNotifier) maybeSendUpdateRequired(agentID int, agentVersion string) {
	if notifier == nil || notifier.messageBus == nil || notifier.serverVersion == "" || agentVersion == "" {
		return
	}
	// Upgrade decision is strict string equality: once heartbeat version equals
	// serverVersion, clear dedupe state so future mismatches can notify again.
	if agentVersion == notifier.serverVersion {
		notifier.setNotified(agentID, false)
		return
	}
	// Avoid repeated update_required pushes for the same mismatch window.
	if notifier.isNotified(agentID) {
		return
	}

	payload := agentproto.UpdateRequiredPayload{Version: notifier.serverVersion, ImageRef: notifier.agentImageRef}
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
