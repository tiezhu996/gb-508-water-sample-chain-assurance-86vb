package repository

import (
	"context"

	"github.com/blueship581/water-sample-chain-assurance/backend/internal/dto"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/model"
	"gorm.io/gorm"
)

// SamplingBatchRepository owns all persistence operations for 采样批次.
type SamplingBatchRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.SamplingBatch], error)
	Get(context.Context, uint) (model.SamplingBatch, error)
	ListByIDs(context.Context, []uint) ([]model.SamplingBatch, error)
	Create(context.Context, *model.SamplingBatch) error
	Update(context.Context, uint, uint, *model.SamplingBatch) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type samplingBatchRepository struct {
	store *Store[model.SamplingBatch]
	db    *gorm.DB
}

func NewSamplingBatchRepository(db *gorm.DB) SamplingBatchRepository {
	return &samplingBatchRepository{store: NewStore[model.SamplingBatch](db), db: db}
}

func (r *samplingBatchRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.SamplingBatch], error) {
	return r.store.List(ctx, q)
}
func (r *samplingBatchRepository) Get(ctx context.Context, id uint) (model.SamplingBatch, error) {
	return r.store.Get(ctx, id)
}

// ListByIDs loads batches in one query so sample views can resolve batch codes
// without per-row lookups.
func (r *samplingBatchRepository) ListByIDs(ctx context.Context, ids []uint) ([]model.SamplingBatch, error) {
	items := make([]model.SamplingBatch, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}
func (r *samplingBatchRepository) Create(ctx context.Context, item *model.SamplingBatch) error {
	return r.store.Create(ctx, item)
}
func (r *samplingBatchRepository) Update(ctx context.Context, id, version uint, item *model.SamplingBatch) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *samplingBatchRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *samplingBatchRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
