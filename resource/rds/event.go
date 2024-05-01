package rds

import (
	"fmt"
	"database/sql"
	//"encoding/json"
	"strings"
	"time"

	//C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/model"
)

func InquireComputedEvent(
	db model.QueryExecutor,
	id int,
) (*model.ComputedEvent, error) {
	if r, e := db.Get(model.ComputedEvent{}, id); e != nil {
		return nil, e
	} else if r == nil {
		return nil, nil
	} else {
		return r.(*model.ComputedEvent), nil
	}
}

func InquireAnnotatedEvent(
	db model.QueryExecutor,
	id int,
) (*model.AnnotatedEvent, error) {
	if r, e := db.Get(model.AnnotatedEvent{}, id); e != nil {
		return nil, e
	} else if r == nil {
		return nil, nil
	} else {
		return r.(*model.AnnotatedEvent), nil
	}
}

func FetchComputedEvent(
	db model.QueryExecutor,
	id int,
) (*model.ComputedEventEntity, error) {
	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			computed_event AS e
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			LEFT JOIN annotated_event AS ae ON e.id = ae.computed_event_id
		WHERE
			e.id = $1
		ORDER BY ae.created_at DESC`,
		prefixColumns(model.ComputedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.AnnotatedEvent{}, "ae", "ae"),
	)

	if rows, e := db.Query(query, id); e != nil {
		return nil, e
	} else {
		var entity *model.ComputedEventEntity = nil

		safeRowsIterator(rows, func(rows *sql.Rows) {
			event := &model.ComputedEvent{}
			measurement := &model.Measurement{}
			annotation := &model.AnnotatedEvent{}

			scanRows(db, rows, event, "e")
			scanRows(db, rows, measurement, "m")
			scanRows(db, rows, annotation, "ae")

			if entity == nil {
				entity = &model.ComputedEventEntity{event, measurement, []*model.AnnotatedEvent{}}
			}

			if annotation.Id != 0 {
				entity.Annotations = append(entity.Annotations, annotation)
			}
		})

		return entity, nil
	}
}

func FetchAnnotatedEvent(
	db model.QueryExecutor,
	id int,
) (*model.AnnotatedEventEntity, error) {
	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			annotated_event AS e
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			LEFT JOIN computed_event AS ce ON e.computed_event_id = ce.id
		WHERE
			e.id = $1`,
		prefixColumns(model.AnnotatedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.ComputedEvent{}, "ce", "ce"),
	)

	if rs, e := constructAnnotatedEntities(db, query, newInterfaces(id)); e != nil {
		return nil, e
	} else if len(rs) == 0 {
		return nil, nil
	} else {
		return rs[0], nil
	}
}

func ListComputedEventsInRange(
	db model.QueryExecutor,
	measurementId int,
	begin time.Time,
	end time.Time,
	hiddenAlso bool,
) ([]*model.ComputedEventEntity, error) {
	ip := incrementalPlaceholder{0}

	q := andQuery().add(fmt.Sprintf("e.measurement_id = $%d", ip.GetIndex()), measurementId)
	q.add(fmt.Sprintf("e.range_from <= $%d", ip.GetIndex()), end)
	q.add(fmt.Sprintf("e.range_until >= $%d", ip.GetIndex()), begin)

	if !hiddenAlso {
		q.add(fmt.Sprintf("not is_hidden"))
	}

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			computed_event AS e
			INNER JOIN measurement AS m ON e.measurement_id = m.id
		%s
		ORDER BY e.range_until DESC`,
		prefixColumns(model.ComputedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		where,
	)

	return constructComputedEntities(db, query, params)
}

func ListAnnotatedEventsInRange(
	db model.QueryExecutor,
	measurementId int,
	begin time.Time,
	end time.Time,
) ([]*model.AnnotatedEventEntity, error) {
	ip := incrementalPlaceholder{0}

	q := andQuery().add(fmt.Sprintf("e.measurement_id = $%d", ip.GetIndex()), measurementId)
	q.add(fmt.Sprintf("e.range_from <= $%d", ip.GetIndex()), end)
	q.add(fmt.Sprintf("e.range_until >= $%d", ip.GetIndex()), begin)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			annotated_event AS e
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			LEFT JOIN computed_event AS ce ON e.computed_event_id = ce.id
		%s
		ORDER BY e.range_until DESC`,
		prefixColumns(model.AnnotatedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.ComputedEvent{}, "ce", "ce"),
		where,
	)

	return constructAnnotatedEntities(db, query, params)
}

