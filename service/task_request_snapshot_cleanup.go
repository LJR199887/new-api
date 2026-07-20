package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	taskRequestSnapshotRetention        = 7 * 24 * time.Hour
	taskRequestSnapshotCleanupInterval  = 24 * time.Hour
	taskRequestSnapshotCleanupBatchSize = 1000
)

var taskRequestSnapshotCleanupOnce sync.Once

func StartTaskRequestSnapshotCleanupTask() {
	taskRequestSnapshotCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			runTaskRequestSnapshotCleanupOnce()
			ticker := time.NewTicker(taskRequestSnapshotCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				runTaskRequestSnapshotCleanupOnce()
			}
		})
	})
}

func runTaskRequestSnapshotCleanupOnce() {
	cutoff := time.Now().Add(-taskRequestSnapshotRetention).Unix()
	total := int64(0)
	for {
		deleted, err := model.CleanupTaskRequestSnapshotsBefore(cutoff, taskRequestSnapshotCleanupBatchSize)
		if err != nil {
			logger.LogWarn(context.Background(), fmt.Sprintf("task request snapshot cleanup failed: %v", err))
			return
		}
		total += deleted
		if deleted < taskRequestSnapshotCleanupBatchSize {
			break
		}
	}
	if common.DebugEnabled && total > 0 {
		logger.LogDebug(context.Background(), "task request snapshot cleanup: deleted=%d", total)
	}
}
