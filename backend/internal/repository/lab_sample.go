package repository

import (
	"context"

	"github.com/blueship581/water-sample-chain-assurance/backend/internal/constants"
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
	CountByBatchAndStatus(context.Context, []uint) (map[uint]map[string]int64, error)
	ListUndisposedByBatch(context.Context, uint) ([]model.LabSample, error)
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

// CountByBatchAndStatus aggregates sample totals per batch and status so the
// 采样批次 views can show how many samples sit in each state.
func (r *labSampleRepository) CountByBatchAndStatus(ctx context.Context, batchIDs []uint) (map[uint]map[string]int64, error) {
	counts := make(map[uint]map[string]int64)
	if len(batchIDs) == 0 {
		return counts, nil
	}
	type countRow struct {
		BatchID uint
		Status  string
		Total   int64
	}
	rows := make([]countRow, 0)
	if err := r.db.WithContext(ctx).Model(&model.LabSample{}).
		Select("batch_id, status, COUNT(*) AS total").
		Where("batch_id IN ?", batchIDs).
		Group("batch_id, status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, ok := counts[row.BatchID]; !ok {
			counts[row.BatchID] = make(map[string]int64)
		}
		counts[row.BatchID][row.Status] = row.Total
	}
	return counts, nil
}

// ListUndisposedByBatch returns samples of the batch that have not reached the
// disposed state; the batch cannot close while any of these remain.
func (r *labSampleRepository) ListUndisposedByBatch(ctx context.Context, batchID uint) ([]model.LabSample, error) {
	items := make([]model.LabSample, 0)
	err := r.db.WithContext(ctx).
		Where("batch_id = ? AND status <> ?", batchID, string(constants.SampleStateDisposed)).
		Order("code ASC").
		Find(&items).Error
	return items, err
}
