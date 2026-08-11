package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func newUserTagRepositoryTestClient(t *testing.T) (*userTagRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	return &userTagRepository{client: client}, mock
}

func TestUserTagRepositoryGetGroupIDsByUserIDScansGroupRows(t *testing.T) {
	repo, mock := newUserTagRepositoryTestClient(t)

	mock.ExpectQuery(`SELECT .* FROM "groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(25)))

	groupIDs, err := repo.GetGroupIDsByUserID(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, []int64{25}, groupIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserTagRepositoryAddUsersToTagReturnsOnlyInsertedRows(t *testing.T) {
	repo, mock := newUserTagRepositoryTestClient(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO user_tag_assignments`).
		WithArgs(int64(9), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(3)).AddRow(int64(7)))
	mock.ExpectCommit()

	added, err := repo.AddUsersToTag(context.Background(), 9, []int64{3, 7, 11})

	require.NoError(t, err)
	require.Equal(t, []int64{3, 7}, added)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserTagRepositoryAddUsersToTagReturnsEmptyWhenNoConflictFreeRows(t *testing.T) {
	repo, mock := newUserTagRepositoryTestClient(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO user_tag_assignments`).
		WithArgs(int64(9), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectCommit()

	added, err := repo.AddUsersToTag(context.Background(), 9, []int64{3})

	require.NoError(t, err)
	require.Empty(t, added)
	require.NoError(t, mock.ExpectationsWereMet())
}
