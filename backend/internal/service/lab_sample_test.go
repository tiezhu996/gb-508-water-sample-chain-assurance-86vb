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
)

// fakeLabSampleRepository is an in-memory LabSampleRepository for rule tests.
type fakeLabSampleRepository struct {
	items      map[uint]model.LabSample
	undisposed []model.LabSample
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
func (f *fakeLabSampleRepository) Delete(_ context.Context, id uint) error {
	delete(f.items, id)
	return nil
}
func (f *fakeLabSampleRepository) CountByStatus(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (f *fakeLabSampleRepository) CountByBatchAndStatus(_ context.Context, ids []uint) (map[uint]map[string]int64, error) {
	counts := make(map[uint]map[string]int64)
	for _, id := range ids {
		counts[id] = map[string]int64{}
		for _, sample := range f.undisposed {
			if sample.BatchID == id {
				counts[id][sample.Status]++
			}
		}
	}
	return counts, nil
}
func (f *fakeLabSampleRepository) ListUndisposedByBatch(_ context.Context, batchID uint) ([]model.LabSample, error) {
	items := make([]model.LabSample, 0)
	for _, sample := range f.undisposed {
		if sample.BatchID == batchID {
			items = append(items, sample)
		}
	}
	return items, nil
}

// fakeSamplingBatchRepository is an in-memory SamplingBatchRepository for rule tests.
type fakeSamplingBatchRepository struct {
	items map[uint]model.SamplingBatch
}

func (f *fakeSamplingBatchRepository) List(context.Context, dto.PageQuery) (repository.Page[model.SamplingBatch], error) {
	return repository.Page[model.SamplingBatch]{}, nil
}
func (f *fakeSamplingBatchRepository) Get(_ context.Context, id uint) (model.SamplingBatch, error) {
	item, ok := f.items[id]
	if !ok {
		return model.SamplingBatch{}, errors.New("record not found")
	}
	return item, nil
}
func (f *fakeSamplingBatchRepository) ListByIDs(_ context.Context, ids []uint) ([]model.SamplingBatch, error) {
	items := make([]model.SamplingBatch, 0, len(ids))
	for _, id := range ids {
		if item, ok := f.items[id]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}
func (f *fakeSamplingBatchRepository) Create(_ context.Context, item *model.SamplingBatch) error {
	item.ID = uint(len(f.items) + 1)
	f.items[item.ID] = *item
	return nil
}
func (f *fakeSamplingBatchRepository) Update(_ context.Context, id, _ uint, item *model.SamplingBatch) error {
	f.items[id] = *item
	return nil
}
func (f *fakeSamplingBatchRepository) Delete(_ context.Context, id uint) error {
	delete(f.items, id)
	return nil
}
func (f *fakeSamplingBatchRepository) CountByStatus(context.Context) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// fakeSecurityService satisfies SecurityService without persistence.
type fakeSecurityService struct{}

func (fakeSecurityService) Login(context.Context, dto.LoginRequest) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}
func (fakeSecurityService) Audit(context.Context, string, string, string, string, uint, string, string, string) error {
	return nil
}
func (fakeSecurityService) ListAudits(context.Context, int, int, string) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}
func (fakeSecurityService) AuditSummary(context.Context, time.Duration) (model.AuditSummary, error) {
	return model.AuditSummary{}, nil
}
func (fakeSecurityService) EntityHistory(context.Context, string, uint, int) ([]model.AuditLog, error) {
	return nil, nil
}
func (fakeSecurityService) RuntimeConfig() config.PublicConfig { return config.PublicConfig{} }

func transitionRequest(status string) dto.TransitionRequest {
	return dto.TransitionRequest{Status: status, ExpectedVersion: 1, Reason: "规则测试"}
}

func TestLabSampleCannotStartTestingWhileBatchNotReceived(t *testing.T) {
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		7: {BaseModel: model.BaseModel{ID: 7, Code: "SB-007", Status: "collecting", Version: 1}},
	}}
	samples := &fakeLabSampleRepository{items: map[uint]model.LabSample{
		3: {BaseModel: model.BaseModel{ID: 3, Code: "LS-003", Status: "accepted", Version: 1}, BatchID: 7},
	}}
	service := NewLabSampleService(samples, batches, fakeSecurityService{})

	_, err := service.Transition(context.Background(), 3, transitionRequest("testing"), "operator", "req-1")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected testing to be blocked while batch is collecting, got %v", err)
	}
	if !strings.Contains(err.Error(), "SB-007") {
		t.Fatalf("expected the error to name the blocking batch, got %v", err)
	}

	batches.items[7] = model.SamplingBatch{BaseModel: model.BaseModel{ID: 7, Code: "SB-007", Status: "received", Version: 1}}
	updated, err := service.Transition(context.Background(), 3, transitionRequest("testing"), "operator", "req-2")
	if err != nil {
		t.Fatalf("expected testing to be allowed once the batch is received: %v", err)
	}
	if updated.Status != "testing" {
		t.Fatalf("expected sample to reach testing, got %s", updated.Status)
	}
}

func TestLabSampleCreateRegistersBatchAndHandover(t *testing.T) {
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		7: {BaseModel: model.BaseModel{ID: 7, Code: "SB-007", Status: "collecting", Version: 1}},
	}}
	samples := &fakeLabSampleRepository{items: map[uint]model.LabSample{}}
	service := NewLabSampleService(samples, batches, fakeSecurityService{})

	input := dto.CreateLabSample{
		Code: "ls-100", Name: "新收样本", Facility: "一区", Owner: "运行一组", Category: "常规",
		RiskLevel: "low", EffectiveAt: time.Now().UTC(), BatchID: 7, HandoverBy: "交接员·王敏",
	}
	created, err := service.Create(context.Background(), input, "operator", "req-3")
	if err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}
	if created.BatchID != 7 || created.BatchCode != "SB-007" || created.HandoverBy != "交接员·王敏" {
		t.Fatalf("expected batch and handover to be registered, got %+v", created)
	}

	input.Code = "ls-101"
	input.HandoverBy = "  "
	if _, err := service.Create(context.Background(), input, "operator", "req-4"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected missing handover person to fail validation, got %v", err)
	}
}

func TestSamplingBatchCloseBlockedByUndisposedSamples(t *testing.T) {
	samples := &fakeLabSampleRepository{
		items: map[uint]model.LabSample{},
		undisposed: []model.LabSample{
			{BaseModel: model.BaseModel{Code: "LS-001", Status: "testing"}, BatchID: 5},
			{BaseModel: model.BaseModel{Code: "LS-002", Status: "hold"}, BatchID: 5},
		},
	}
	batches := &fakeSamplingBatchRepository{items: map[uint]model.SamplingBatch{
		5: {BaseModel: model.BaseModel{ID: 5, Code: "SB-005", Status: "received", Version: 1}},
	}}
	service := NewSamplingBatchService(batches, samples, fakeSecurityService{})

	_, err := service.Transition(context.Background(), 5, transitionRequest("closed"), "operator", "req-5")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected closing to be blocked by undisposed samples, got %v", err)
	}
	for _, code := range []string{"LS-001", "LS-002"} {
		if !strings.Contains(err.Error(), code) {
			t.Fatalf("expected the error to name blocking sample %s, got %v", code, err)
		}
	}

	samples.undisposed = nil
	updated, err := service.Transition(context.Background(), 5, transitionRequest("closed"), "operator", "req-6")
	if err != nil {
		t.Fatalf("expected closing to succeed once samples are disposed: %v", err)
	}
	if updated.Status != "closed" {
		t.Fatalf("expected batch to close, got %s", updated.Status)
	}
}
