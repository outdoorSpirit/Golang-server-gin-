package lib

import (
	"fmt"
	"context"
	"log"
	"os"
	"path"
	//"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/influxdata/influxdb-client-go/v2"
)

const (
	testBucket string = "spiker"
)

type TestMeasurement struct {
	Tag1   string
	Tag2   string
	Field1 string
	Field2 int64
	Now    time.Time
}

func (p TestMeasurement) Measurement() string {
	return "unittest"
}

func (p *TestMeasurement) ToRecord(record *SchemaRecord) {
	record.Tags["tag1"] = p.Tag1
	record.Tags["tag2"] = p.Tag2
	record.Fields["field1"] = p.Field1
	record.Fields["field2"] = p.Field2
	record.Timestamp = p.Now
}

func (p *TestMeasurement) FromRecord(record *PointRecord) (err error) {
	p.Tag1 = record.Tags["tag1"]
	p.Tag2 = record.Tags["tag2"]

	switch record.Field {
	case "field1":
		p.Field1 = record.Value.(string)
	case "field2":
		p.Field2 = record.Value.(int64)
	}
	p.Now = record.Timestamp
	return
}

var (
	epochPast time.Time = time.Unix(0, 0)
	epochFuture time.Time = time.Now().Add(time.Duration(100*365*24)*time.Hour)
)

func TestMain(m *testing.M) {
	root := os.Getenv("SERVER_ROOT")

	paths := []string{
		path.Join(root, "data/config", ".env.test"),
		path.Join(root, "data/config", ".env.local.test"),
	}
	if e := godotenv.Load(paths...); e != nil {
		log.Fatalf("Failed to load %v: %v\n", paths, e)
	}

	config := InfluxDBConfiguration{}

	if e := envconfig.Process("influxdb", &config); e != nil {
		log.Fatalf("Failed to process: %v\n", e)
	}

	SetupInfluxDB(&config)

	m.Run()
}

func TestInfluxDB_Insert(t *testing.T) {
	client := GetInfluxDB()

	client.Delete(testBucket, epochPast, epochFuture, "")

	errors := client.Insert(
		testBucket,
		&TestMeasurement{"abc", "xyz", "ABC", 100, time.Date(2021, time.May, 1, 11, 22, 33, 0, time.UTC)},
		&TestMeasurement{"def", "uvw", "DEF", 200, time.Date(2021, time.May, 2, 11, 22, 33, 0, time.UTC)},
		&TestMeasurement{"ghi", "rst", "GHI", 300, time.Date(2021, time.May, 3, 11, 22, 33, 0, time.UTC)},
	)

	assert.EqualValues(t, 0, len(errors))

	result, err := client.BaseClient().QueryAPI("test-spiker").Query(
		context.Background(),
		fmt.Sprintf(`from(bucket:"%s") |> range(start: 2021-05-01, stop:2021-06-01)`, testBucket),
	)

	assert.NoError(t, err)

	if e := result.Err(); e != nil {
		assert.FailNow(t, e.Error())
	}

	records := []PointRecord{}

	for result.Next() {
		r := result.Record()

		values := r.Values()

		p := PointRecord{
			r.Measurement(),
			map[string]string{
				"tag1": values["tag1"].(string),
				"tag2": values["tag2"].(string),
			},
			r.Field(),
			r.Value(),
			r.Time(),
		}

		records = append(records, p)
	}

	assert.EqualValues(t, []PointRecord{
		PointRecord{
			"unittest",
			map[string]string{"tag1": "abc", "tag2": "xyz"},
			"field1",
			"ABC",
			time.Date(2021, time.May, 1, 11, 22, 33, 0, time.UTC),
		},
		PointRecord{
			"unittest",
			map[string]string{"tag1": "abc", "tag2": "xyz"},
			"field2",
			int64(100),
			time.Date(2021, time.May, 1, 11, 22, 33, 0, time.UTC),
		},
		PointRecord{
			"unittest",
			map[string]string{"tag1": "def", "tag2": "uvw"},
			"field1",
			"DEF",
			time.Date(2021, time.May, 2, 11, 22, 33, 0, time.UTC),
		},
		PointRecord{
			"unittest",
			map[string]string{"tag1": "def", "tag2": "uvw"},
			"field2",
			int64(200),
			time.Date(2021, time.May, 2, 11, 22, 33, 0, time.UTC),
		},
		PointRecord{
			"unittest",
			map[string]string{"tag1": "ghi", "tag2": "rst"},
			"field1",
			"GHI",
			time.Date(2021, time.May, 3, 11, 22, 33, 0, time.UTC),
		},
		PointRecord{
			"unittest",
			map[string]string{"tag1": "ghi", "tag2": "rst"},
			"field2",
			int64(300),
			time.Date(2021, time.May, 3, 11, 22, 33, 0, time.UTC),
		},
	}, records)
}

func TestInfluxDB_Select(t *testing.T) {
	client := GetInfluxDB()

	factory := func() Point { return &TestMeasurement{} }

	RegisterPointType(factory)
	defer UnregisterPointType(factory)

	client.Delete(testBucket, epochPast, epochFuture, "")

	api := client.BaseClient().WriteAPI("test-spiker", testBucket)

	baseTime := time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		api.WritePoint(influxdb2.NewPoint(
			"unittest",
			map[string]string{"tag1": fmt.Sprintf("aa%d", i), "tag2": fmt.Sprintf("ab%d", i)},
			map[string]interface{}{"field1": fmt.Sprintf("F%d", i), "field2": 100*(i+1)},
			baseTime.Add(time.Duration(i) * time.Hour),
		))
	}

	api.Flush()

	records := []*TestMeasurement{}
	
	client.Select(
		fmt.Sprintf(`from(bucket:"%s") |> range(start: 2021-05-01, stop:2021-06-01)`, testBucket),
		PointConsumer(func(point Point, field string) error {
			records = append(records, point.(*TestMeasurement))
			return nil
		}),
	)

	assert.EqualValues(t, []*TestMeasurement{
		&TestMeasurement{"aa0", "ab0", "F0", 0, time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC)},
		&TestMeasurement{"aa0", "ab0", "", 100, time.Date(2021, time.May, 1, 0, 0, 0, 0, time.UTC)},
		&TestMeasurement{"aa1", "ab1", "F1", 0, time.Date(2021, time.May, 1, 1, 0, 0, 0, time.UTC)},
		&TestMeasurement{"aa1", "ab1", "", 200, time.Date(2021, time.May, 1, 1, 0, 0, 0, time.UTC)},
		&TestMeasurement{"aa2", "ab2", "F2", 0, time.Date(2021, time.May, 1, 2, 0, 0, 0, time.UTC)},
		&TestMeasurement{"aa2", "ab2", "", 300, time.Date(2021, time.May, 1, 2, 0, 0, 0, time.UTC)},
	}, records)
}