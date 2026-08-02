package model

import "testing"

func TestTaskGetStatsAggregatesByMediaType(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{
		TaskID:     "task-image-success",
		UserId:     1,
		Action:     "imageGenerate",
		Status:     TaskStatusSuccess,
		SubmitTime: 1711933200,
		Progress:   "100%",
	})
	insertTask(t, &Task{
		TaskID:     "task-image-failure",
		UserId:     1,
		Action:     "imageEdit",
		Status:     TaskStatusUnknown,
		SubmitTime: 1711936800,
		FailReason: "upstream failed",
	})
	insertTask(t, &Task{
		TaskID:     "task-video-running-progress",
		UserId:     1,
		Action:     "generate",
		Status:     TaskStatusUnknown,
		SubmitTime: 1712019600,
		Progress:   "85%",
	})
	insertTask(t, &Task{
		TaskID:     "task-video-running-status",
		UserId:     1,
		Action:     "textGenerate",
		Status:     TaskStatus("PENDING"),
		SubmitTime: 1712023200,
	})
	insertTask(t, &Task{
		TaskID:     "task-video-success",
		UserId:     1,
		Action:     "remixGenerate",
		Status:     TaskStatusSuccess,
		SubmitTime: 1712026800,
		Progress:   "100%",
	})
	insertTask(t, &Task{
		TaskID:     "task-non-media",
		UserId:     1,
		Action:     "speech",
		Status:     TaskStatusSuccess,
		SubmitTime: 1712026800,
		Progress:   "100%",
	})

	stats := TaskGetStats(SyncTaskQueryParams{
		MediaType:      TaskMediaTypeAll,
		StartTimestamp: 1711929600,
		EndTimestamp:   1712102399,
	})

	if stats.RunningCount != 2 {
		t.Fatalf("expected running_count=2, got %d", stats.RunningCount)
	}
	if len(stats.DailyCounts) != 0 {
		t.Fatalf("expected no daily counts, got %d", len(stats.DailyCounts))
	}
	if stats.TotalStats.Success != 2 || stats.TotalStats.Failure != 1 || stats.TotalStats.Running != 2 {
		t.Fatalf("unexpected total stats: %+v", stats.TotalStats)
	}
	if stats.ImageStats.Success != 1 || stats.ImageStats.Failure != 1 || stats.ImageStats.Running != 0 {
		t.Fatalf("unexpected image stats: %+v", stats.ImageStats)
	}
	if stats.VideoStats.Success != 1 || stats.VideoStats.Running != 2 || stats.VideoStats.Failure != 0 {
		t.Fatalf("unexpected video stats: %+v", stats.VideoStats)
	}
}

func TestTaskGetUserStatsFiltersByUser(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{
		TaskID:     "task-user-1",
		UserId:     1,
		Action:     "generate",
		Status:     TaskStatus("PROCESSING"),
		SubmitTime: 1712019600,
	})
	insertTask(t, &Task{
		TaskID:     "task-user-2",
		UserId:     2,
		Action:     "generate",
		Status:     TaskStatusFailure,
		SubmitTime: 1712019600,
		FailReason: "failed",
		Progress:   "100%",
	})

	stats := TaskGetUserStats(1, SyncTaskQueryParams{
		MediaType:      TaskMediaTypeAll,
		StartTimestamp: 1711929600,
		EndTimestamp:   1712102399,
	})

	if stats.TotalStats.Running != 1 || stats.TotalStats.Success != 0 || stats.TotalStats.Failure != 0 {
		t.Fatalf("unexpected user-scoped stats: %+v", stats.TotalStats)
	}
}

