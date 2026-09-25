package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/water-sample-chain-assurance/backend/internal/constants"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/dto"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/model"
	"github.com/blueship581/water-sample-chain-assurance/backend/internal/repository"
)

type LabSampleService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.LabSample], error)
	Get(context.Context, uint) (model.LabSample, error)
	Create(context.Context, dto.CreateLabSample, string, string) (model.LabSample, error)
	Update(context.Context, uint, dto.UpdateLabSample, string, string) (model.LabSample, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.LabSample, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type labSampleService struct {
	repository repository.LabSampleRepository
	batches    repository.SamplingBatchRepository
	security   SecurityService
}

func NewLabSampleService(repo repository.LabSampleRepository, batches repository.SamplingBatchRepository, security SecurityService) LabSampleService {
	return &labSampleService{repository: repo, batches: batches, security: security}
}

func (s *labSampleService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.LabSample], error) {
	page, err := s.repository.List(ctx, query)
	if err != nil {
		return page, err
	}
	return page, s.fillBatchCodes(ctx, page.Items)
}

func (s *labSampleService) Get(ctx context.Context, id uint) (model.LabSample, error) {
	item, err := s.repository.Get(ctx, id)
	if err != nil {
		return item, err
	}
	items := []model.LabSample{item}
	if err := s.fillBatchCodes(ctx, items); err != nil {
		return model.LabSample{}, err
	}
	return items[0], nil
}

// fillBatchCodes resolves the denormalized batch code for display, mutating the
// slice in place so list and detail responses share one lookup path.
func (s *labSampleService) fillBatchCodes(ctx context.Context, items []model.LabSample) error {
	ids := make([]uint, 0, len(items))
	seen := make(map[uint]bool)
	for _, item := range items {
		if item.BatchID != 0 && !seen[item.BatchID] {
			seen[item.BatchID] = true
			ids = append(ids, item.BatchID)
		}
	}
	batches, err := s.batches.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	codes := make(map[uint]string, len(batches))
	for _, batch := range batches {
		codes[batch.ID] = batch.Code
	}
	for i := range items {
		items[i].BatchCode = codes[items[i].BatchID]
	}
	return nil
}

func (s *labSampleService) Create(ctx context.Context, input dto.CreateLabSample, actor, requestID string) (model.LabSample, error) {
	if err := validateLabSampleBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.LabSample{}, err
	}
	if strings.TrimSpace(input.HandoverBy) == "" {
		return model.LabSample{}, fmt.Errorf("%w: handover person is required", ErrInvalidInput)
	}
	batch, err := s.batches.Get(ctx, input.BatchID)
	if err != nil {
		return model.LabSample{}, fmt.Errorf("load 采样批次 %d: %w", input.BatchID, err)
	}
	item := model.LabSample{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.LabSampleInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
		BatchID:     batch.ID,
		HandoverBy:  strings.TrimSpace(input.HandoverBy),
		HandoverAt:  time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.LabSample{}, fmt.Errorf("create 实验室样本: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "LabSample", item.ID, "", item.Status,
		fmt.Sprintf("created 实验室样本, 挂接批次 %s, 交接人 %s", batch.Code, item.HandoverBy))
	item.BatchCode = batch.Code
	return item, nil
}

func (s *labSampleService) Update(ctx context.Context, id uint, input dto.UpdateLabSample, actor, requestID string) (model.LabSample, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.LabSample{}, err
	}
	if err := validateLabSampleBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.LabSample{}, err
	}
	if strings.TrimSpace(input.HandoverBy) == "" {
		return model.LabSample{}, fmt.Errorf("%w: handover person is required", ErrInvalidInput)
	}
	if _, err := s.batches.Get(ctx, input.BatchID); err != nil {
		return model.LabSample{}, fmt.Errorf("load 采样批次 %d: %w", input.BatchID, err)
	}
	current.Name = strings.TrimSpace(input.Name)
	current.Description = strings.TrimSpace(input.Description)
	current.Facility = strings.TrimSpace(input.Facility)
	current.Owner = strings.TrimSpace(input.Owner)
	current.Category = strings.TrimSpace(input.Category)
	current.RiskLevel = input.RiskLevel
	current.MetricValue = input.MetricValue
	current.MetricUnit = strings.TrimSpace(input.MetricUnit)
	current.EffectiveAt = input.EffectiveAt.UTC()
	current.Evidence = strings.TrimSpace(input.Evidence)
	current.RelatedCode = strings.ToUpper(strings.TrimSpace(input.RelatedCode))
	current.BatchID = input.BatchID
	current.HandoverBy = strings.TrimSpace(input.HandoverBy)
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.LabSample{}, fmt.Errorf("update 实验室样本: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "LabSample", id, current.Status, current.Status, "updated business fields")
	return s.Get(ctx, id)
}

func (s *labSampleService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.LabSample, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.LabSample{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.LabSampleTransitions, current.Status, target) {
		return model.LabSample{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == string(constants.SampleStateTesting) {
		if err := s.ensureBatchReadyForTesting(ctx, current); err != nil {
			return model.LabSample{}, err
		}
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.LabSample{}, fmt.Errorf("transition 实验室样本: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "LabSample", id, before, target, input.Reason); err != nil {
		return model.LabSample{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.Get(ctx, id)
}

// ensureBatchReadyForTesting blocks 开工检测 while the owning batch is still
// planned or collecting; only batches that reached received may run tests.
func (s *labSampleService) ensureBatchReadyForTesting(ctx context.Context, sample model.LabSample) error {
	if sample.BatchID == 0 {
		return fmt.Errorf("%w: 样本未挂接采样批次，不能开工检测", ErrInvalidTransition)
	}
	batch, err := s.batches.Get(ctx, sample.BatchID)
	if err != nil {
		return fmt.Errorf("load 采样批次 %d: %w", sample.BatchID, err)
	}
	if batch.Status == "planned" || batch.Status == "collecting" {
		return fmt.Errorf("%w: 采样批次 %s 仍为 %s，样本不能开工检测", ErrInvalidTransition, batch.Code, batch.Status)
	}
	return nil
}

func (s *labSampleService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "LabSample", id, current.Status, "deleted", "soft deleted 实验室样本")
}

func (s *labSampleService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateLabSampleBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
