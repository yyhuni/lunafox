package installapp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxEventHistory = 4000
	defaultSubBuffer       = 512
)

type installJob struct {
	id          string
	state       InstallState
	startedAt   time.Time
	finishedAt  time.Time
	errorMsg    string
	logs        *LogRing
	currentStep int
	totalSteps  int
	stepTitle   string

	nextEventID int64
	events      []InstallEvent
	subscribers map[int]chan InstallEvent
	nextSubID   int

	mu sync.RWMutex
}

func newInstallJob(id string, totalSteps int) *installJob {
	now := time.Now().UTC()
	job := &installJob{
		id:          id,
		state:       StateRunning,
		startedAt:   now,
		totalSteps:  totalSteps,
		logs:        NewLogRing(DefaultMaxLogLines, DefaultMaxLineSize),
		events:      make([]InstallEvent, 0, defaultMaxEventHistory),
		subscribers: map[int]chan InstallEvent{},
	}
	job.emitLocked(EventSnapshot, job.snapshotLocked())
	job.emitLocked(EventState, job.stateEventLocked())
	return job
}

func (job *installJob) isRunning() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.state == StateRunning
}

func (job *installJob) appendLog(raw string) {
	lines := []string{}

	job.mu.Lock()
	lines = job.logs.Append(raw)
	for _, line := range lines {
		if stepNo, stepTotal, title, ok := parseStepLine(line); ok {
			job.currentStep = stepNo
			if stepTotal > 0 {
				job.totalSteps = stepTotal
			}
			job.stepTitle = title
			job.emitLocked(EventState, job.stateEventLocked())
		}
		job.emitLocked(EventLog, LogEvent{Message: line})
	}
	job.mu.Unlock()
}

func (job *installJob) markDone(state InstallState, err error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if transition(job.state, state) != nil {
		return
	}
	job.state = state
	job.finishedAt = time.Now().UTC()
	if err != nil {
		job.errorMsg = strings.TrimSpace(err.Error())
	} else {
		job.errorMsg = ""
	}
	job.emitLocked(EventState, job.stateEventLocked())
	job.emitLocked(EventDone, job.snapshotLocked())
}

func (job *installJob) snapshot() InstallStateSnapshot {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.snapshotLocked()
}

func (job *installJob) snapshotLocked() InstallStateSnapshot {
	snapshot := InstallStateSnapshot{
		JobID:       job.id,
		State:       string(job.state),
		Error:       job.errorMsg,
		Logs:        job.logs.String(),
		CurrentStep: job.currentStep,
		TotalSteps:  job.totalSteps,
		StepTitle:   job.stepTitle,
	}
	if !job.startedAt.IsZero() {
		snapshot.StartedAt = job.startedAt.Format(time.RFC3339)
	}
	if !job.finishedAt.IsZero() {
		snapshot.FinishedAt = job.finishedAt.Format(time.RFC3339)
	}
	return snapshot
}

func (job *installJob) stateEventLocked() StateEvent {
	out := StateEvent{
		State:       string(job.state),
		Error:       job.errorMsg,
		CurrentStep: job.currentStep,
		TotalSteps:  job.totalSteps,
		StepTitle:   job.stepTitle,
	}
	if !job.startedAt.IsZero() {
		out.StartedAt = job.startedAt.Format(time.RFC3339)
	}
	if !job.finishedAt.IsZero() {
		out.FinishedAt = job.finishedAt.Format(time.RFC3339)
	}
	return out
}

func (job *installJob) emitLocked(eventType InstallEventType, data any) {
	job.nextEventID++
	event := InstallEvent{
		ID:        job.nextEventID,
		JobID:     job.id,
		Type:      eventType,
		Timestamp: nowRFC3339(),
		Data:      data,
	}
	job.events = append(job.events, event)
	if len(job.events) > defaultMaxEventHistory {
		start := len(job.events) - defaultMaxEventHistory
		job.events = append([]InstallEvent(nil), job.events[start:]...)
	}
	for _, subscriber := range job.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (job *installJob) subscribe(afterID int64) (<-chan InstallEvent, func()) {
	job.mu.Lock()
	backlog := make([]InstallEvent, 0, len(job.events))
	for _, event := range job.events {
		if event.ID > afterID {
			backlog = append(backlog, event)
		}
	}
	chSize := defaultSubBuffer
	if len(backlog)+defaultSubBuffer > chSize {
		chSize = len(backlog) + defaultSubBuffer
	}
	ch := make(chan InstallEvent, chSize)
	job.nextSubID++
	subID := job.nextSubID
	job.subscribers[subID] = ch
	job.mu.Unlock()

	for _, event := range backlog {
		ch <- event
	}

	cancel := func() {
		job.mu.Lock()
		defer job.mu.Unlock()
		if subscriber, ok := job.subscribers[subID]; ok {
			delete(job.subscribers, subID)
			close(subscriber)
		}
	}
	return ch, cancel
}

var stepPattern = regexp.MustCompile(`^\[(\d+)/(\d+)\]\s+(.+?)\s*$`)

func parseStepLine(line string) (int, int, string, bool) {
	match := stepPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return 0, 0, "", false
	}
	current, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, "", false
	}
	total, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, "", false
	}
	title := strings.TrimSpace(match[3])
	if title == "" {
		return 0, 0, "", false
	}
	return current, total, title, true
}

type errJobNotFound struct {
	jobID string
}

func (err *errJobNotFound) Error() string {
	return fmt.Sprintf("job not found: %s", err.jobID)
}

func (err *errJobNotFound) IsJobNotFound() bool {
	return true
}

type errJobRunning struct {
	jobID string
}

func (err *errJobRunning) Error() string {
	return fmt.Sprintf("job already running: %s", err.jobID)
}

func (err *errJobRunning) RunningJobID() string {
	return err.jobID
}
