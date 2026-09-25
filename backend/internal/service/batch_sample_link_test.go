package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/blueship581/water-sample-chain-assurance/backend/internal/config"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/dto"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/model"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/repository"
	"gorm.io/gorm"
)

// fakeLabSampleRepository implements repository.LabSampleRepository in memory.
type fakeLabSampleRepository struct {
	items        map[uint]model.LabSample
	openByBatch  map[uint][]model.LabSample
	countsByBatch map[uint]map[string]int64
}

func (f *fakeLabSampleRepository) List(context.Context, dto.PageQuery) (repository.Page[model.LabSample], error) {
	return repository.Page[model.LabSample]{}, nil
}
func (f *fakeLabSampleRepository) Get(_ context.Context, id uint) (model.LabSample, error) {
	item, ok := f.items[id]
	if !ok {
		return model.LabSample{}, errors.New("record not found")
	}
	return item, nil
}
func (f *fakeLabSampleRepository) Create(_ context.Context, item *model.LabSample) error {
	item.ID = uint(len(f.items) + 1)
	f.items[item.ID] = *item
	return nil
}
func (f *fakeLabSampleRepository) Update(_ context.Context, id, _ uint, item *model.LabSample) error {
	f.items[id] = *item
	return nil
}
func (f *fakeLabSampleRepository) Delete(context.Context, uint) error { return nil }
func (f *fakeLabSampleRepository) CountByStatus(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (f *fakeLabSampleRepository) CountByBatchAndStatus(context.Context) (map[uint]map[string]int64, error) {
	return f.countsByBatch, nil
}
func (f *fakeLabSampleRepository) OpenSamplesByBatch(_ context.Context, batchID uint, _ int) ([]model.LabSample, error) {
	return f.openByBatch[batchID], nil
}

// fakeSamplingBatchRepository implements repository.SamplingBatchRepository in memory.
type fakeSamplingBatchRepository struct {
	items map[uint]model.SamplingBatch
}

func (f *fakeSamplingBatchRepository) List(context.Context, dto.PageQuery) (repository.Page[model.SamplingBatch], error) {
	return repository.Page[model.SamplingBatch]{}, nil
}
func (f *fakeSamplingBatchRepository) Get(_ context.Context, id uint) (model.SamplingBatch, error) {
	item, ok := f.items[id]
	if !ok {
		return model.SamplingBatch{}, gorm.ErrRecordNotFound
	}
	return item, nil
}
func (f *fakeSamplingBatchRepository) Create(context.Context, *model.SamplingBatch) error { return nil }
func (f *fakeSamplingBatchRepository) Update(_ context.Context, id, _ uint, item *model.SamplingBatch) error {
	f.items[id] = *item
	return nil
}
func (f *fakeSamplingBatchRepository) Delete(context.Context, uint) error { return nil }
func (f *fakeSamplingBatchRepository) CountByStatus(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// noopSecurity satisfies SecurityService without writing audit rows.
type noopSecurity struct{}

func (noopSecurity) Login(context.Context, dto.LoginRequest) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}
func (noopSecurity) Audit(context.Context, string, string, string, string, uint, string, string, string) error {
	return nil
}
func (noopSecurity) ListAudits(context.Context, int, int, string) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}
func (noopSecurity) AuditSummary(context.Context, time.Duration) (model.AuditSummary, error) {
	return model.AuditSummary{}, nil
}
func (noopSecurity) EntityHistory(context.Context, string, uint, int) ([]model.AuditLog, error) {
	return nil, nil
}
func (noopSecurity) RuntimeConfig() config.PublicConfig { return config.PublicConfig{} }

func newSampleFixture(status string) model.LabSample {
	return model.LabSample{
		BaseModel: model.BaseModel{ID: 7, Code: "LS-100", Name: "链接样本", Status: status, Version: 1},
		BatchID:   3, HandoverBy: "张三",
	}
}

func TestSampleCannotStartTestingWhileBatchNotReceived(t *testing.T) {
	samples := &fakeLabSampleRepository{items: map[uint]model.LabSample{7: newSampleFixture("accepted")}}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		3: {BaseModel: model.BaseModel{ID: 3, Code: "SB-100", Status: "collecting", Version: 1}},
	}}
	service := NewLabSampleService(samples, batches, noopSecurity{})

	_, err := service.Transition(context.Background(), 7, dto.TransitionRequest{Status: "testing", ExpectedVersion: 1, Reason: "开始检测"}, "operator", "req-1")
	if !errors.Is(err, ErrBatchNotReceived) {
		t.Fatalf("expected ErrBatchNotReceived for collecting batch, got %v", err)
	}
	if !strings.Contains(err.Error(), "SB-100") || !strings.Contains(err.Error(), "collecting") {
		t.Fatalf("error should name the blocking batch and its status, got %v", err)
	}
	if samples.items[7].Status != "accepted" {
		t.Fatalf("sample status must stay accepted, got %s", samples.items[7].Status)
	}
}

