package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type TaskRequestSnapshotPayload string

func (TaskRequestSnapshotPayload) GormDataType() string {
	return "text"
}

func (TaskRequestSnapshotPayload) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Dialector.Name() == "mysql" {
		return "MEDIUMTEXT"
	}
	return "TEXT"
}

// TaskRequestSnapshot stores a bounded, sanitized copy of the client request.
// It lives outside tasks so polling and task list queries never load the payload.
type TaskRequestSnapshot struct {
	ID           int64                      `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt    int64                      `json:"created_at" gorm:"bigint;index"`
	TaskID       string                     `json:"task_id" gorm:"type:varchar(191);uniqueIndex"`
	Method       string                     `json:"method" gorm:"type:varchar(16)"`
	RequestPath  string                     `json:"request_path" gorm:"type:varchar(512)"`
	ContentType  string                     `json:"content_type" gorm:"type:varchar(191)"`
	Body         TaskRequestSnapshotPayload `json:"body"`
	OriginalSize int64                      `json:"original_size"`
	Truncated    bool                       `json:"truncated"`
}

func InsertTaskRequestSnapshot(snapshot *TaskRequestSnapshot) error {
	if snapshot == nil || snapshot.TaskID == "" {
		return nil
	}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_id"}},
		DoNothing: true,
	}).Create(snapshot).Error
}

func GetTaskRequestSnapshot(taskID string) (*TaskRequestSnapshot, bool, error) {
	if taskID == "" {
		return nil, false, nil
	}
	var snapshot TaskRequestSnapshot
	err := DB.Where("task_id = ?", taskID).First(&snapshot).Error
	exists, err := RecordExist(err)
	if err != nil || !exists {
		return nil, exists, err
	}
	return &snapshot, true, nil
}

func CleanupTaskRequestSnapshotsBefore(cutoff int64, limit int) (int64, error) {
	if cutoff <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = 1000
	}

	var ids []int64
	if err := DB.Model(&TaskRequestSnapshot{}).
		Where("created_at < ?", cutoff).
		Order("id").
		Limit(limit).
		Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}

	result := DB.Where("id IN ?", ids).Delete(&TaskRequestSnapshot{})
	return result.RowsAffected, result.Error
}
