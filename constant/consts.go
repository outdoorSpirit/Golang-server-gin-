package constant

import (
	"time"
)

// Language 言語。
type Language string

const (
	LanguageJa Language = "ja" // 日本語。
	LanguageEn Language = "en" // 英語。
)

// 計測種別。
type MeasurementType string

const (
	MeasurementTypeHeartRate MeasurementType = "heartrate"
	MeasurementTypeTOCO      MeasurementType = "toco"
)

// アラート関連。
const (
	AlertBackingDuration     time.Duration = time.Duration(60) * time.Minute
	AlertRiskThreshold       int           = 3
	AlertExecutionDuration   time.Duration = time.Duration(30) * time.Second
	AlertSilentDuration      time.Duration = time.Duration(5) * time.Minute
	AlertMeasurementInterval time.Duration = time.Duration(5) * time.Minute
)