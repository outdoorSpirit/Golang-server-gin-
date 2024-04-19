package service

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gorp.v2"

	"github.com/spiker/spiker-server/model"
	"github.com/spiker/spiker-server/resource/rds"
)

type AssessmentService struct {
	*Service
	DB *gorp.DbMap
}

type diagnosisItem struct {
	rangeFrom int64
	rangeUntil int64
	parameters map[string]interface{}
}

var (
	mkdirChannel = make(chan bool, 1)
)

// 診断処理を行う。
//
// measurementは、診断対象となる計測記録。
//
// diagnosisTimeは診断対象の日時を示し、ここからdurationだけ遡った期間のデータを対象とする。
//
// rootDirは診断処理プログラムの配置されるディレクトリ。この下のinputディレクトリにデータを整形したcsvファイルが出力される。
// commandはプログラム実行ファイル、parametersFileは設定ファイル。いずれもrootDir以下のパスを与える。
//
// algorithmNameとalgorithmVersionは、利用するアルゴリズムを識別する値。
// 現時点ではアルゴリズムによる挙動の変更はなく、単に診断結果として残されるのみ。
func (s *AssessmentService) Execute(
	measurement *model.DiagnosisMeasurmentEntity,
	diagnosisTime time.Time,
	duration time.Duration,
	rootDir string,
	command string,
	parametersFile string,
	algorithmName string,
	algorithmVersion string,
) (*model.DiagnosisEntity, error) {
	algorithm, err := rds.FetchAlgorithmByName(s.DB, algorithmName, algorithmVersion)

	if err != nil {
		return nil, err
	} else if algorithm == nil {
		return nil, fmt.Errorf("No algorithm found for '%s:%s'", algorithmName, algorithmVersion)
	}

	inputDir, _, err := s.prepareDirectories(rootDir)

	if err != nil {
		return nil, err
	}

	// 心拍。
	hrPath := filepath.Join(
		inputDir,
		fmt.Sprintf("%s-%d-HR.csv", diagnosisTime.Format("20060102-150405"), measurement.Id),
	)

	if e := s.prepareInputFile(hrPath, func() [][]string {
		data := [][]string{[]string{"RecordTime", "F1"}}

		for _, v := range measurement.HeartRates {
			data = append(data, []string{
				fmt.Sprintf("%d", v.Timestamp.UnixNano() / 1000000),
				fmt.Sprintf("%d", v.Value),
			})
		}

		return data
	}); e != nil {
		return nil, e
	}

	// TOCO。
	ucPath := filepath.Join(
		inputDir,
		fmt.Sprintf("%s-%d-UC.csv", diagnosisTime.Format("20060102-150405"), measurement.Id),
	)

	if e := s.prepareInputFile(ucPath, func() [][]string {
		data := [][]string{[]string{"RecordTime", "UC"}}

		for _, v := range measurement.TOCOs {
			data = append(data, []string{
				fmt.Sprintf("%d", v.Timestamp.UnixNano() / 1000000),
				fmt.Sprintf("%d", v.Value),
			})
		}

		return data
	}); e != nil {
		return nil, e
	}

	// コマンド実行。
	bpm := "null"
	if measurement.LatestDiagnosis != nil {
		if b := measurement.LatestDiagnosis.BaselineBpm; b != nil {
			bpm = fmt.Sprintf("%d", *b)
		}
	}

	commandPath := filepath.Join(rootDir, command)
	parametersPath := filepath.Join(rootDir, parametersFile)

	cmd := exec.Command(commandPath, hrPath, ucPath, parametersPath, bpm)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	if e := cmd.Start(); e != nil {
		return nil, e
	}

	items, err := s.parseOutput(stdout)

	if err != nil {
		return nil, err
	}

	if e := cmd.Wait(); e != nil {
		return nil, e
	}

	now := time.Now()

	var baselineBpm *int = nil
	var maximumRisk *int = nil

	contents := []*model.DiagnosisContent{}

	for _, i := range items {
		paramsJson, _ := json.Marshal(i.parameters)

		dc := &model.DiagnosisContent{
			Risk: nil,
			RangeFrom: time.Unix(i.rangeFrom / 1000, (i.rangeFrom % 1000) * 1000000),
			RangeUntil: time.Unix(i.rangeUntil / 1000, (i.rangeUntil % 1000) * 1000000),
			Parameters: model.JSON(paramsJson),
		}

		if v, be := i.parameters["Baseline-NORMAL"]; be {
			if bb, ok := v.(int); ok {
				baselineBpm = &bb
			}
		}

		if v, be := i.parameters["Risk"]; be {
			if risk, ok := v.(int); ok {
				dc.Risk = &risk

				if maximumRisk == nil || risk > *maximumRisk {
					maximumRisk = &risk
				}
			}
		}

		contents = append(contents, dc)
	}

	diagnosis := &model.Diagnosis{
		MeasurementId: measurement.Id,
		BaselineBpm: baselineBpm,
		MaximumRisk: maximumRisk,
		Memo: "",
		RangeFrom: diagnosisTime.Add(-duration),
		RangeUntil: diagnosisTime,
		CreatedAt: now,
		ModifiedAt: now,
	}

	return &model.DiagnosisEntity{
		Diagnosis: diagnosis,
		Algorithm: algorithm,
		Contents: contents,
	}, nil
}