// ある病院において、指定した日時以降の既定値以上のリスクを持つ自動診断を古い順に取得する。
func ListUnreadComputedEvents(
	db model.QueryExecutor,
	hospitalId int,
	risk int,
	beginTime time.Time,
	measurementFrom time.Time,
) ([]*model.ComputedEventEntity, error) {
	ip := &incrementalPlaceholder{0}

	q := andQuery().add(`NOT EXISTS (SELECT * FROM annotated_event WHERE computed_event_id = e.id)`)
	q.add(`NOT e.is_suspended`)
	q.add(fmt.Sprintf(`m.last_time >= $%d`, ip.GetIndex()), measurementFrom)
	q.add(fmt.Sprintf(`p.hospital_id = $%d`, ip.GetIndex()), hospitalId)
	q.add(fmt.Sprintf(`e.risk >= $%d`, ip.GetIndex()), risk)
	q.add(fmt.Sprintf(`e.range_from >= $%d`, ip.GetIndex()), beginTime)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s
		FROM
			computed_event AS e
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			INNER JOIN patient AS p ON m.patient_id = p.id
		%s
		ORDER BY e.range_from ASC`,
		prefixColumns(model.ComputedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		where,
	)

	return constructComputedEntities(db, query, params)
}

// 指定した計測に関して、それぞれ最新のアノテーションを取得する。
func FetchLatestAnnotationsForMeasurements(
	db model.QueryExecutor,
	measurements []int,
	now time.Time,
) ([]*model.AnnotatedEventEntity, error) {
	if len(measurements) == 0 {
		return []*model.AnnotatedEventEntity{}, nil
	}

	ip := &incrementalPlaceholder{}

	midHolder := []string{}
	mids := []interface{}{}
	for _, m := range measurements {
		mids = append(mids, m)
		midHolder = append(midHolder, fmt.Sprintf("$%d", ip.GetIndex()))
	}

	q := andQuery()
	q.add(`NOT m.is_closed`)
	q.add(fmt.Sprintf(`e.measurement_id IN (%s)`, strings.Join(midHolder, ",")), mids...)
	q.add(fmt.Sprintf(`NOT EXISTS (
		SELECT * FROM measurement_alert WHERE measurement_id = m.id AND silent_from <= $%d AND silent_until >= $%d
	)`, ip.GetIndex(), ip.GetIndex()), now, now)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			annotated_event AS e
			INNER JOIN (
				SELECT
					id,
					ROW_NUMBER() OVER (PARTITION BY measurement_id ORDER BY range_until DESC, id DESC) AS row
				FROM
					annotated_event
			) AS e_ ON e.id = e_.id AND e_.row = 1
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			LEFT JOIN computed_event AS ce ON e.computed_event_id = ce.id
		%s
		ORDER BY e.range_from ASC`,
		prefixColumns(model.AnnotatedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.ComputedEvent{}, "ce", "ce"),
		where,
	)

	return constructAnnotatedEntities(db, query, params)
}

// ある病院において、指定した日時以降の既定値以上のリスクを持つアノテーションを古い順に取得する。
// 自動診断イベントを参照しないアノテーションは、条件を満たせば取得対象となる。
// 参照する場合、同じイベントを参照するアノテーションの内、最も新しいものだけが取得対象となる。
func ListAlertAnnotatedEvents(
	db model.QueryExecutor,
	hospitalId int,
	risk int,
	beginTime time.Time,
	now time.Time,
) ([]*model.AnnotatedEventEntity, error) {
	ip := &incrementalPlaceholder{0}

	q := andQuery()
	q.add(fmt.Sprintf(`p.hospital_id = $%d`, ip.GetIndex()), hospitalId)
	q.add(fmt.Sprintf(`e.range_until >= $%d`, ip.GetIndex()), beginTime)
	q.add(fmt.Sprintf(`e.risk >= $%d`, ip.GetIndex()), risk)
	q.add(fmt.Sprintf(`NOT EXISTS (
		SELECT * FROM measurement_alert WHERE measurement_id = m.id AND silent_from <= $%d AND silent_until >= $%d
	)`, ip.GetIndex(), ip.GetIndex()), now, now)

	where, params := q.where()

	query := fmt.Sprintf(
		`SELECT
			%s, %s, %s
		FROM
			annotated_event AS e
			INNER JOIN (
				SELECT
					id, computed_event_id,
					ROW_NUMBER() OVER (PARTITION BY computed_event_id ORDER BY created_at DESC, id DESC) AS row
				FROM
					annotated_event
			) AS e_ ON e.id = e_.id AND (e_.computed_event_id IS NULL OR e_.row = 1)
			INNER JOIN measurement AS m ON e.measurement_id = m.id
			INNER JOIN patient AS p ON m.patient_id = p.id
			LEFT JOIN computed_event AS ce ON e.computed_event_id = ce.id
		%s
		ORDER BY e.range_from ASC`,
		prefixColumns(model.AnnotatedEvent{}, "e", "e"),
		prefixColumns(model.Measurement{}, "m", "m"),
		prefixColumns(model.ComputedEvent{}, "ce", "ce"),
		where,
	)

	return constructAnnotatedEntities(db, query, params)
}

