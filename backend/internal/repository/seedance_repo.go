package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type seedanceRepository struct {
	db *sql.DB
}

func NewSeedanceRepository(db *sql.DB) service.SeedanceRepository {
	return &seedanceRepository{db: db}
}

func (r *seedanceRepository) CreateResource(ctx context.Context, resource *service.SeedanceResource) error {
	if resource == nil {
		return errors.New("seedance resource is nil")
	}
	return r.db.QueryRowContext(ctx, `
		INSERT INTO seedance_resources
			(resource_id, resource_type, channel, user_id, api_key_id, group_id, account_id, parent_id, task_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at`,
		resource.ResourceID, resource.ResourceType, resource.Channel, resource.UserID,
		resource.APIKeyID, resource.GroupID, resource.AccountID, resource.ParentID, resource.TaskID,
	).Scan(&resource.ID, &resource.CreatedAt, &resource.UpdatedAt)
}

func (r *seedanceRepository) GetResource(ctx context.Context, userID int64, apiKeyID *int64, resourceType, resourceID string) (*service.SeedanceResource, error) {
	resource := &service.SeedanceResource{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, resource_id, resource_type, channel, user_id, api_key_id, group_id,
		       account_id, parent_id, task_id, created_at, updated_at
		FROM seedance_resources
		WHERE user_id = $1 AND api_key_id IS NOT DISTINCT FROM $2
		  AND resource_type = $3 AND resource_id = $4`,
		userID, apiKeyID, resourceType, resourceID,
	).Scan(
		&resource.ID, &resource.ResourceID, &resource.ResourceType, &resource.Channel,
		&resource.UserID, &resource.APIKeyID, &resource.GroupID, &resource.AccountID,
		&resource.ParentID, &resource.TaskID, &resource.CreatedAt, &resource.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSeedanceResourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seedance resource: %w", err)
	}
	return resource, nil
}

func (r *seedanceRepository) CreateTask(ctx context.Context, task *service.SeedanceVideoTask) error {
	if task == nil {
		return errors.New("seedance task is nil")
	}
	return r.db.QueryRowContext(ctx, `
		INSERT INTO seedance_video_tasks
			(task_id, user_id, api_key_id, group_id, account_id, model, status,
			 duration_seconds, resolution, request_body, response_body,
			 last_error_code, last_error_message, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		task.TaskID, task.UserID, task.APIKeyID, task.GroupID, task.AccountID, task.Model,
		task.Status, task.DurationSeconds, task.Resolution, task.RequestBody, task.ResponseBody,
		task.LastErrorCode, task.LastErrorMessage, task.CompletedAt,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

const seedanceTaskColumns = `
	id, task_id, user_id, api_key_id, group_id, account_id, model, status,
	duration_seconds, resolution, request_body, response_body,
	last_error_code, last_error_message, billed_at, created_at, updated_at, completed_at`

func scanSeedanceTask(scanner interface{ Scan(dest ...any) error }) (*service.SeedanceVideoTask, error) {
	task := &service.SeedanceVideoTask{}
	err := scanner.Scan(
		&task.ID, &task.TaskID, &task.UserID, &task.APIKeyID, &task.GroupID,
		&task.AccountID, &task.Model, &task.Status, &task.DurationSeconds,
		&task.Resolution, &task.RequestBody, &task.ResponseBody,
		&task.LastErrorCode, &task.LastErrorMessage, &task.BilledAt,
		&task.CreatedAt, &task.UpdatedAt, &task.CompletedAt,
	)
	return task, err
}

func (r *seedanceRepository) GetTask(ctx context.Context, userID int64, apiKeyID *int64, taskID string) (*service.SeedanceVideoTask, error) {
	task, err := scanSeedanceTask(r.db.QueryRowContext(ctx, `
		SELECT `+seedanceTaskColumns+`
		FROM seedance_video_tasks
		WHERE user_id = $1 AND api_key_id IS NOT DISTINCT FROM $2 AND task_id = $3`,
		userID, apiKeyID, taskID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSeedanceTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get seedance task: %w", err)
	}
	return task, nil
}

func (r *seedanceRepository) ListTasks(ctx context.Context, userID int64, apiKeyID *int64, page, pageSize int) (*service.SeedanceVideoTaskList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	result := &service.SeedanceVideoTaskList{}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM seedance_video_tasks
		WHERE user_id = $1 AND api_key_id IS NOT DISTINCT FROM $2`, userID, apiKeyID,
	).Scan(&result.Total); err != nil {
		return nil, fmt.Errorf("count seedance tasks: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+seedanceTaskColumns+`
		FROM seedance_video_tasks
		WHERE user_id = $1 AND api_key_id IS NOT DISTINCT FROM $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`, userID, apiKeyID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list seedance tasks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		task, scanErr := scanSeedanceTask(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan seedance task: %w", scanErr)
		}
		result.Items = append(result.Items, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate seedance tasks: %w", err)
	}
	return result, nil
}

func (r *seedanceRepository) UpdateTask(ctx context.Context, task *service.SeedanceVideoTask) error {
	if task == nil {
		return errors.New("seedance task is nil")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE seedance_video_tasks
		SET status=$2, duration_seconds=$3, resolution=$4, response_body=$5,
		    last_error_code=$6, last_error_message=$7, completed_at=$8, updated_at=NOW()
		WHERE task_id=$1`,
		task.TaskID, task.Status, task.DurationSeconds, task.Resolution, task.ResponseBody,
		task.LastErrorCode, task.LastErrorMessage, task.CompletedAt,
	)
	return err
}

func (r *seedanceRepository) ClaimTaskBilling(ctx context.Context, taskID string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE seedance_video_tasks SET billed_at=NOW(), updated_at=NOW()
		WHERE task_id=$1 AND status='completed' AND billed_at IS NULL`, taskID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *seedanceRepository) ReleaseTaskBilling(ctx context.Context, taskID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE seedance_video_tasks SET billed_at=NULL, updated_at=NOW() WHERE task_id=$1`, taskID)
	return err
}

var _ service.SeedanceRepository = (*seedanceRepository)(nil)
