package lib

import (
	"fmt"
	"time"
	"unsafe"
)

type TRCFormat uint32

const (
	TRCFormat1000 TRCFormat = 0x1000
	TRCFormat0400 TRCFormat = 0x0400
)

type TRCData struct {
	PatientId    string
	StartTime    time.Time
	SamplingTime time.Duration
	FHR1         []int
	FHR2         []int
	TOCO         []int
}

func ParseTRC(
	data []byte,
) (*TRCData, error) {
	// データ開始位置: 0x04~ 4byte
	dataIndex := *(*uint32)(unsafe.Pointer(&data[4]))

	format := TRCFormat(dataIndex)

	if format != TRCFormat0400 && format != TRCFormat1000 {
		return nil, fmt.Errorf("Unexpected data index: 0x%08x", dataIndex)
	}

	// 患者ID: 0x60~ 20byte
	// 機種により、[00 xx 00 xx ...] or [xx 00 xx 00 ...] のいずれか。
	idBytes := []byte{}
	for i := 0x60; i < 0x74; i++ {
		if data[i] != 0x00 {
			idBytes = append(idBytes, data[i])
		}
	}
	patientId := string(idBytes)

	// 開始時刻: [ff]*12 後から数えて 0x64~ 8byte
	// 開始位置0x1000: 8byte(LE)を浮動小数点に変換して、日数として1899/12/30 00:00:00に加算
	// 開始位置0x0400: [00 秒 分 時 日 月 年(2byte,LE)]
	timeIndex := findTimeMarker(data[0x74:]) + 0x74 + 12 + 0x64

	if timeIndex < 0 {
		return nil, fmt.Errorf("Start time marker (ff*12) is not found")
	}

	timeBytes := data[timeIndex:timeIndex+8]

	var startTime time.Time

	if format == TRCFormat1000 {
		days := *(*float64)(unsafe.Pointer(&timeBytes[0]))

		diff := time.Duration(int64(float64(time.Hour * time.Duration(24)) * days))

		startTime = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC).Add(diff)
	} else {
		year := *(*uint16)(unsafe.Pointer(&timeBytes[6]))

		startTime = time.Date(
			int(year),
			time.Month(int(timeBytes[5])),
			int(timeBytes[4]),
			int(timeBytes[3]),
			int(timeBytes[2]),
			int(timeBytes[1]),
			0, time.UTC,
		)
	}

	// データ長: 開始時刻2回繰り返し後の4byte
	lenIndex := timeIndex + 0x10

	dataLength := *(*uint32)(unsafe.Pointer(&data[lenIndex]))

	// データ: データの開始位置から、データ長分のデータがFHR1,FHR2,TOCOの順に並ぶ。
	fhr1 := make([]int, dataLength)
	fhr2 := make([]int, dataLength)
	toco := make([]int, dataLength)

	var i uint32

	for i = 0; i < dataLength; i++ {
		value := data[dataIndex+i]
		if value == 0xff {
			fhr1[i] = 0
		} else {
			fhr1[i] = int(value)
		}
	}
	for i = 0; i < dataLength; i++ {
		value := data[dataIndex+dataLength+i]
		if value == 0xff {
			fhr2[i] = 0
		} else {
			fhr2[i] = int(value)
		}
	}
	for i = 0; i < dataLength; i++ {
		value := data[dataIndex+dataLength*2+i]
		if value == 0xff {
			toco[i] = 0
		} else {
			toco[i] = int(value)
		}
	}

	return &TRCData{patientId, startTime, time.Second, fhr1, fhr2, toco}, nil
}

// 0xffが12連続する最初のインデックスを取得する。
func findTimeMarker(data []byte) int {
	index := 0

	end := len(data) - 11

	for index < end {
		match := true

		for i := 0; i < 12; i++ {
			if data[index+i] != 0xff {
				index += i+1
				match = false
				break
			}
		}

		if match {
			return index
		}
	}

	return -1
}