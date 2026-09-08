package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestIsTaskAwaitingUpstreamID(t *testing.T) {
	tests := []struct {
		name string
		task *model.Task
		want bool
	}{
		{"nil", nil, false},
		{"local placeholder", &model.Task{TaskID: "task_local"}, true},
		{"blank mapping", &model.Task{TaskID: "task_local", PrivateData: model.TaskPrivateData{UpstreamTaskID: " "}}, true},
		{"mapped UUID", &model.Task{TaskID: "task_local", PrivateData: model.TaskPrivateData{UpstreamTaskID: "0123456789abcdef0123456789abcdef"}}, false},
		{"mapped upstream new-api ID", &model.Task{TaskID: "task_local", PrivateData: model.TaskPrivateData{UpstreamTaskID: "task_remote"}}, false},
		{"legacy provider ID", &model.Task{TaskID: "0123456789abcdef0123456789abcdef"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTaskAwaitingUpstreamID(tt.task); got != tt.want {
				t.Fatalf("isTaskAwaitingUpstreamID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVideoPollingWaitsForSubmissionMapping(t *testing.T) {
	for _, modelName := range []string{"933-video2.0", "933-video2.0-480p", "933-video2.0-mini", "933-video2.0-mini-480p"} {
		t.Run(modelName, func(t *testing.T) {
			task := &model.Task{
				TaskID: "task_local", ChannelId: 1,
				Properties: model.Properties{OriginModelName: modelName, UpstreamModelName: modelName},
			}
			adaptor := &transientFailureAdaptor{statusCode: http.StatusServiceUnavailable}
			for _, status := range []model.TaskStatus{model.TaskStatusSubmitted, model.TaskStatusInProgress} {
				task.Status = status
				// Both background polling and explicit refresh must wait while the
				// upstream POST is still running, without marking the task failed.
				if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, task.TaskID, map[string]*model.Task{task.TaskID: task}); err != nil {
					t.Fatal(err)
				}
				if err := RefreshVideoTask(context.Background(), task); err != nil {
					t.Fatal(err)
				}
				if adaptor.fetchCalls != 0 || task.Status != status || task.FailReason != "" || task.FinishTime != 0 {
					t.Fatalf("unmapped task was polled or failed: calls=%d task=%+v", adaptor.fetchCalls, task)
				}
			}
			// Once submission saves the mapping, polling must use the provider ID.
			task.PrivateData.UpstreamTaskID = "0123456789abcdef0123456789abcdef"
			if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, task.TaskID, map[string]*model.Task{task.TaskID: task}); err != nil {
				t.Fatal(err)
			}
			if adaptor.fetchCalls != 1 || adaptor.fetchedID != task.PrivateData.UpstreamTaskID {
				t.Fatalf("polling used wrong ID: calls=%d id=%q", adaptor.fetchCalls, adaptor.fetchedID)
			}
		})
	}
}

func TestVideoPollingPreservesLegacyAndMappedProviderIDs(t *testing.T) {
	for _, task := range []*model.Task{
		{TaskID: "legacy-provider-id", Status: model.TaskStatusInProgress},
		{TaskID: "task_local", Status: model.TaskStatusInProgress, PrivateData: model.TaskPrivateData{UpstreamTaskID: "task_remote"}},
	} {
		adaptor := &transientFailureAdaptor{statusCode: http.StatusServiceUnavailable}
		if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, task.TaskID, map[string]*model.Task{task.TaskID: task}); err != nil {
			t.Fatal(err)
		}
		if adaptor.fetchCalls != 1 || adaptor.fetchedID != task.GetUpstreamTaskID() {
			t.Fatalf("provider ID not polled: calls=%d id=%q", adaptor.fetchCalls, adaptor.fetchedID)
		}
	}
}

func TestVideoPollingSkipsTerminalTasks(t *testing.T) {
	for _, status := range []model.TaskStatus{model.TaskStatusSuccess, model.TaskStatusFailure} {
		task := &model.Task{
			TaskID: "task_local", Status: status,
			PrivateData: model.TaskPrivateData{UpstreamTaskID: "0123456789abcdef0123456789abcdef"},
		}
		adaptor := &transientFailureAdaptor{statusCode: http.StatusNotFound}
		if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, task.TaskID, map[string]*model.Task{task.TaskID: task}); err != nil {
			t.Fatal(err)
		}
		if adaptor.fetchCalls != 0 || task.Status != status {
			t.Fatalf("terminal task was polled: calls=%d status=%s", adaptor.fetchCalls, task.Status)
		}
	}
}

func TestVideoPollingPersistedSubmissionMapping(t *testing.T) {
	task := &model.Task{
		TaskID: model.GenerateTaskID(), ChannelId: 1,
		Status: model.TaskStatusInProgress, Progress: "0%",
		Properties: model.Properties{OriginModelName: "933-video2.0"},
	}
	if err := task.Insert(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { model.DB.Delete(&model.Task{}, task.ID) })

	// The polling query really does select placeholders while POST is in flight.
	var pending *model.Task
	for _, candidate := range model.GetAllUnFinishSyncTasks(1000) {
		if candidate.ID == task.ID {
			pending = candidate
			break
		}
	}
	if pending == nil || !isTaskAwaitingUpstreamID(pending) {
		t.Fatal("persisted placeholder was not recognized as awaiting submission")
	}
	adaptor := &transientFailureAdaptor{
		responseBody: []byte(`{"status":"queued"}`),
		taskInfo:     &relaycommon.TaskInfo{Status: model.TaskStatusQueued},
	}
	if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, pending.TaskID, map[string]*model.Task{pending.TaskID: pending}); err != nil {
		t.Fatal(err)
	}
	if adaptor.fetchCalls != 0 {
		t.Fatal("placeholder triggered an upstream request")
	}

	// Submission finishes and persists the real ID. A stale polling snapshot
	// must not send the public ID or overwrite the newly saved mapping.
	task.PrivateData.UpstreamTaskID = "0123456789abcdef0123456789abcdef"
	if err := task.Update(); err != nil {
		t.Fatal(err)
	}
	if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, pending.TaskID, map[string]*model.Task{pending.TaskID: pending}); err != nil {
		t.Fatal(err)
	}
	if adaptor.fetchCalls != 0 {
		t.Fatal("stale placeholder triggered an upstream request")
	}
	ready, exists, err := model.GetByOnlyTaskId(task.TaskID)
	if err != nil || !exists {
		t.Fatalf("reload submitted task: exists=%v err=%v", exists, err)
	}
	if ready.PrivateData.UpstreamTaskID != task.PrivateData.UpstreamTaskID {
		t.Fatal("submission mapping was overwritten")
	}
	if err := updateVideoSingleTask(context.Background(), adaptor, &model.Channel{}, ready.TaskID, map[string]*model.Task{ready.TaskID: ready}); err != nil {
		t.Fatal(err)
	}
	if adaptor.fetchCalls != 1 || adaptor.fetchedID != task.PrivateData.UpstreamTaskID {
		t.Fatalf("wrong upstream query: calls=%d id=%q", adaptor.fetchCalls, adaptor.fetchedID)
	}
	updated, exists, err := model.GetByOnlyTaskId(task.TaskID)
	if err != nil || !exists || updated.Status != model.TaskStatusQueued || updated.PrivateData.UpstreamTaskID != task.PrivateData.UpstreamTaskID {
		t.Fatalf("poll result not persisted correctly: task=%+v err=%v", updated, err)
	}
}