func TestSampleStartsTestingOnceBatchReceived(t *testing.T) {
	samples := &fakeLabSampleRepository{items: map[uint]model.LabSample{7: newSampleFixture("accepted")}}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		3: {BaseModel: model.BaseModel{ID: 3, Code: "SB-100", Status: "received", Version: 1}},
	}}
	service := NewLabSampleService(samples, batches, noopSecurity{})

	updated, err := service.Transition(context.Background(), 7, dto.TransitionRequest{Status: "testing", ExpectedVersion: 1, Reason: "开始检测"}, "operator", "req-2")
	if err != nil {
		t.Fatalf("expected testing to start once batch is received, got %v", err)
	}
	if updated.Status != "testing" {
		t.Fatalf("expected testing status, got %s", updated.Status)
	}
}

func TestCreateSampleRequiresExistingBatch(t *testing.T) {
	samples := &fakeLabSampleRepository{items: map[uint]model.LabSample{}}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{}}
	service := NewLabSampleService(samples, batches, noopSecurity{})

	input := dto.CreateLabSample{
		Code: "LS-200", Name: "无批次样本", Facility: "一区", Owner: "运行一组",
		Category: "常规", RiskLevel: "low", EffectiveAt: time.Now().UTC(),
		BatchID: 99, HandoverBy: "张三",
	}
	if _, err := service.Create(context.Background(), input, "operator", "req-3"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when batch is missing, got %v", err)
	}

	batches.items[99] = model.SamplingBatch{BaseModel: model.BaseModel{ID: 99, Code: "SB-200", Status: "planned", Version: 1}}
	created, err := service.Create(context.Background(), input, "operator", "req-4")
	if err != nil {
		t.Fatalf("expected create to succeed with existing batch, got %v", err)
	}
	if created.BatchID != 99 || created.HandoverBy != "张三" {
		t.Fatalf("expected batch and handover to be recorded, got %+v", created)
	}
}

func TestBatchCannotCloseWithOpenSamples(t *testing.T) {
	open := []model.LabSample{
		{BaseModel: model.BaseModel{ID: 11, Code: "LS-101", Status: "testing"}},
		{BaseModel: model.BaseModel{ID: 12, Code: "LS-102", Status: "hold"}},
	}
	samples := &fakeLabSampleRepository{
		items:       map[uint]model.LabSample{},
		openByBatch: map[uint][]model.LabSample{5: open},
	}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		5: {BaseModel: model.BaseModel{ID: 5, Code: "SB-500", Status: "received", Version: 1}},
	}}
	service := NewSamplingBatchService(batches, samples, noopSecurity{})

	_, err := service.Transition(context.Background(), 5, dto.TransitionRequest{Status: "closed", ExpectedVersion: 1, Reason: "批次收尾"}, "operator", "req-5")
	if !errors.Is(err, ErrBatchHasOpenSamples) {
		t.Fatalf("expected ErrBatchHasOpenSamples, got %v", err)
	}
	message := err.Error()
	for _, want := range []string{"LS-101(testing)", "LS-102(hold)", "SB-500"} {
		if !strings.Contains(message, want) {
			t.Fatalf("close error should list blocking samples, want %q in %q", want, message)
		}
	}
	if batches.items[5].Status != "received" {
		t.Fatalf("batch status must stay received, got %s", batches.items[5].Status)
	}
}

func TestBatchClosesWhenAllSamplesDisposed(t *testing.T) {
	samples := &fakeLabSampleRepository{
		items:       map[uint]model.LabSample{},
		openByBatch: map[uint][]model.LabSample{},
	}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		5: {BaseModel: model.BaseModel{ID: 5, Code: "SB-500", Status: "received", Version: 1}},
	}}
	service := NewSamplingBatchService(batches, samples, noopSecurity{})

	updated, err := service.Transition(context.Background(), 5, dto.TransitionRequest{Status: "closed", ExpectedVersion: 1, Reason: "全部样本已处置"}, "operator", "req-6")
	if err != nil {
		t.Fatalf("expected close to succeed without open samples, got %v", err)
	}
	if updated.Status != "closed" {
		t.Fatalf("expected closed status, got %s", updated.Status)
	}
}
