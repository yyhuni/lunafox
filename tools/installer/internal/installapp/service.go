package installapp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/steps"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

type InstallerFactory interface {
	New(options cli.Options, runner execx.Runner, printer *ui.Printer) Runner
}

type Runner interface {
	Run(ctx context.Context) error
}

type defaultInstallerFactory struct{}

func (factory defaultInstallerFactory) New(options cli.Options, runner execx.Runner, printer *ui.Printer) Runner {
	return steps.NewInstaller(options, runner, printer)
}

type Service struct {
	runner           execx.Runner
	baseOut          io.Writer
	baseErr          io.Writer
	installerFactory InstallerFactory
	totalSteps       int

	mu      sync.RWMutex
	current *installJob
	last    *installJob
	seq     uint64
}

func NewService(runner execx.Runner, out io.Writer, err io.Writer) *Service {
	return &Service{
		runner:           runner,
		baseOut:          out,
		baseErr:          err,
		installerFactory: defaultInstallerFactory{},
		totalSteps:       steps.TotalSteps,
	}
}

func (service *Service) Start(options cli.Options) (string, error) {
	job := service.startJob()
	if job == nil {
		service.mu.RLock()
		currentJobID := ""
		if service.current != nil {
			currentJobID = service.current.id
		}
		service.mu.RUnlock()
		return "", &errJobRunning{jobID: currentJobID}
	}

	go service.runJob(job, options)
	return job.id, nil
}

func (service *Service) Snapshot(jobID string) (InstallStateSnapshot, error) {
	job, err := service.lookupJob(jobID)
	if err != nil {
		return InstallStateSnapshot{}, err
	}
	return job.snapshot(), nil
}

func (service *Service) Subscribe(jobID string, afterID int64) (<-chan InstallEvent, func(), error) {
	job, err := service.lookupJob(jobID)
	if err != nil {
		return nil, nil, err
	}
	ch, cancel := job.subscribe(afterID)
	return ch, cancel, nil
}

func (service *Service) startJob() *installJob {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.current != nil && service.current.isRunning() {
		return nil
	}

	jobID := fmt.Sprintf("job-%d", atomic.AddUint64(&service.seq, 1))
	job := newInstallJob(jobID, service.totalSteps)
	service.current = job
	service.last = job
	return job
}

func (service *Service) lookupJob(jobID string) (*installJob, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	if service.current != nil && service.current.id == jobID {
		return service.current, nil
	}
	if service.last != nil && service.last.id == jobID {
		return service.last, nil
	}
	return nil, &errJobNotFound{jobID: jobID}
}

func (service *Service) runJob(job *installJob, options cli.Options) {
	logWriter := &jobLogWriter{job: job}
	stdout := io.MultiWriter(service.baseOut, logWriter)
	stderr := io.MultiWriter(service.baseErr, logWriter)

	printer := ui.NewPrinter(stdout, stderr)
	installer := service.installerFactory.New(options, service.runner, printer)
	err := installer.Run(context.Background())
	if err != nil {
		printer.Error("安装失败: %v", err)
		job.markDone(StateFailed, err)
		return
	}
	job.markDone(StateSucceeded, nil)
}

type jobLogWriter struct {
	job *installJob
}

func (writer *jobLogWriter) Write(p []byte) (int, error) {
	raw := string(p)
	writer.job.appendLog(raw)
	return len(p), nil
}

func IsJobRunning(err error) (string, bool) {
	type runningJobError interface {
		RunningJobID() string
	}
	typed, ok := err.(runningJobError)
	if !ok {
		return "", false
	}
	return typed.RunningJobID(), true
}

func IsJobNotFound(err error) bool {
	type jobNotFoundError interface {
		IsJobNotFound() bool
	}
	typed, ok := err.(jobNotFoundError)
	if !ok {
		return false
	}
	return typed.IsJobNotFound()
}
