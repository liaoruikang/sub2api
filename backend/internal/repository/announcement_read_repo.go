package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/announcementgrouppriceread"
	"github.com/Wei-Shaw/sub2api/ent/announcementread"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type announcementReadRepository struct {
	client *dbent.Client
}

func NewAnnouncementReadRepository(client *dbent.Client) service.AnnouncementReadRepository {
	return &announcementReadRepository{client: client}
}

func (r *announcementReadRepository) MarkRead(ctx context.Context, announcementID, userID int64, readAt time.Time) error {
	client := clientFromContext(ctx, r.client)
	err := client.AnnouncementRead.Create().
		SetAnnouncementID(announcementID).
		SetUserID(userID).
		SetReadAt(readAt).
		OnConflictColumns(announcementread.FieldAnnouncementID, announcementread.FieldUserID).
		DoNothing().
		Exec(ctx)
	if isSQLNoRowsError(err) {
		return nil
	}
	return err
}

func (r *announcementReadRepository) GetReadMapByUser(ctx context.Context, userID int64, announcementIDs []int64) (map[int64]time.Time, error) {
	if len(announcementIDs) == 0 {
		return map[int64]time.Time{}, nil
	}

	rows, err := r.client.AnnouncementRead.Query().
		Where(
			announcementread.UserIDEQ(userID),
			announcementread.AnnouncementIDIn(announcementIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]time.Time, len(rows))
	for i := range rows {
		out[rows[i].AnnouncementID] = rows[i].ReadAt
	}
	return out, nil
}

func (r *announcementReadRepository) GetReadMapByUsers(ctx context.Context, announcementID int64, userIDs []int64) (map[int64]time.Time, error) {
	if len(userIDs) == 0 {
		return map[int64]time.Time{}, nil
	}

	rows, err := r.client.AnnouncementRead.Query().
		Where(
			announcementread.AnnouncementIDEQ(announcementID),
			announcementread.UserIDIn(userIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]time.Time, len(rows))
	for i := range rows {
		out[rows[i].UserID] = rows[i].ReadAt
	}
	return out, nil
}

func (r *announcementReadRepository) CountByAnnouncementID(ctx context.Context, announcementID int64) (int64, error) {
	count, err := r.client.AnnouncementRead.Query().
		Where(announcementread.AnnouncementIDEQ(announcementID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *announcementReadRepository) MarkGroupPriceChangesRead(ctx context.Context, announcementID, userID int64, groupIDs []int64, readAt time.Time) error {
	if len(groupIDs) == 0 {
		return nil
	}
	client := clientFromContext(ctx, r.client)
	builders := make([]*dbent.AnnouncementGroupPriceReadCreate, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		builders = append(builders, client.AnnouncementGroupPriceRead.Create().
			SetAnnouncementID(announcementID).
			SetUserID(userID).
			SetGroupID(groupID).
			SetReadAt(readAt))
	}
	err := client.AnnouncementGroupPriceRead.CreateBulk(builders...).
		OnConflictColumns(
			announcementgrouppriceread.FieldAnnouncementID,
			announcementgrouppriceread.FieldUserID,
			announcementgrouppriceread.FieldGroupID,
		).
		DoNothing().
		Exec(ctx)
	if isSQLNoRowsError(err) {
		return nil
	}
	return err
}

func (r *announcementReadRepository) GetGroupPriceReadMapByUser(ctx context.Context, userID int64, announcementIDs []int64) (map[int64]map[int64]time.Time, error) {
	out := make(map[int64]map[int64]time.Time, len(announcementIDs))
	if len(announcementIDs) == 0 {
		return out, nil
	}
	rows, err := r.client.AnnouncementGroupPriceRead.Query().
		Where(
			announcementgrouppriceread.UserIDEQ(userID),
			announcementgrouppriceread.AnnouncementIDIn(announcementIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if out[row.AnnouncementID] == nil {
			out[row.AnnouncementID] = make(map[int64]time.Time)
		}
		out[row.AnnouncementID][row.GroupID] = row.ReadAt
	}
	return out, nil
}
