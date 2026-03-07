package server

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yyhuni/lunafox/worker/internal/pkg"
	"go.uber.org/zap"
)

const defaultMaxQueuedItemsFactor = 10

var ErrBatchQueueOverflow = errors.New("batch sender queue limit exceeded")

// BatchSender handles batched sending of scan results to Server.
// It accumulates items and sends them in batches to reduce HTTP overhead.
type BatchSender struct {
	ctx       context.Context
	client    ServerClient
	scanID    int
	targetID  int
	dataType  string // "subdomain", "website", "endpoint", "port"
	batchSize int
	// maxQueuedItems bounds in-memory queue growth when send retries keep failing.
	maxQueuedItems int

	mu      sync.Mutex
	batch   []any
	sent    int // total items sent
	batches int // total batches sent
	retries int // total retry re-queue attempts
	dropped int // total dropped items caused by queue overflow
}

// NewBatchSender creates a new batch sender
func NewBatchSender(ctx context.Context, client ServerClient, scanID, targetID int, dataType string, batchSize int) *BatchSender {
	return NewBatchSenderWithQueueLimit(ctx, client, scanID, targetID, dataType, batchSize, 0)
}

// NewBatchSenderWithQueueLimit creates a new batch sender with explicit queue limit.
// maxQueuedItems<=0 uses default (batchSize * defaultMaxQueuedItemsFactor).
func NewBatchSenderWithQueueLimit(ctx context.Context, client ServerClient, scanID, targetID int, dataType string, batchSize, maxQueuedItems int) *BatchSender {
	if batchSize <= 0 {
		batchSize = 1000 // default batch size
	}
	if maxQueuedItems <= 0 {
		maxQueuedItems = batchSize * defaultMaxQueuedItemsFactor
	}
	if maxQueuedItems < batchSize {
		maxQueuedItems = batchSize
	}
	return &BatchSender{
		ctx:            ctx,
		client:         client,
		scanID:         scanID,
		targetID:       targetID,
		dataType:       dataType,
		batchSize:      batchSize,
		maxQueuedItems: maxQueuedItems,
		batch:          make([]any, 0, batchSize),
	}
}

// Add adds an item to the batch. Automatically sends when batch is full.
// Returns context.Canceled or context.DeadlineExceeded if context is done.
func (s *BatchSender) Add(item any) error {
	// Check context before processing
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	s.mu.Lock()
	if len(s.batch) >= s.maxQueuedItems {
		s.dropped++
		droppedTotal := s.dropped
		queued := len(s.batch)
		limit := s.maxQueuedItems
		s.mu.Unlock()

		pkg.Logger.Error("Batch queue limit exceeded; dropping new item",
			zap.String("type", s.dataType),
			zap.Int("queue.length", queued),
			zap.Int("queue.max_items", limit),
			zap.Int("drop.total", droppedTotal),
		)
		return queueOverflowError(s.dataType, queued, limit, 1)
	}
	s.batch = append(s.batch, item)
	shouldSend := len(s.batch) >= s.batchSize
	s.mu.Unlock()

	if shouldSend {
		return s.sendBatch()
	}
	return nil
}

// Flush sends any remaining items in the batch
func (s *BatchSender) Flush() error {
	s.mu.Lock()
	if len(s.batch) == 0 {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	return s.sendBatch()
}

// Stats returns the total items and batches sent
func (s *BatchSender) Stats() (items, batches int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent, s.batches
}

// QueueMetrics returns queue observability indicators:
// queued items, retry count, and dropped count.
func (s *BatchSender) QueueMetrics() (queued, retries, dropped int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.batch), s.retries, s.dropped
}

// sendBatch sends the current batch to the server
func (s *BatchSender) sendBatch() error {
	// Check context before sending
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

	s.mu.Lock()
	if len(s.batch) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Copy batch and clear so new items can be queued while sending
	toSend := make([]any, len(s.batch))
	copy(toSend, s.batch)
	s.batch = s.batch[:0]
	s.mu.Unlock()

	if err := s.client.PostBatch(s.ctx, s.scanID, s.targetID, s.dataType, toSend); err != nil {
		// Check if it's a non-retryable error (4xx)
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && !httpErr.IsRetryable() {
			pkg.Logger.Error("Non-retryable error sending batch (data validation issue)",
				zap.String("type", s.dataType),
				zap.Int("count", len(toSend)),
				zap.Int("http.response.status_code", httpErr.StatusCode),
				zap.String("response", httpErr.Body))
		} else {
			pkg.Logger.Error("Failed to send batch after retries",
				zap.String("type", s.dataType),
				zap.Int("count", len(toSend)),
				zap.Error(err))
		}
		// Re-queue batch for retry on next Flush/send, preserving existing order.
		s.mu.Lock()
		requeue := append(toSend, s.batch...)
		s.retries++
		retryCount := s.retries

		droppedNow := 0
		if len(requeue) > s.maxQueuedItems {
			droppedNow = len(requeue) - s.maxQueuedItems
			requeue = requeue[:s.maxQueuedItems]
			s.dropped += droppedNow
		}
		s.batch = requeue
		queued := len(s.batch)
		droppedTotal := s.dropped
		s.mu.Unlock()

		if droppedNow > 0 {
			pkg.Logger.Error("Batch queue overflow after send failure; dropping newest queued items",
				zap.String("type", s.dataType),
				zap.Int("queue.length", queued),
				zap.Int("queue.max_items", s.maxQueuedItems),
				zap.Int("retry.count", retryCount),
				zap.Int("drop.count", droppedNow),
				zap.Int("drop.total", droppedTotal),
			)
			return errors.Join(
				fmt.Errorf("failed to send %s batch: %w", s.dataType, err),
				queueOverflowError(s.dataType, queued, s.maxQueuedItems, droppedNow),
			)
		}

		pkg.Logger.Warn("Batch re-queued for retry",
			zap.String("type", s.dataType),
			zap.Int("queue.length", queued),
			zap.Int("retry.count", retryCount),
			zap.Int("drop.total", droppedTotal),
		)
		return fmt.Errorf("failed to send %s batch: %w", s.dataType, err)
	}

	s.mu.Lock()
	s.sent += len(toSend)
	s.batches++
	totalSent := s.sent
	totalBatches := s.batches
	s.mu.Unlock()

	pkg.Logger.Debug("Batch sent",
		zap.String("type", s.dataType),
		zap.Int("count", len(toSend)),
		zap.Int("send.item_total", totalSent),
		zap.Int("send.batch_total", totalBatches))

	return nil
}

func queueOverflowError(dataType string, queued, limit, dropped int) error {
	return fmt.Errorf("%w: type=%s queued=%d limit=%d dropped=%d", ErrBatchQueueOverflow, dataType, queued, limit, dropped)
}
