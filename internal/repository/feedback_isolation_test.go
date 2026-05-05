// File: internal/repository/feedback_isolation_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/azharf99/algo-feedback/internal/repository"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupFeedbackMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestFeedbackRepository_DataIsolation(t *testing.T) {
	t.Run("User A should only see their own data", func(t *testing.T) {
		gormDB, mock := setupFeedbackMockDB(t)
		repo := repository.NewFeedbackRepository(gormDB)

		// Arrange
		userID := uint(1)
		ctx := ctxutil.WithUserID(context.Background(), userID)
		
		// Expect query to include WHERE user_id = 1
		mock.ExpectQuery(`SELECT \* FROM "feedbacks" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(101, userID))

		// Act
		feedbacks, err := repo.GetAll(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, feedbacks, 1)
		assert.Equal(t, userID, feedbacks[0].UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User B should not be able to get User A's data by ID", func(t *testing.T) {
		gormDB, mock := setupFeedbackMockDB(t)
		repo := repository.NewFeedbackRepository(gormDB)

		// Arrange
		userB_ID := uint(2)
		ctx := ctxutil.WithUserID(context.Background(), userB_ID)
		feedbackA_ID := uint(101)

		// GORM First adds ORDER BY and LIMIT. Also handle Preload if necessary, but we can just expect the primary query.
		// Expect query: SELECT * FROM "feedbacks" WHERE "feedbacks"."id" = $1 AND user_id = $2 ORDER BY "feedbacks"."id" LIMIT 1
		mock.ExpectQuery(`SELECT \* FROM "feedbacks" WHERE "feedbacks"\."id" = \$1 AND user_id = \$2 ORDER BY "feedbacks"\."id" LIMIT \$3`).
			WithArgs(feedbackA_ID, userB_ID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"})) // Empty result

		// Act
		_, err := repo.GetByID(ctx, feedbackA_ID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User B should not be able to delete User A's data", func(t *testing.T) {
		gormDB, mock := setupFeedbackMockDB(t)
		repo := repository.NewFeedbackRepository(gormDB)

		// Arrange
		userB_ID := uint(2)
		ctx := ctxutil.WithUserID(context.Background(), userB_ID)
		feedbackA_ID := uint(101)

		// GORM wraps Delete in a transaction
		mock.ExpectBegin()
		// DELETE FROM "feedbacks" WHERE "feedbacks"."id" = $1 AND user_id = $2
		mock.ExpectExec(`DELETE FROM "feedbacks" WHERE "feedbacks"\."id" = \$1 AND user_id = \$2`).
			WithArgs(feedbackA_ID, userB_ID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected
		mock.ExpectCommit()

		// Act
		err := repo.Delete(ctx, feedbackA_ID)

		// Assert
		assert.NoError(t, err) // GORM Delete returns no error if 0 rows affected unless we use certain settings
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Admin should be able to see all data (bypass isolation)", func(t *testing.T) {
		gormDB, mock := setupFeedbackMockDB(t)
		repo := repository.NewFeedbackRepository(gormDB)

		// Arrange
		ctx := ctxutil.WithRole(context.Background(), "Admin")
		
		// Expect query WITHOUT user_id filter
		mock.ExpectQuery(`SELECT \* FROM "feedbacks"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(101, 1).AddRow(102, 2))

		// Act
		feedbacks, err := repo.GetAll(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, feedbacks, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