func TestTaskStatusGroupFilters(t *testing.T) {
	truncateTables(t)

	tasks := []*Task{
		{TaskID: "queued-not-start", Status: TaskStatusNotStart},
		{TaskID: "queued-submitted", Status: TaskStatusSubmitted},
		{TaskID: "queued-queued", Status: TaskStatusQueued},
		{TaskID: "queued-pending", Status: TaskStatus("PENDING")},
		{TaskID: "running-in-progress", Status: TaskStatusInProgress},
		{TaskID: "running-processing", Status: TaskStatus("PROCESSING")},
		{TaskID: "running-progress", Status: TaskStatusUnknown, Progress: "85%"},
		{TaskID: "success-status", Status: TaskStatusSuccess, Progress: "100%"},
		{TaskID: "success-progress", Status: TaskStatusUnknown, Progress: "100%"},
		{TaskID: "failure-status", Status: TaskStatusFailure, FailReason: "failed"},
		{TaskID: "failure-reason", Status: TaskStatusUnknown, FailReason: "upstream failed"},
	}

	for index, task := range tasks {
		task.UserId = 1
		task.Action = "generate"
		task.SubmitTime = int64(1712019600 + index)
		insertTask(t, task)
	}

	testCases := []struct {
		status string
		want   int64
	}{
		{status: "queued", want: 4},
		{status: "running", want: 3},
		{status: "success", want: 2},
		{status: "failure", want: 2},
	}

	for _, testCase := range testCases {
		t.Run(testCase.status, func(t *testing.T) {
			params := SyncTaskQueryParams{Status: testCase.status}
			if got := int64(len(TaskGetAllTasks(0, 100, params))); got != testCase.want {
				t.Fatalf("TaskGetAllTasks() returned %d tasks, want %d", got, testCase.want)
			}
			if got := TaskCountAllTasks(params); got != testCase.want {
				t.Fatalf("TaskCountAllTasks() = %d, want %d", got, testCase.want)
			}
			if got := TaskCountAllUserTask(1, params); got != testCase.want {
				t.Fatalf("TaskCountAllUserTask() = %d, want %d", got, testCase.want)
			}

			stats := TaskGetStats(params)
			gotStatsTotal := stats.TotalStats.Running + stats.TotalStats.Success + stats.TotalStats.Failure
			if gotStatsTotal != testCase.want {
				t.Fatalf("TaskGetStats() counted %d tasks, want %d", gotStatsTotal, testCase.want)
			}
		})
	}

	if got := TaskCountAllTasks(SyncTaskQueryParams{Status: string(TaskStatusSubmitted)}); got != 1 {
		t.Fatalf("exact status filter returned %d tasks, want 1", got)
	}
}

func TestGetTaskActionsForMediaType(t *testing.T) {
	allActions := getTaskActionsForMediaType(TaskMediaTypeAll)
	if len(allActions) != 7 {
		t.Fatalf("expected 7 actions for all media type, got %d", len(allActions))
	}

	imageActions := getTaskActionsForMediaType(TaskMediaTypeImage)
	if len(imageActions) != 2 {
		t.Fatalf("expected 2 image actions, got %d", len(imageActions))
	}

	videoActions := getTaskActionsForMediaType(TaskMediaTypeVideo)
	if len(videoActions) != 5 {
		t.Fatalf("expected 5 video actions, got %d", len(videoActions))
	}
}

func TestCountUserActiveMediaTasks(t *testing.T) {
	truncateTables(t)

	insertTask(t, &Task{
		TaskID: "active-image",
		UserId: 1,
		Action: "imageGenerate",
		Status: TaskStatusSubmitted,
	})
	insertTask(t, &Task{
		TaskID: "active-video",
		UserId: 1,
		Action: "textGenerate",
		Status: TaskStatusInProgress,
	})
	insertTask(t, &Task{
		TaskID: "completed-image",
		UserId: 1,
		Action: "imageEdit",
		Status: TaskStatusSuccess,
	})
	insertTask(t, &Task{
		TaskID: "active-non-media",
		UserId: 1,
		Action: "speech",
		Status: TaskStatusInProgress,
	})
	insertTask(t, &Task{
		TaskID: "other-user-video",
		UserId: 2,
		Action: "generate",
		Status: TaskStatusInProgress,
	})

	count, err := CountUserActiveMediaTasks(1)
	if err != nil {
		t.Fatalf("CountUserActiveMediaTasks() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountUserActiveMediaTasks() = %d, want 2", count)
	}
}
