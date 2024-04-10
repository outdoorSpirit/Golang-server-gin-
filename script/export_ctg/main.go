package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	//"time"

	"github.com/spiker/spiker-server/config"
	C "github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
	//"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
)

type ExportValue struct {
	PatientCode string
	FHR1        int
	UC          int
	Timestamp   int64
}

func fetchData(hospitalUuid string, measurementCode string) ([]*ExportValue, error) {
	db := lib.GetDB(lib.ReadDBKey)

	// 病院取得。
	hospital, err := (&S.HospitalService{
		Service: nil,
		DB: db,
	}).InquireByUuid(hospitalUuid)

	if err != nil {
		return nil, err
	}

	// 計測取得。
	measurement, err := (&S.MeasurementService{
		Service: nil,
		DB: db,
	}).InquireByCode(hospital.Id, measurementCode)

	if err != nil {
		return nil, err
	}

	// 心拍。
	service := &S.DataService{
		Service: nil,
		DB: db,
		Influx: lib.GetInfluxDB(),
	}

	values := []*ExportValue{}

	if hrs, e := service.ListCTGData(measurement.Id, C.MeasurementTypeHeartRate, nil, nil); e != nil {
		return nil, e
	} else if ucs, e := service.ListCTGData(measurement.Id, C.MeasurementTypeTOCO, nil, nil); e != nil {
		return nil, e
	} else if len(hrs) != len(ucs) {
		return nil, fmt.Errorf("Sizes of FHR1(%d) and UC(%d) are not the same.")
	} else {
		for i, hr := range hrs {
			values = append(values, &ExportValue{
				PatientCode: measurementCode,
				FHR1: hr.Value,
				UC: ucs[i].Value,
				Timestamp: hr.ObservedAt,
			})
		}

		return values, nil
	}
}

type Args struct {
	hospitalUuid    string
	measurementCode string
}

func parseArgs() (*Args, error) {
	flag.Parse()

	args := flag.Args()

	if len(args) != 2 {
		return nil, fmt.Errorf("usage: go run script/export_ctg/main.go [hospital_uuid] [measurement_code]")
	}

	return &Args{args[0], args[1]}, nil
}

func main() {
	os.Setenv("SERVER_ENV", "prod")

	// 引数
	var args *Args

	if a, e := parseArgs(); e != nil {
		log.Fatal(e)
	} else {
		args = a
	}

	// 設定
	config.SetupAll()

	// データ取得
	if values, e := fetchData(args.hospitalUuid, args.measurementCode); e != nil {
		log.Fatal(e)
	} else {
		rows := [][]string{}

		for _, v := range values {
			rows = append(rows, []string{
				v.PatientCode,
				fmt.Sprintf("%d", v.FHR1),
				fmt.Sprintf("%d", v.UC),
				fmt.Sprintf("%d", v.Timestamp),
			})
		}

		w := csv.NewWriter(os.Stdout)

		if e := w.WriteAll(rows); e != nil {
			log.Fatal(e)
		}

		w.Flush()
	}
}