package repository

import (
	"errors"

	"kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StateRepository owns kungal_user_state — the slim local table that holds
// kungal-specific business fields (moemoepoint / daily counters). Identity
// fields (name / avatar / email / bio / status / role) are owned by OAuth and
// must be fetched via pkg/userclient. user_id here = OAuth user.id.
type StateRepository struct {
	db *gorm.DB
}

func NewStateRepository(db *gorm.DB) *StateRepository {
	return &StateRepository{db: db}
}

func (r *StateRepository) DB() *gorm.DB { return r.db }

// Ensure idempotently creates the row for a freshly-seen user. Called from
// the OAuth callback so newly-onboarded users start with the default
// moemoepoint balance and zeroed daily counters.
func (r *StateRepository) Ensure(uid int) error {
	if uid <= 0 {
		return errors.New("invalid uid")
	}
	row := model.KungalUserState{UserID: uid, Moemoepoint: 7}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

// FindByID returns the state row or sql.ErrNoRows if missing.
func (r *StateRepository) FindByID(uid int) (*model.KungalUserState, error) {
	var s model.KungalUserState
	err := r.db.First(&s, "user_id = ?", uid).Error
	return &s, err
}

// IncrementMoemoepoint adds delta (may be negative) to the user's balance.
func (r *StateRepository) IncrementMoemoepoint(uid, delta int) error {
	if delta == 0 || uid <= 0 {
		return nil
	}
	return r.db.Model(&model.KungalUserState{}).
		Where("user_id = ?", uid).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", delta)).Error
}

// AdjustMoemoepointTx is a tx-scoped variant of IncrementMoemoepoint used by
// callers that need atomicity with surrounding writes (likes, dislikes, etc.).
// No-op when uid<=0 or delta==0.
func (r *StateRepository) AdjustMoemoepointTx(tx *gorm.DB, uid, delta int) error {
	if uid <= 0 || delta == 0 {
		return nil
	}
	return tx.Model(&model.KungalUserState{}).
		Where("user_id = ?", uid).
		Update("moemoepoint", gorm.Expr("moemoepoint + ?", delta)).Error
}

// LockForUpdate acquires a SELECT ... FOR UPDATE lock on the state row, used
// by interaction paths that read-then-write moemoepoint inside a tx. Replaces
// the old UserRepository.LockUserForUpdate that locked the obsolete user table.
func (r *StateRepository) LockForUpdate(tx *gorm.DB, uid int) (*model.KungalUserState, error) {
	var s model.KungalUserState
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", uid).
		First(&s).Error
	return &s, err
}

// CheckIn flips daily_check_in to 1 and grants `points` moemoepoint in a
// single UPDATE. Caller is responsible for rate-limit / once-per-day logic
// before calling.
func (r *StateRepository) CheckIn(uid, points int) error {
	return r.db.Model(&model.KungalUserState{}).Where("user_id = ?", uid).
		Updates(map[string]any{
			"daily_check_in": 1,
			"moemoepoint":    gorm.Expr("moemoepoint + ?", points),
		}).Error
}

// IncrementDailyCounter bumps a single daily_* column by 1; used by image /
// toolset upload paths.
func (r *StateRepository) IncrementDailyCounter(uid int, column string) error {
	return r.db.Model(&model.KungalUserState{}).Where("user_id = ?", uid).
		Update(column, gorm.Expr(column+" + 1")).Error
}

// ResetDailyCounters zeros all per-day fields. Run by the daily cron at
// midnight (cron/cron.go), replacing the old UPDATE "user" SET daily_*
// query that touched the obsolete identity table.
func (r *StateRepository) ResetDailyCounters() (int64, error) {
	res := r.db.Exec(`
		UPDATE kungal_user_state SET
			daily_check_in = 0,
			daily_image_count = 0,
			daily_toolset_upload_count = 0
		WHERE daily_check_in != 0
		   OR daily_image_count != 0
		   OR daily_toolset_upload_count != 0
	`)
	return res.RowsAffected, res.Error
}
