package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	S "github.com/spiker/spiker-server/service"
)

const (
	endpointUrl = "http://localhost:1323/ctg/data"
	apiKey = "EOtd08fod-EKxV8PU7jyNw"
)

func readCsv(file string) ([][]string, error) {
	if f, e := os.Open(file); e != nil {
		return nil, e
	} else {
		defer f.Close()

		if rs, e := csv.NewReader(f).ReadAll(); e != nil {
			return nil, e
		} else {
			return rs, nil
		}
	}
}

type ctgRow struct {
	mid string
	pid string
	val int
	timestamp int64
}

func row2ctg(row []string) (*ctgRow, error) {
	ctg := &ctgRow{
		mid: row[0],
		pid: row[1],
		val: 0,
		timestamp: 0,
	}

	if v, e := strconv.Atoi(row[2]); e != nil {
		return nil, e
	} else {
		ctg.val = v
	}

	if v, e := strconv.ParseInt(row[3], 10, 64); e != nil {
		return nil, e
	} else {
		ctg.timestamp = v
	}

	return ctg, nil
}

func readCTG(hrFile string, ucFile string, currentMilli int64) ([]S.CTGData, error) {
	var hrs [][]string
	var ucs [][]string

	if rs, e := readCsv(hrFile); e != nil {
		return nil, e
	} else {
		hrs = rs
	}

	if rs, e := readCsv(ucFile); e != nil {
		return nil, e
	} else {
		ucs = rs
	}

	if len(hrs) == 0 || len(ucs) == 0 {
		return nil, fmt.Errorf("No data: %d, %d", len(hrs), len(ucs))
	}

	hrRows := []*ctgRow{}
	ucRows := []*ctgRow{}

	for _, row := range hrs {
		if v, e := row2ctg(row); e != nil {
			return nil, e
		} else {
			hrRows = append(hrRows, v)
		}
	}

	for _, row := range ucs {
		if v, e := row2ctg(row); e != nil {
			return nil, e
		} else {
			ucRows = append(ucRows, v)
		}
	}

	// 先頭の時刻を決める。
	start := hrRows[0].timestamp
	if start > ucRows[0].timestamp {
		start = ucRows[0].timestamp
	}

	data := []S.CTGData{}

	hi := 0
	ui := 0

	for hi < len(hrRows) && ui < len(ucRows) {
		if hrRows[hi].timestamp == ucRows[ui].timestamp {
			ts := currentMilli + hrRows[hi].timestamp - start

			data = append(data, S.CTGData{
				MachineId: hrRows[hi].mid,
				PatientId: hrRows[hi].pid,
				FHR1: fmt.Sprintf("%d", hrRows[hi].val),
				UC: fmt.Sprintf("%d", ucRows[ui].val),
				FHR2: "0",
				Timestamp: fmt.Sprintf("%d", ts),
			})
			hi++
			ui++
		} else if hrRows[hi].timestamp > ucRows[ui].timestamp {
			// UCを進める。
			ui++
		} else {
			// HRを進める。
			hi++
		}
	}

	return data, nil
}

func postData(data []S.CTGData) {
	fmt.Printf("Request to server: %d data\n", len(data))

	body, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	//log.Println(string(body))

	req, err := http.NewRequest("POST", endpointUrl, bytes.NewBuffer(body))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	if r, e := (&http.Client{}).Do(req); e != nil {
		log.Fatal(e)
	} else {
		log.Printf("Status = %s", r.Status)
	}
}

func iterateRequest(data []S.CTGData) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	done := make(chan bool)
	defer close(done)

	for {
		select {
		case <- ticker.C:
			now := time.Now()

			index := -1

			for i, d := range data {
				if ts, e := d.GetTimestamp(); e != nil {
					return e
				} else if ts > now.Unix() * 1000 {
					index = i
					break
				}
			}

			if index < 0 {
				go func() {
					postData(data)
					done <- true
				}()
			} else if index > 0 {
				req := data[0:index]
				data = data[index:]

				go func() {
					postData(req)
				}()
			}
		case <- done:
			return nil
		}
	}
}

func main() {
	flag.Parse()

	files := flag.Args()

	if len(files) != 2 {
		log.Fatal("usage: go run script/clinet_simulator/main.go [hr_file] [uc_file]")
	}

	hrFile, err := filepath.Abs(files[0])
	if err != nil {
		log.Fatal(err)
	}

	ucFile, err := filepath.Abs(files[1])
	if err != nil {
		log.Fatal(err)
	}

	current := time.Now().Unix() * 1000

	data, err := readCTG(hrFile, ucFile, current)
	if err != nil {
		log.Fatal(err)
	}

	iterateRequest(data)

	log.Println("Finish")
}