// ある計測の自動判読イベントに新規診断結果を接続する。
func MergeNewDiagnosis(
	db model.QueryExecutor,
	measurementId int,
	contents []*model.DiagnosisContent,
	now time.Time,
) error {
	if len(contents) == 0 {
		return nil
	}

	// 最も古い診断結果以降のイベントを取得。
	currentEvents := []*model.ComputedEvent{}

	query := `SELECT * FROM computed_event WHERE measurement_id = $1 AND range_until >= $2 ORDER BY range_from ASC`

	if _, e := db.Select(&currentEvents, query, measurementId, contents[0].RangeFrom); e != nil {
		return e
	}

	// 最新の既存イベントの期間の先頭より前の新規診断結果は捨てる。
	var latestEvent *model.ComputedEvent = nil

	if len(currentEvents) > 0 {
		latestEvent = currentEvents[len(currentEvents)-1]

		index := 0

		for _, c := range contents {
			if c.RangeUntil.Before(latestEvent.RangeFrom) {
				index++
			} else {
				break
			}
		}

		contents = contents[index:]
	}

	// 時間的なオーバーラップがある診断結果はマージして、新しい結果で置き換える。
	latestModified := false

	if latestEvent != nil {
		index := 0

		for _, c := range contents {
			if c.RangeFrom.Before(latestEvent.RangeUntil) {
				index++

				latestModified = true

				latestEvent.Risk = c.Risk
				latestEvent.RangeUntil = c.RangeUntil
				latestEvent.Parameters = c.Parameters
			} else {
				break
			}
		}

		contents = contents[index:]
	}

	// 変更があれば既存の最新イベントを更新。
	if latestModified {
		if _, e := db.Update(latestEvent); e != nil {
			return e
		}
	}

	// 残された診断結果をイベントとして登録。
	if len(contents) > 0 {
		holder := []interface{}{}

		for _, c := range contents {
			holder = append(holder, &model.ComputedEvent{
				MeasurementId: measurementId,
				Risk: c.Risk,
				Memo: "",
				Parameters: c.Parameters,
				IsHidden: false,
				RangeFrom: c.RangeFrom,
				RangeUntil: c.RangeUntil,
				CreatedAt: now,
				ModifiedAt: now,
			})
		}

		if e := db.Insert(holder...); e != nil {
			return e
		}
	}

	return nil
}

func constructComputedEntities(
	db model.QueryExecutor,
	query string,
	params *interfaces,
) ([]*model.ComputedEventEntity, error) {
	records := []*model.ComputedEventEntity{}

	entityMap := map[int]*model.ComputedEventEntity{}
	ids := []int{}

	if rows, e := db.Query(query, params.values...); e != nil {
		return nil, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			event := &model.ComputedEvent{}
			measurement := &model.Measurement{}

			scanRows(db, rows, event, "e")
			scanRows(db, rows, measurement, "m")

			entity := &model.ComputedEventEntity{event, measurement, []*model.AnnotatedEvent{}}

			records = append(records, entity)

			entityMap[event.Id] = entity
			ids = append(ids, event.Id)
		})
	}

	// アノテーションを追加。
	if len(ids) > 0 {
		annotations := []*model.AnnotatedEvent{}

		if _, e := db.Select(
			&annotations,
			`SELECT * FROM annotated_event WHERE computed_event_id IN (:ids) ORDER BY created_at DESC`,
			map[string]interface{}{"ids": ids},
		); e != nil {
			return nil, e
		}

		for _, ann := range annotations {
			entity := entityMap[*ann.ComputedEventId]
			entity.Annotations = append(entity.Annotations, ann)
		}
	}

	return records, nil
}

func constructAnnotatedEntities(
	db model.QueryExecutor,
	query string,
	params *interfaces,
) ([]*model.AnnotatedEventEntity, error) {
	records := []*model.AnnotatedEventEntity{}

	if rows, e := db.Query(query, params.values...); e != nil {
		return nil, e
	} else {
		safeRowsIterator(rows, func(rows *sql.Rows) {
			event := &model.AnnotatedEvent{}
			measurement := &model.Measurement{}
			computed := &model.ComputedEvent{}

			scanRows(db, rows, event, "e")
			scanRows(db, rows, measurement, "m")
			scanRows(db, rows, computed, "ce")

			entity := &model.AnnotatedEventEntity{event, measurement, nil}

			if computed.Id > 0 {
				entity.Event = computed
			}

			records = append(records, entity)
		})
	}

	return records, nil
}