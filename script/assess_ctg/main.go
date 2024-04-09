package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spiker/spiker-server/config"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
)

const (
	dataCountPerSecond int = 1
	// 診断処理を実行する最小のデータ割合。
	minimumCTGRatio float64 = 0.9
)

func main() {
	// REVIEW 環境固定。
	os.Setenv("SERVER_ENV", "prod")

	// ログファイル準備
	var rootDir string

	if self, e := os.Executable(); e != nil {
		log.Fatal(e)
	} else {
		rootDir = filepath.Dir(self)
	}

	if e := os.Chdir(rootDir); e != nil {
		log.Fatal(e)
	}

	logDir := filepath.Join(rootDir, "logs")

	if e := os.MkdirAll(logDir, 0777); e != nil {
		log.Fatal(e)
	}

	today := time.Now()

	logPath := filepath.Join(logDir, fmt.Sprintf("risk-assessmentor-%04d%02d%02d.log", today.Year(), today.Month(), today.Day()))

	if f, e := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); e != nil {
		log.Fatal(e)
	} else {
		defer f.Close()
		log.SetOutput(f)
	}

	// 設定
	config.SetupAll()

	assessmentConfig := config.AssessmentConfig()

	// コマンドライン引数
	var currentTime string
	flag.StringVar(&currentTime, "now", "", "Current UTC time of master data in YYYYMMDDhhmmss format")

	flag.Parse()

	var diagnosisTime time.Time

	if currentTime == "" {
		diagnosisTime = time.Now().Add(time.Duration(-assessmentConfig.Delay)*time.Second)
	} else if t, e := time.Parse("20060102150405", currentTime); e != nil {
		log.Fatal("Failed to parse current time: %v\n", e)
	} else {
		diagnosisTime = t.UTC()
	}

	log.Printf("Diagnosis time: %v\n", diagnosisTime)

	var measurements []*model.DiagnosisMeasurmentEntity

	dataSeconds := assessmentConfig.Duration + 2*assessmentConfig.Cutoff

	dataDuration := time.Duration(dataSeconds) * time.Second

	if ms, e := (&S.MeasurementService{
		Service: nil,
		DB: lib.GetDB(lib.ReadDBKey),
		Influx: lib.GetInfluxDB(),
	}).CollectForAssessment(
		diagnosisTime,
		dataDuration,
		time.Duration(assessmentConfig.Interval)*time.Second,
	); e != nil {
		log.Fatal(e)
	} else {
		measurements = ms
	}

	if len(measurements) == 0 {
		log.Printf("No measurements are found.\n")
		os.Exit(0)
	}

	log.Printf("Execute for %d measurements\n", len(measurements))

	errorChannel := make(chan error)
	diagnosisChannel := make(chan *model.DiagnosisEntity)

	service := &S.AssessmentService{
		Service: nil,
		DB: lib.GetDB(lib.ReadDBKey),
	}

	routineCount := 0

	dataCountThreshold := int(float64(dataCountPerSecond * dataSeconds) * minimumCTGRatio)

	for _, m := range measurements {
		mm := m

		if len(mm.HeartRates) <= dataCountThreshold || len(mm.TOCOs) <= dataCountThreshold {
			log.Printf("Too few data: HR = %d, UC = %d", len(mm.HeartRates), len(mm.TOCOs))
			continue
		}

		routineCount++

		go func() {
			log.Printf("Start measurement #%d\n", mm.Id)

			diagnosis, err := service.Execute(
				mm,
				diagnosisTime,
				time.Duration(assessmentConfig.Duration)*time.Second,
				assessmentConfig.Root,
				assessmentConfig.Command,
				assessmentConfig.Parameters,
				assessmentConfig.Algorithm,
				assessmentConfig.Version,
			)

			if err != nil {
				errorChannel <- err
				return
			}

			// 両端を切り捨てる。
			if len(diagnosis.Contents) > 0 && assessmentConfig.Cutoff > 0 {
				from := diagnosis.RangeFrom.Add(time.Duration(assessmentConfig.Cutoff)*time.Second)
				until := diagnosis.RangeUntil.Add(time.Duration(-assessmentConfig.Cutoff)*time.Second)

				validFrom := 0
				validUntil := len(diagnosis.Contents)-1

				for _, c := range diagnosis.Contents {
					if c.RangeFrom.Before(from) {
						validFrom++
					} else {
						break
					}
				}

				for i := len(diagnosis.Contents)-1; i >= 0; i-- {
					if diagnosis.Contents[i].RangeUntil.After(until) {
						validUntil--
					} else {
						break
					}
				}

				if validFrom > validUntil {
					diagnosis.Contents = []*model.DiagnosisContent{}
				} else {
					diagnosis.Contents = diagnosis.Contents[validFrom:validUntil]
				}
			}

			if err != nil {
				errorChannel <- err
			} else {
				diagnosisChannel <- diagnosis
			}
		}()
	}

	if routineCount == 0 {
		log.Printf("No measurements are used for diagnosis.")
		os.Exit(0)
	}

	counter := 0

	diagnoses := []*model.DiagnosisEntity{}

	for {
		select {
		case d := (<-diagnosisChannel):
			log.Printf("Success measurement #%d\n", d.MeasurementId)
			diagnoses = append(diagnoses, d)
		case e := (<-errorChannel):
			log.Printf("[NG] %v\n", e)
		}

		counter++
		if counter == routineCount {
			break
		}
	}

	tx, err := lib.GetDB(lib.WriteDBKey).Begin()

	if err != nil {
		log.Fatal(err)
	}

	status := true

	defer func() {
		if e := recover(); e != nil {
			tx.Rollback()
			log.Println(e)
		} else if !status {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if e := (&S.DiagnosisTxService{
		Service: nil,
		DB: tx,
	}).Register(diagnoses); e != nil {
		status = false
		log.Printf("Failed to register diagnoses: %v", e)
	} else {
		log.Printf("Succeeded to register %d diagnoses", len(diagnoses))
	}
}