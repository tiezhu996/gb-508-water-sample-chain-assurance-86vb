package repository

import (
	"context"

	"github.com/blueship581/water-sample-chain-assurance/backend/internal/dto"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/model"
	"gorm.io/gorm"
)

// LabSampleRepository owns all persistence operations for 实验室样本.
type LabSampleRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.LabSample], error)
	Get(context.Context, uint) (model.LabSample, error)
	Create(context.Context, *model.LabSample) error
	Update(context.Context, uint, uint, *model.LabSample) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
	// CountByBatchAndStatus groups sample counts per batch and per sample status.
	CountByBatchAndStatus(context.Context) (map[uint]map[string]int64, error)
	// OpenSamplesByBatch returns samples under the batch that are not disposed yet.
	OpenSamplesByBatch(context.Context, uint, int) ([]model.LabSample, error)
}

type labSampleRepository struct {
	store *Store[model.LabSample]
	db    *gorm.DB
}

func NewLabSampleRepository(db *gorm.DB) LabSampleRepository {
	return &labSampleRepository{store: NewStore[model.LabSample](db), db: db}
}

func (r *labSampleRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.LabSample], error) {
	return r.store.List(ctx, q)
}
func (r *labSampleRepository) Get(ctx context.Context, id uint) (model.LabSample, error) {
	return r.store.Get(ctx, id)
}
func (r *labSampleRepository) Create(ctx context.Context, item *model.LabSample) error {
	return r.store.Create(ctx, item)
}
func (r *labSampleRepository) Update(ctx context.Context, id, version uint, item *model.LabSample) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *labSampleRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *labSampleRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}

func (r *labSampleRepository) CountByBatchAndStatus(ctx context.Context) (map[uint]map[string]int64, error) {
	rows, err := r.db.WithContext(ctx).Model(&model.LabSample{}).
		Select("batch_id, status, COUNT(*) AS total").Group("batch_id, status").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[uint]map[string]int64)
	for rows.Next() {
		var batchID uint
		var status string
		var total int64
		if err := rows.Scan(&batchID, &status, &total); err != nil {
			return nil, err
		}
		if counts[batchID] == nil {
			counts[batchID] = make(map[string]int64)
		}
		counts[batchID][status] = total
	}
	return counts, rows.Err()
}

func (r *labSampleRepository) OpenSamplesByBatch(ctx context.Context, batchID uint, limit int) ([]model.LabSample, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items := make([]model.LabSample, 0)
	err := r.db.WithContext(ctx).Model(&model.LabSample{}).
		Where("batch_id = ? AND status <> ?", batchID, model.LabSampleDisposedStatus).
		Order("id ASC").Limit(limit).Find(&items).Error
	return items, err
}
