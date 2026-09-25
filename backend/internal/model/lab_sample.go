package model

import "time"

// LabSample models 实验室样本 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
type LabSample struct {
	BaseModel
	Facility    string    `json:"facility" gorm:"size:120;index"`
	Owner       string    `json:"owner" gorm:"size:120;index"`
	Category    string    `json:"category" gorm:"size:80;index"`
	RiskLevel   string    `json:"riskLevel" gorm:"size:32;index"`
	MetricValue float64   `json:"metricValue"`
	MetricUnit  string    `json:"metricUnit" gorm:"size:24"`
	EffectiveAt time.Time `json:"effectiveAt"`
	Evidence    string    `json:"evidence" gorm:"size:2000"`
	RelatedCode string    `json:"relatedCode" gorm:"size:64;index"`
	// BatchID links the sample to the 采样批次 it was received under; HandoverBy
	// records who handed the sample over at reception. BatchCode is a read-only
	// denormalized view filled by the service layer for list/detail responses.
	BatchID    uint      `json:"batchId" gorm:"index"`
	BatchCode  string    `json:"batchCode,omitempty" gorm:"-"`
	HandoverBy string    `json:"handoverBy" gorm:"size:120"`
	HandoverAt time.Time `json:"handoverAt"`
}

func (item *LabSample) GetBase() *BaseModel { return &item.BaseModel }

func (item LabSample) TableName() string { return "lab_samples" }

var LabSampleInitialStatus = "received"
