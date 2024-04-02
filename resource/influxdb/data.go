package influxdb

import (
	"fmt"
	"time"

	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
)

// 心拍データを古い順に取得する。
func ListHeartRate(
	influx lib.InfluxDBClient,
	measurementId int,
	begin time.Time,
	end time.Time,
) ([]*model.HeartRate, error) {
	query := fmt.Sprintf(
		`from(bucket:"spiker")
			|> range(start:%d, stop:%d)
			|> filter(fn: (r) => r._measurement == "%s" and r.measurement_id == "%d")
			|> group()
			|> sort(columns:["_time"])`,
		begin.Unix(), end.Unix(),
		C.MeasurementTypeHeartRate, measurementId,
	)

	results := []*model.HeartRate{}

	if e := influx.Select(query, lib.PointConsumer(func(p lib.Point, field string) error {
		if m, ok := p.(*model.HeartRate); !ok {
			return fmt.Errorf("Invalid measurement type for heart rate: %s", p.Measurement())
		} else {
			results = append(results, m)
			return nil
		}
	})); e != nil {
		return nil, e
	}

	return results, nil
}

// TOCOデータを古い順に取得する。
func ListTOCO(
	influx lib.InfluxDBClient,
	measurementId int,
	begin time.Time,
	end time.Time,
) ([]*model.TOCO, error) {
	query := fmt.Sprintf(
		`from(bucket:"spiker")
			|> range(start:%d, stop:%d)
			|> filter(fn: (r) => r._measurement == "%s" and r.measurement_id == "%d")
			|> group()
			|> sort(columns:["_time"])`,
		begin.Unix(), end.Unix(),
		C.MeasurementTypeTOCO, measurementId,
	)

	results := []*model.TOCO{}

	if e := influx.Select(query, lib.PointConsumer(func(p lib.Point, field string) error {
		if m, ok := p.(*model.TOCO); !ok {
			return fmt.Errorf("Invalid measurement type for heart rate: %s", p.Measurement())
		} else {
			results = append(results, m)
			return nil
		}
	})); e != nil {
		return nil, e
	}

	return results, nil
}