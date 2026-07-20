package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRequestSnapshotInsertIsIdempotentAndCleanupIsBatched(t *testing.T) {
	truncateTables(t)
	require.NoError(t, InsertTaskRequestSnapshot(&TaskRequestSnapshot{
		CreatedAt: 100,
		TaskID:    "task_snapshot",
		Body:      TaskRequestSnapshotPayload(`{"prompt":"first"}`),
	}))
	require.NoError(t, InsertTaskRequestSnapshot(&TaskRequestSnapshot{
		CreatedAt: 200,
		TaskID:    "task_snapshot",
		Body:      TaskRequestSnapshotPayload(`{"prompt":"second"}`),
	}))
	require.NoError(t, InsertTaskRequestSnapshot(&TaskRequestSnapshot{
		CreatedAt: 300,
		TaskID:    "task_newer",
		Body:      TaskRequestSnapshotPayload(`{"prompt":"newer"}`),
	}))

	snapshot, exists, err := GetTaskRequestSnapshot("task_snapshot")
	require.NoError(t, err)
	require.True(t, exists)
	assert.Contains(t, string(snapshot.Body), "first")

	deleted, err := CleanupTaskRequestSnapshotsBefore(250, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, exists, err = GetTaskRequestSnapshot("task_snapshot")
	require.NoError(t, err)
	assert.False(t, exists)
	_, exists, err = GetTaskRequestSnapshot("task_newer")
	require.NoError(t, err)
	assert.True(t, exists)
}
