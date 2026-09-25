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
	// BatchID links the received sample to its 采样批次 and HandoverBy records
	// who handed the sample over at reception. Both are fixed at reception time.
	BatchID    uint   `json:"batchId" gorm:"index;not null"`
	HandoverBy string `json:"handoverBy" gorm:"size:120;not null"`
}

func (item *LabSample) GetBase() *BaseModel { return &item.BaseModel }

func (item LabSample) TableName() string { return "lab_samples" }

var LabSampleInitialStatus = "received"

// LabSampleDisposedStatus is the terminal disposal state. Samples in any other
// state are considered unfinished and keep their batch open.
const LabSampleDisposedStatus = "disposed"
