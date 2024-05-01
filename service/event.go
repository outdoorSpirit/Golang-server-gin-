package service

import (
	"fmt"
	"time"

	"gopkg.in/gorp.v2"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type EventService struct {
	*Service
	DB *gorp.DbMap
}

type EventTxService struct {
	*Service
	DB *gorp.Transaction
}

func (s *EventService) CheckComputedEventInMeasurement(
	measurementId int,
	eventId int,
) error {
	if r, e := rds.InquireComputedEvent(s.DB, eventId); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.MeasurementId != measurementId {
		return C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Computed event #%d is not found in measurement #%d", eventId, measurementId),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

func (s *EventService) CheckAnnotatedEventInMeasurement(
	measurementId int,
	eventId int,
) error {
	if r, e := rds.InquireAnnotatedEvent(s.DB, eventId); e != nil {
		return C.DB_OPERATION_ERROR(e)
	} else if r == nil || r.MeasurementId != measurementId {
		return C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Computed event #%d is not found in measurement #%d", eventId, measurementId),
			map[string]interface{}{},
		)
	} else {
		return nil
	}
}

func (s *EventService) FetchComputedEvent(
	eventId int,
	hiddenAlso bool,
) (*model.ComputedEventEntity, error) {
	if r, e := rds.FetchComputedEvent(s.DB, eventId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil || (!hiddenAlso && r.IsHidden) {
		return nil, C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Computed event #%d is not found", eventId),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

func (s *EventService) FetchAnnotatedEvent(
	eventId int,
) (*model.AnnotatedEventEntity, error) {
	if r, e := rds.FetchAnnotatedEvent(s.DB, eventId); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else if r == nil {
		return nil, C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Annotated event #%d is not found", eventId),
			map[string]interface{}{},
		)
	} else {
		return r, nil
	}
}

func (s *EventService) ListComputedEventsInRange(
	measurementId int,
	begin time.Time,
	end time.Time,
	hiddenAlso bool,
) ([]*model.ComputedEventEntity, error) {
	if r, e := rds.ListComputedEventsInRange(s.DB, measurementId, begin, end, hiddenAlso); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return r, nil
	}
}

func (s *EventService) ListAnnotatedEventsInRange(
	measurementId int,
	begin time.Time,
	end time.Time,
) ([]*model.AnnotatedEventEntity, error) {
	if r, e := rds.ListAnnotatedEventsInRange(s.DB, measurementId, begin, end); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return r, nil
	}
}

// ある病院において、指定したイベント以降の既定値以上のリスクを持つ自動診断イベントを全て取得する。
func (s *EventService) ListUnreadComputedEvents(
	hospitalId int,
	latest *int,
) ([]*model.ComputedEventEntity, error) {
	var beginTime time.Time

	now := time.Now()

	if latest != nil {
		if r, e := rds.InquireComputedEvent(s.DB, *latest); e != nil {
			return nil, C.DB_OPERATION_ERROR(e)
		} else if r != nil {
			beginTime = r.RangeUntil
		} else {
			beginTime = now.Add(-C.AlertBackingDuration)
		}
	} else {
		beginTime = now.Add(-C.AlertBackingDuration)
	}

	measurementFrom := now.Add(-C.AlertMeasurementInterval)

	if r, e := rds.ListUnreadComputedEvents(s.DB, hospitalId, C.AlertRiskThreshold, beginTime, measurementFrom); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		return r, nil
	}
}

// 病院に対するアラート用のイベントを古い順に取得する。
func (s *EventService) ListAlertEvents(
	measurements []int,
) ([]*model.AnnotatedEventEntity, error) {
	now := time.Now()

	if entities, e := rds.FetchLatestAnnotationsForMeasurements(s.DB, measurements, now); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	} else {
		results := []*model.AnnotatedEventEntity{}

		for _, a := range entities {
			if !a.AnnotatedEvent.IsClosed {
				results = append(results, a)
			}
		}

		return results, nil
	}
}

