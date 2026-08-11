package repository

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/groupusertag"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usertag"
	"github.com/Wei-Shaw/sub2api/ent/usertagassignment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userTagRepository struct {
	client *dbent.Client
}

func NewUserTagRepository(client *dbent.Client) service.UserTagRepository {
	return &userTagRepository{client: client}
}

func (r *userTagRepository) Create(ctx context.Context, tag *service.UserTag) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.UserTag.Create().SetName(tag.Name).Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrUserTagNameExists)
	}
	copyUserTag(tag, created)
	return nil
}

func (r *userTagRepository) GetByID(ctx context.Context, id int64) (*service.UserTag, error) {
	client := clientFromContext(ctx, r.client)
	entity, err := client.UserTag.Query().Where(usertag.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserTagNotFound, nil)
	}
	return userTagEntityToService(entity), nil
}

func (r *userTagRepository) List(ctx context.Context) ([]service.UserTag, error) {
	client := clientFromContext(ctx, r.client)
	entities, err := client.UserTag.Query().Order(usertag.ByName()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]service.UserTag, 0, len(entities))
	for _, entity := range entities {
		result = append(result, *userTagEntityToService(entity))
	}
	return result, nil
}

func (r *userTagRepository) Update(ctx context.Context, tag *service.UserTag) error {
	client := clientFromContext(ctx, r.client)
	updated, err := client.UserTag.UpdateOneID(tag.ID).SetName(tag.Name).Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserTagNotFound, service.ErrUserTagNameExists)
	}
	copyUserTag(tag, updated)
	return nil
}

func (r *userTagRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserTag.Delete().Where(usertag.IDEQ(id)).Exec(ctx)
	return translatePersistenceError(err, service.ErrUserTagNotFound, nil)
}

func (r *userTagRepository) GetByIDs(ctx context.Context, ids []int64) ([]service.UserTag, error) {
	if len(ids) == 0 {
		return []service.UserTag{}, nil
	}
	client := clientFromContext(ctx, r.client)
	entities, err := client.UserTag.Query().Where(usertag.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(entities) != len(ids) {
		return nil, service.ErrUserTagNotFound
	}
	result := make([]service.UserTag, 0, len(entities))
	for _, entity := range entities {
		result = append(result, *userTagEntityToService(entity))
	}
	return result, nil
}

func (r *userTagRepository) GetByUserID(ctx context.Context, userID int64) ([]service.UserTag, error) {
	client := clientFromContext(ctx, r.client)
	entities, err := client.UserTag.Query().
		Where(usertag.HasUsersWith(user.IDEQ(userID))).
		Order(usertag.ByName()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return userTagEntitiesToService(entities), nil
}

func (r *userTagRepository) ReplaceUserTags(ctx context.Context, userID int64, tagIDs []int64) error {
	return r.inTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		if _, err := client.UserTagAssignment.Delete().
			Where(usertagassignment.UserIDEQ(userID)).Exec(txCtx); err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			if err := client.UserTagAssignment.Create().
				SetUserID(userID).
				SetTagID(tagID).
				Exec(txCtx); err != nil {
				return fmt.Errorf("create user tag assignment: %w", err)
			}
		}
		return nil
	})
}

func (r *userTagRepository) GetByGroupID(ctx context.Context, groupID int64) ([]service.UserTag, error) {
	client := clientFromContext(ctx, r.client)
	entities, err := client.UserTag.Query().
		Where(usertag.HasGroupsWith(group.IDEQ(groupID))).
		Order(usertag.ByName()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return userTagEntitiesToService(entities), nil
}

func (r *userTagRepository) ReplaceGroupTags(ctx context.Context, groupID int64, tagIDs []int64) error {
	return r.inTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		if _, err := client.GroupUserTag.Delete().
			Where(groupusertag.GroupIDEQ(groupID)).Exec(txCtx); err != nil {
			return err
		}
		for _, tagID := range tagIDs {
			if err := client.GroupUserTag.Create().
				SetGroupID(groupID).
				SetTagID(tagID).
				Exec(txCtx); err != nil {
				return fmt.Errorf("create group tag assignment: %w", err)
			}
		}
		return nil
	})
}

func (r *userTagRepository) GetUserIDsByTagID(ctx context.Context, tagID int64) ([]int64, error) {
	client := clientFromContext(ctx, r.client)
	assignments, err := client.UserTagAssignment.Query().
		Where(usertagassignment.TagIDEQ(tagID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, assignment.UserID)
	}
	return result, nil
}

func (r *userTagRepository) AddUsersToTag(ctx context.Context, tagID int64, userIDs []int64) ([]int64, error) {
	if len(userIDs) == 0 {
		return []int64{}, nil
	}

	added := make([]int64, 0, len(userIDs))
	err := r.inTx(ctx, func(txCtx context.Context, client *dbent.Client) error {
		rows, err := client.QueryContext(txCtx, `
			INSERT INTO user_tag_assignments (user_id, tag_id, created_at)
			SELECT input.user_id, $1, NOW()
			FROM unnest($2::bigint[]) AS input(user_id)
			ON CONFLICT (user_id, tag_id) DO NOTHING
			RETURNING user_id
		`, tagID, pq.Array(userIDs))
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var userID int64
			if err := rows.Scan(&userID); err != nil {
				return err
			}
			added = append(added, userID)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}

func (r *userTagRepository) GetGroupIDsByUserID(ctx context.Context, userID int64) ([]int64, error) {
	client := clientFromContext(ctx, r.client)
	groupIDs, err := client.UserTagAssignment.Query().
		Where(usertagassignment.UserIDEQ(userID)).
		QueryTag().
		Where(usertag.DeletedAtIsNil()).
		QueryGroups().
		Unique(true).
		Where(
			group.DeletedAtIsNil(),
			group.StatusEQ(service.StatusActive),
			group.IsExclusiveEQ(true),
			group.SubscriptionTypeEQ(service.SubscriptionTypeStandard),
		).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	return groupIDs, nil
}

func (r *userTagRepository) inTx(ctx context.Context, fn func(context.Context, *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func copyUserTag(dst *service.UserTag, src *dbent.UserTag) {
	dst.ID = src.ID
	dst.Name = src.Name
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func userTagEntityToService(entity *dbent.UserTag) *service.UserTag {
	if entity == nil {
		return nil
	}
	result := &service.UserTag{}
	copyUserTag(result, entity)
	return result
}

func userTagEntitiesToService(entities []*dbent.UserTag) []service.UserTag {
	result := make([]service.UserTag, 0, len(entities))
	for _, entity := range entities {
		result = append(result, *userTagEntityToService(entity))
	}
	return result
}
