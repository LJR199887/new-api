package service

import (
	"errors"
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/model"
)

var ErrMediaTaskInvalid = errors.New("invalid media task")

type MediaTaskConcurrencyError struct {
	Active int64
	Limit  int
}

func (e *MediaTaskConcurrencyError) Error() string {
	return fmt.Sprintf("当前用户最多同时运行 %d 个图片/视频任务，当前已有 %d 个任务运行中，请等待已有任务完成后再试", e.Limit, e.Active)
}

// Per-user admission locks prevent a pair of requests handled by the same
// process from both passing the count check before either task is inserted.
// The database remains the source of truth for active task state.
var mediaTaskAdmissionLocks [128]sync.Mutex

func InsertMediaTaskWithLimit(task *model.Task) error {
	if task == nil || task.UserId <= 0 {
		return ErrMediaTaskInvalid
	}

	lock := &mediaTaskAdmissionLocks[task.UserId%len(mediaTaskAdmissionLocks)]
	lock.Lock()
	defer lock.Unlock()

	user, err := model.GetUserById(task.UserId, false)
	if err != nil {
		return err
	}

	if user.MaxConcurrentMediaTasks > 0 {
		active, countErr := model.CountUserActiveMediaTasks(task.UserId)
		if countErr != nil {
			return countErr
		}
		if active >= int64(user.MaxConcurrentMediaTasks) {
			return &MediaTaskConcurrencyError{
				Active: active,
				Limit:  user.MaxConcurrentMediaTasks,
			}
		}
	}

	return task.Insert()
}