func (s *EventTxService) RegisterAnnotated(
	measurementId int,
	annotatorId int,
	eventId *int,
	risk int,
	memo string,
	rangeFrom time.Time,
	rangeUntil time.Time,
) (*model.AnnotatedEvent, error) {
	if eventId != nil {
		if r, e := rds.InquireComputedEvent(s.DB, *eventId); e != nil {
			return nil, C.DB_OPERATION_ERROR(e)
		} else if r == nil {
			return nil, C.NewBadRequestError(
				"event_not_found",
				fmt.Sprintf("Event #%d is not found", *eventId),
				map[string]interface{}{},
			)
		} else if r.MeasurementId != measurementId {
			return nil, C.NewBadRequestError(
				"event_not_found",
				fmt.Sprintf("Event #%d is not found in measurement #%d", *eventId, measurementId),
				map[string]interface{}{},
			)
		}
	}

	now := time.Now()

	record := &model.AnnotatedEvent{
		MeasurementId: measurementId,
		AnnotatorId: &annotatorId,
		ComputedEventId: eventId,
		Risk: &risk,
		Memo: memo,
		RangeFrom: rangeFrom,
		RangeUntil: rangeUntil,
		CreatedAt: now,
		ModifiedAt: now,
	}

	if e := s.DB.Insert(record); e != nil {
		return nil, C.DB_OPERATION_ERROR(e)
	}

	// リスクが規定値以上ならばサイレントを解除。
	if risk >= C.AlertRiskThreshold {
		if e := rds.StopCurrentSilent(s.DB, measurementId, now); e != nil {
			return nil, C.DB_OPERATION_ERROR(e)
		}
	}

	return record, nil
}

func (s *EventTxService) UpdateAnnotated(
	annotationID int,
	annotatorId int,
	risk int,
	memo string,
) error {
	record, err := rds.InquireAnnotatedEvent(s.DB, annotationID)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if record == nil {
		return C.NewNotFoundError(
			"annotation_not_found",
			fmt.Sprintf("Annotation #%d is not found", annotationID),
			map[string]interface{}{},
		)
	}

	now := time.Now()

	record.AnnotatorId = &annotatorId
	record.Risk = &risk
	record.Memo = memo
	record.ModifiedAt = now

	if _, e := s.DB.Update(record); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	// リスクが規定値以上ならばサイレントを解除。
	if risk >= C.AlertRiskThreshold {
		if e := rds.StopCurrentSilent(s.DB, record.MeasurementId, now); e != nil {
			return C.DB_OPERATION_ERROR(e)
		}
	}

	return nil
}

func (s *EventTxService) DeleteAnnotated(
	annotationId int,
) error {
	record, err := rds.InquireAnnotatedEvent(s.DB, annotationId)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if record == nil {
		return C.NewNotFoundError(
			"annotation_not_found",
			fmt.Sprintf("Annotaion #%d is not found", annotationId),
			map[string]interface{}{},
		)
	}

	if _, e := s.DB.Delete(record); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}

func (s *EventTxService) ToggleComputedEventVisibility(
	eventId int,
	isVisible bool,
) error {
	event, err := rds.InquireComputedEvent(s.DB, eventId)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if event == nil {
		return C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Computed event #%d is not found", eventId),
			map[string]interface{}{},
		)
	}

	now := time.Now()

	event.IsHidden = !isVisible
	event.ModifiedAt = now

	if _, e := s.DB.Update(event); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}

func (s *EventTxService) SuspendComputedEvent(
	eventId int,
) error {
	event, err := rds.InquireComputedEvent(s.DB, eventId)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if event == nil {
		return C.NewNotFoundError(
			"event_not_found",
			fmt.Sprintf("Computed event #%d is not found", eventId),
			map[string]interface{}{},
		)
	}

	now := time.Now()

	event.IsSuspended = true
	event.ModifiedAt = now

	if _, e := s.DB.Update(event); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}

func (s *EventTxService) CloseAnnotated(
	annotationId int,
	memo string,
) error {
	record, err := rds.InquireAnnotatedEvent(s.DB, annotationId)

	if err != nil {
		return C.DB_OPERATION_ERROR(err)
	} else if record == nil {
		return C.NewNotFoundError(
			"annotation_not_found",
			fmt.Sprintf("Annotaion #%d is not found", annotationId),
			map[string]interface{}{},
		)
	}

	now := time.Now()

	record.IsClosed = true
	record.ClosingMemo = &memo
	record.ClosedAt = &now
	record.ModifiedAt = now

	if _, e := s.DB.Update(record); e != nil {
		return C.DB_OPERATION_ERROR(e)
	}

	return nil
}