package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/announcement"
	"github.com/Wei-Shaw/sub2api/ent/announcementgrouppricechange"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type announcementRepository struct {
	client *dbent.Client
}

func NewAnnouncementRepository(client *dbent.Client) service.AnnouncementRepository {
	return &announcementRepository{client: client}
}

func (r *announcementRepository) Create(ctx context.Context, a *service.Announcement) error {
	client := clientFromContext(ctx, r.client)
	return createAnnouncement(ctx, client, a)
}

func createAnnouncement(ctx context.Context, client *dbent.Client, a *service.Announcement) error {
	if a.Kind == "" {
		a.Kind = service.AnnouncementKindManual
	}
	builder := client.Announcement.Create().
		SetKind(a.Kind).
		SetTitle(a.Title).
		SetContent(a.Content).
		SetStatus(a.Status).
		SetNotifyMode(a.NotifyMode).
		SetTargeting(a.Targeting)

	if a.StartsAt != nil {
		builder.SetStartsAt(*a.StartsAt)
	}
	if a.EndsAt != nil {
		builder.SetEndsAt(*a.EndsAt)
	}
	if a.CreatedBy != nil {
		builder.SetCreatedBy(*a.CreatedBy)
	}
	if a.UpdatedBy != nil {
		builder.SetUpdatedBy(*a.UpdatedBy)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	applyAnnouncementEntityToService(a, created)
	return nil
}

func (r *announcementRepository) CreateWithGroupPriceChanges(ctx context.Context, a *service.Announcement, changes []service.AnnouncementGroupPriceChange) error {
	if len(changes) == 0 {
		return fmt.Errorf("group price announcement requires changes")
	}
	if dbent.TxFromContext(ctx) != nil {
		return createAnnouncementWithGroupPriceChanges(ctx, clientFromContext(ctx, r.client), a, changes)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := createAnnouncementWithGroupPriceChanges(txCtx, tx.Client(), a, changes); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func createAnnouncementWithGroupPriceChanges(ctx context.Context, client *dbent.Client, a *service.Announcement, changes []service.AnnouncementGroupPriceChange) error {
	if err := createAnnouncement(ctx, client, a); err != nil {
		return err
	}
	builders := make([]*dbent.AnnouncementGroupPriceChangeCreate, 0, len(changes))
	for i := range changes {
		change := changes[i]
		builders = append(builders, client.AnnouncementGroupPriceChange.Create().
			SetAnnouncementID(a.ID).
			SetGroupID(change.GroupID).
			SetGroupName(change.GroupName).
			SetOldRate(change.OldRate).
			SetNewRate(change.NewRate).
			SetSequence(change.Sequence))
	}
	if _, err := client.AnnouncementGroupPriceChange.CreateBulk(builders...).Save(ctx); err != nil {
		return err
	}
	return nil
}

func (r *announcementRepository) GetByID(ctx context.Context, id int64) (*service.Announcement, error) {
	m, err := r.client.Announcement.Query().
		Where(announcement.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrAnnouncementNotFound, nil)
	}
	return announcementEntityToService(m), nil
}

func (r *announcementRepository) Update(ctx context.Context, a *service.Announcement) error {
	client := clientFromContext(ctx, r.client)
	if a.Kind == "" {
		a.Kind = service.AnnouncementKindManual
	}
	builder := client.Announcement.UpdateOneID(a.ID).
		SetKind(a.Kind).
		SetTitle(a.Title).
		SetContent(a.Content).
		SetStatus(a.Status).
		SetNotifyMode(a.NotifyMode).
		SetTargeting(a.Targeting)

	if a.StartsAt != nil {
		builder.SetStartsAt(*a.StartsAt)
	} else {
		builder.ClearStartsAt()
	}
	if a.EndsAt != nil {
		builder.SetEndsAt(*a.EndsAt)
	} else {
		builder.ClearEndsAt()
	}
	if a.CreatedBy != nil {
		builder.SetCreatedBy(*a.CreatedBy)
	} else {
		builder.ClearCreatedBy()
	}
	if a.UpdatedBy != nil {
		builder.SetUpdatedBy(*a.UpdatedBy)
	} else {
		builder.ClearUpdatedBy()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrAnnouncementNotFound, nil)
	}

	a.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *announcementRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.Announcement.Delete().Where(announcement.IDEQ(id)).Exec(ctx)
	return err
}

func (r *announcementRepository) List(
	ctx context.Context,
	params pagination.PaginationParams,
	filters service.AnnouncementListFilters,
) ([]service.Announcement, *pagination.PaginationResult, error) {
	q := r.client.Announcement.Query()

	if filters.Status != "" {
		q = q.Where(announcement.StatusEQ(filters.Status))
	}
	if filters.Search != "" {
		q = q.Where(
			announcement.Or(
				announcement.TitleContainsFold(filters.Search),
				announcement.ContentContainsFold(filters.Search),
			),
		)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.
		Offset(params.Offset()).
		Limit(params.Limit())
	for _, order := range announcementListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := announcementEntitiesToService(items)
	return out, paginationResultFromTotal(int64(total), params), nil
}

func announcementListOrder(params pagination.PaginationParams) (string, string) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	switch sortBy {
	case "title":
		return announcement.FieldTitle, sortOrder
	case "status":
		return announcement.FieldStatus, sortOrder
	case "notify_mode":
		return announcement.FieldNotifyMode, sortOrder
	case "starts_at":
		return announcement.FieldStartsAt, sortOrder
	case "ends_at":
		return announcement.FieldEndsAt, sortOrder
	case "id":
		return announcement.FieldID, sortOrder
	case "", "created_at":
		return announcement.FieldCreatedAt, sortOrder
	default:
		return announcement.FieldCreatedAt, pagination.SortOrderDesc
	}
}

func announcementListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	field, sortOrder := announcementListOrder(params)

	if sortOrder == pagination.SortOrderAsc {
		if field == announcement.FieldID {
			return []func(*entsql.Selector){
				dbent.Asc(field),
			}
		}
		return []func(*entsql.Selector){
			dbent.Asc(field),
			dbent.Asc(announcement.FieldID),
		}
	}

	if field == announcement.FieldID {
		return []func(*entsql.Selector){
			dbent.Desc(field),
		}
	}
	return []func(*entsql.Selector){
		dbent.Desc(field),
		dbent.Desc(announcement.FieldID),
	}
}