func (s *AssessmentService) prepareDirectories(
	root string,
) (string, string, error) {
	if !filepath.IsAbs(root) {
		return "", "", fmt.Errorf("Root directory must be configured by absolute path: %s", root)
	}

	now := time.Now()

	inputDir := filepath.Join(
		root,
		"input",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	mkdirChannel <- true

	defer func() {
		<-mkdirChannel
	}()

	if info, e := os.Stat(inputDir); e != nil {
		// 存在しないため作成。
		if e := os.MkdirAll(inputDir, 0777); e != nil {
			return "", "", e
		}
	} else if !info.IsDir() {
		return "", "", fmt.Errorf("Path to input directory does not indicate a directory: %s", inputDir)
	}

	// ファイルの出力は行わない。
	//outputDir := filepath.Join(root, "output")

	//if info, e := os.Stat(outputDir); e != nil {
	//	// 存在しないため作成。
	//	if e := os.Mkdir(outputDir, 0777); e != nil {
	//		return "", "", e
	//	}
	//} else if !info.IsDir() {
	//	return "", "", fmt.Errorf("Path to output directory does not indicate a directory: %s", outputDir)
	//}

	return inputDir, "", nil
}

func (s *AssessmentService) prepareInputFile(
	path string,
	generator func() [][]string,
) error {
	if f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755); e != nil {
		return e
	} else {
		defer f.Close()

		w := csv.NewWriter(f)

		data := generator()

		w.WriteAll(data)
		w.Flush()

		return nil
	}
}

func (s *AssessmentService) parseOutput(reader io.Reader) ([]*diagnosisItem, error) {
	scanner := bufio.NewScanner(reader)

	items := []*diagnosisItem{}

	// 1行読み捨て。
	scanner.Scan()

	var currentItem *diagnosisItem = nil

	for scanner.Scan() {
		line := scanner.Text()

		if line == "Data End" {
			break
		}

		tokens := strings.Split(line, " ")

		if len(tokens) < 4 || tokens[1] != "-" {
			log.Printf("Unexpected line: %s\n", line)
			continue
		}

		var rangeFrom int64
		var rangeUntil int64

		if v, e := strconv.ParseInt(tokens[0], 10, 64); e != nil {
			log.Printf("Unexpected line: %s\n", line)
			continue
		} else {
			rangeFrom = v
		}

		if v, e := strconv.ParseInt(tokens[2], 10, 64); e != nil {
			log.Printf("Unexpected line: %s\n", line)
			continue
		} else {
			rangeUntil = v
		}

		if currentItem == nil || (currentItem.rangeFrom != rangeFrom || currentItem.rangeUntil != rangeUntil) {
			if currentItem != nil {
				items = append(items, currentItem)
			}

			currentItem = &diagnosisItem{
				rangeFrom: rangeFrom,
				rangeUntil: rangeUntil,
				parameters: map[string]interface{}{},
			}
		}

		var parameter interface{} = nil

		if len(tokens) >= 5 {
			// 数値ならば数値に変換する。
			if v, e := strconv.Atoi(tokens[4]); e == nil {
				parameter = v
			} else if v, e := strconv.ParseFloat(tokens[4], 64); e == nil {
				parameter = v
			} else {
				parameter = tokens[4]
			}
		}

		currentItem.parameters[tokens[3]] = parameter
	}

	if currentItem != nil {
		items = append(items, currentItem)
	}

	return items, nil
}