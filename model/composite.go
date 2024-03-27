package model

import (
	//"time"
)

type measurementComposite struct {
	*Measurement
	Terminal *MeasurementTerminal `json:"terminal"`
	Patient  *Patient             `json:"patient"`
}

// 計測内の診断記録情報。
type DiagnosisEntity struct {
	*Diagnosis
	Algorithm *DiagnosisAlgorithm `json:"algorithm"`
	Contents  []*DiagnosisContent `json:"contents"`
	//Annotator *Annotator `json:"annotator"`
}

type MeasurementEntity measurementComposite

// センサ値。
type SensorValue struct {
	Value      int   `json:"value"`
	ObservedAt int64 `json:"observedAt"`
}

// ログイン中の医師。
type HospitalDoctor struct {
	*Doctor
	Hospital *Hospital `json:"hospital"`
}

// 最新の診断と診断用データを持つ計測。
type DiagnosisMeasurmentEntity struct {
	*Measurement
	LatestDiagnosis *Diagnosis   `json:"latestDiagnosis"`
	HeartRates      []*HeartRate `json:"heartRates"`
	TOCOs           []*TOCO      `json:"tocos"`
}

// 自動診断イベント。
type ComputedEventEntity struct {
	*ComputedEvent
	Measurement *Measurement       `json:"measurement"`
	Annotations []*AnnotatedEvent  `json:"annotations"`
}

// アノテーター登録イベント。
type AnnotatedEventEntity struct {
	*AnnotatedEvent
	Measurement *Measurement   `json:"measurement"`
	Event       *ComputedEvent `json:"event"`
}