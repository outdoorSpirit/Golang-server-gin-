package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spiker/spiker-server/config"
	"github.com/spiker/spiker-server/lib"
	S "github.com/spiker/spiker-server/service"
)

const (
	patientIdPrefix string = "TRC"
	defaultMachineId string = "unspecified"
)

func moveCurrentDir() (string, error) {
	var rootDir string

	if self, e := os.Executable(); e != nil {
		return "", e
	} else {
		rootDir = filepath.Dir(self)
	}

	if e := os.Chdir(rootDir); e != nil {
		return "", e
	}

	return rootDir, nil
}

func setupLogging(rootDir, logDirName, dailyFileFormat string) (func(), error) {
	logDir := filepath.Join(rootDir, logDirName)

	if e := os.MkdirAll(logDir, 0777); e != nil {
		log.Fatal(e)
	}

	today := time.Now()

	dayPart := fmt.Sprintf("%04d%02d%02d", today.Year(), today.Month(), today.Day())

	logPath := filepath.Join(logDir, fmt.Sprintf(dailyFileFormat, dayPart))

	if f, e := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); e != nil {
		return nil, e
	} else {
		log.SetOutput(f)

		return func() {
			f.Close()
		}, nil
	}

}

type Args struct {
	hospitalUuid   string
	machineId      string
	trcFile        string
	abortIfExists  bool
}

func parseArgs() (*Args, error) {
	machineId := flag.String("m", defaultMachineId, "ID of machine used in the measurement")
	abortIfExists := flag.Bool("x", false, "Flag to abort if patient code is already registered in the hospital")

	flag.Parse()

	args := flag.Args()

	if len(args) != 2 {
		return nil, fmt.Errorf("usage: go run script/import_trc/main.go [hospital_uuid] [trc_file]")
	}

	absPath, err := filepath.Abs(args[1])

	if err != nil {
		return nil, err
	}

	return &Args{args[0], *machineId, absPath, *abortIfExists}, nil
}

func parseTRC(path string) (*lib.TRCData, error) {
	if data, e := ioutil.ReadFile(path); e != nil {
		return nil, e
	} else {
		return lib.ParseTRC(data)
	}
}

func registerTRC(hospitalUuid string, machineId string, trc *lib.TRCData, abortIfExists bool) error {
	db := lib.GetDB(lib.WriteDBKey)

	// 病院取得。
	hs := &S.HospitalService{
		Service: nil,
		DB: db,
	}

	hospital, err := hs.InquireByUuid(hospitalUuid)

	if err != nil {
		return err
	}

	status := false

	tx, err := db.Begin()

	if err != nil {
		return err
	} else {
		defer func() {
			if status {
				tx.Commit()
			} else {
				tx.Rollback()
			}
		}()
	}

	// 既存の計測に対処。
	measurementCode := fmt.Sprintf("%s-%s", patientIdPrefix, trc.PatientId)

	if e := (&S.MeasurementTxService{
		Service: nil,
		DB: tx,
		Influx: lib.GetInfluxDB(),
	}).EnsureNotExists(hospital.Id, measurementCode, !abortIfExists); e != nil {
		return e
	}

	// データ登録。
	ctgData := []S.CTGData{}

	dataLength := len(trc.FHR1)

	for i := 0; i < dataLength; i++ {
		timestamp := trc.StartTime.Add(trc.SamplingTime*time.Duration(i))

		ctgData = append(ctgData, S.CTGData{
			machineId,
			measurementCode,
			fmt.Sprintf("%d", trc.FHR1[i]),
			fmt.Sprintf("%d", trc.TOCO[i]),
			fmt.Sprintf("%d", trc.FHR2[i]),
			fmt.Sprintf("%d", int64(timestamp.UnixNano() / 1000000)),
		})
	}

	service := &S.DataTxService{
		Service: nil,
		DB: tx,
		Influx: lib.GetInfluxDB(),
	}

	if _, e := service.RegisterCTGData(hospital.Id, ctgData); e != nil {
		return e
	}

	status = true

	return nil
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

	// ディレクトリ移動
	var rootDir string

	if d, e := moveCurrentDir(); e != nil {
		log.Fatal(e)
	} else {
		rootDir = d
	}

	// ログファイル準備
	if clean, e := setupLogging(rootDir, "logs", "import-trc-%s.log"); e != nil {
		log.Fatal(e)
	} else {
		defer clean()
	}

	log.Printf("Arguments:\n")
	log.Printf("  Hospital UUID:    %s\n", args.hospitalUuid)
	log.Printf("  Machine ID:       %s\n", args.machineId)
	log.Printf("  TRC file:         %s\n", args.trcFile)
	log.Printf("  Abort if exists:  %v\n", args.abortIfExists)

	// 設定
	config.SetupAll()

	// データパーズ
	var trc *lib.TRCData

	if d, e := parseTRC(args.trcFile); e != nil {
		log.Fatal(e)
	} else {
		trc = d
	}

	log.Printf("Patient ID:    %s\n", trc.PatientId)
	log.Printf("Start time:    %v\n", trc.StartTime)
	log.Printf("Sampling time: %v\n", trc.SamplingTime)
	log.Printf("Data length:   %d\n", len(trc.FHR1))

	// データ登録
	if e := registerTRC(args.hospitalUuid, args.machineId, trc, args.abortIfExists); e != nil {
		log.Fatal(e)
	}

	fmt.Println("Success")
}