func (r *announcementRepository) ListActive(ctx context.Context, now time.Time) ([]service.Announcement, error) {
	return r.ListActivePage(ctx, now, 0, 200)
}

func (r *announcementRepository) ListActivePage(ctx context.Context, now time.Time, beforeID int64, limit int) ([]service.Announcement, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	q := r.client.Announcement.Query().
		Where(
			announcement.StatusEQ(service.AnnouncementStatusActive),
			announcement.Or(announcement.StartsAtIsNil(), announcement.StartsAtLTE(now)),
			announcement.Or(announcement.EndsAtIsNil(), announcement.EndsAtGT(now)),
		)
	if beforeID > 0 {
		q = q.Where(announcement.IDLT(beforeID))
	}
	q = q.Order(dbent.Desc(announcement.FieldID)).Limit(limit)

	items, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return announcementEntitiesToService(items), nil
}

func (r *announcementRepository) ListGroupPriceChanges(ctx context.Context, announcementIDs []int64) (map[int64][]service.AnnouncementGroupPriceChange, error) {
	out := make(map[int64][]service.AnnouncementGroupPriceChange, len(announcementIDs))
	if len(announcementIDs) == 0 {
		return out, nil
	}
	items, err := r.client.AnnouncementGroupPriceChange.Query().
		Where(announcementgrouppricechange.AnnouncementIDIn(announcementIDs...)).
		Order(dbent.Asc(announcementgrouppricechange.FieldAnnouncementID), dbent.Asc(announcementgrouppricechange.FieldSequence)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out[item.AnnouncementID] = append(out[item.AnnouncementID], service.AnnouncementGroupPriceChange{
			ID: item.ID, AnnouncementID: item.AnnouncementID, GroupID: item.GroupID,
			GroupName: item.GroupName, OldRate: item.OldRate, NewRate: item.NewRate, Sequence: item.Sequence,
		})
	}
	return out, nil
}

func applyAnnouncementEntityToService(dst *service.Announcement, src *dbent.Announcement) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func announcementEntityToService(m *dbent.Announcement) *service.Announcement {
	if m == nil {
		return nil
	}
	return &service.Announcement{
		ID:         m.ID,
		Kind:       m.Kind,
		Title:      m.Title,
		Content:    m.Content,
		Status:     m.Status,
		NotifyMode: m.NotifyMode,
		Targeting:  m.Targeting,
		StartsAt:   m.StartsAt,
		EndsAt:     m.EndsAt,
		CreatedBy:  m.CreatedBy,
		UpdatedBy:  m.UpdatedBy,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func announcementEntitiesToService(models []*dbent.Announcement) []service.Announcement {
	out := make([]service.Announcement, 0, len(models))
	for i := range models {
		if s := announcementEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
