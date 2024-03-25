package lib

import (
	"fmt"
	"context"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	Q "github.com/influxdata/influxdb-client-go/v2/api/query"
	influxdb_log "github.com/influxdata/influxdb-client-go/v2/log"
)

//----------------------------------------------------------------
// 設定関連
//----------------------------------------------------------------
type InfluxDBConfiguration struct {
	Url           string
	Token         string
	Organization  string
	LogLevel      int
}

func (cfg *InfluxDBConfiguration) String() string {
	return fmt.Sprintf(`[InfluxDB]
Url:          %v
Organization: %v
LogLevel:     %v`, cfg.Url, cfg.Organization, cfg.LogLevel)
}

type configuredClient struct {
	influxdb2.Client
	configuration *InfluxDBConfiguration
}

var defaultClient *configuredClient

func SetupInfluxDB(cfg *InfluxDBConfiguration) error {
	options := influxdb2.DefaultOptions()

	switch uint(cfg.LogLevel) {
	case influxdb_log.ErrorLevel:
		options.SetLogLevel(influxdb_log.ErrorLevel)
	case influxdb_log.WarningLevel:
		options.SetLogLevel(influxdb_log.WarningLevel)
	case influxdb_log.InfoLevel:
		options.SetLogLevel(influxdb_log.InfoLevel)
	case influxdb_log.DebugLevel:
		options.SetLogLevel(influxdb_log.DebugLevel)
	}

	client := influxdb2.NewClientWithOptions(cfg.Url, cfg.Token, options)

	if isReady, e := client.Ready(context.Background()); e != nil {
		return e
	} else if !isReady {
		return fmt.Errorf("InfluxDB is not ready yet")
	}

	defaultClient = &configuredClient{client, cfg}

	return nil
}

var pointFactories = map[string]func()Point{}

func RegisterPointType(factory func()Point) {
	point := factory()
	pointFactories[point.Measurement()] = factory
}

func UnregisterPointType(factory func()Point) {
	point := factory()
	delete(pointFactories, point.Measurement())
}

func GetInfluxDB() InfluxDBClient {
	return defaultClient
}

//----------------------------------------------------------------
// プリミティブ関数
//----------------------------------------------------------------
func GetWriteAPI(bucket string) api.WriteAPI {
	return defaultClient.WriteAPI(defaultClient.configuration.Organization, bucket)
}

func GetQueryAPI() api.QueryAPI {
	return defaultClient.QueryAPI(defaultClient.configuration.Organization)
}

//----------------------------------------------------------------
// InfluxDBアクセサ
//----------------------------------------------------------------
type PointRecord struct {
	Measurement string
	Tags        map[string]string
	Field       string
	Value       interface{}
	Timestamp   time.Time
}

type SchemaRecord struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

type Point interface {
	Measurement() string
	FromRecord(*PointRecord) error
	ToRecord(*SchemaRecord)
}

type RecordConsumer func(int, string, *PointRecord)error

type InfluxDBClient interface {
	BaseClient() influxdb2.Client
	Insert(bucket string, points ...Point) []error
	Delete(bucket string, start, stop time.Time, predicate string) error
	DeleteAll(bucket string, predicate string) error
	Select(query string, consumer RecordConsumer) error
}

func (c *configuredClient) BaseClient() influxdb2.Client {
	return c.Client
}

func (c *configuredClient) Insert(bucket string, points ...Point) []error {
	api := c.Client.WriteAPI(c.configuration.Organization, bucket)

	errors := []error{}
	errorChannel := api.Errors()

	go func() {
		for e := range errorChannel {
			errors = append(errors, e)
		}
	}()

	for _, p := range points {
		record := SchemaRecord{"", map[string]string{}, map[string]interface{}{}, time.Unix(0, 0)}

		p.ToRecord(&record)

		api.WritePoint(influxdb2.NewPoint(
			p.Measurement(),
			record.Tags,
			record.Fields,
			record.Timestamp,
		))
	}

	api.Flush()

	return errors
}

func (c *configuredClient) Delete(bucket string, start, stop time.Time, predicate string) error {
	api := c.Client.DeleteAPI()

	return api.DeleteWithName(context.Background(), c.configuration.Organization, bucket, start, stop, predicate)
}

func (c *configuredClient) DeleteAll(bucket string, predicate string) error {
	// エポック日時から現在の100年後の期間に全てのデータが含まれると見なす。
	start := time.Unix(0, 0)
	stop := time.Now().Add(time.Duration(24*365*100)*time.Hour)

	return c.Delete(bucket, start, stop, predicate)
}

func (c *configuredClient) Select(query string, consumer RecordConsumer) error {
	api := c.Client.QueryAPI(c.configuration.Organization)

	result, err := api.Query(context.Background(), query)

	if err != nil {
		return err
	}

	return mapResult(result, consumer)
}

func PointConsumer(consumer func(Point, string)error) RecordConsumer {
	return func(i int, f string, r *PointRecord) error {
		factory, be := pointFactories[r.Measurement]

		if !be {
			return fmt.Errorf("No factory is registered for measurement '%s'", r.Measurement)
		}

		point := factory()

		if e := point.FromRecord(r); e != nil {
			return e
		} else {
			return consumer(point, f)
		}
	}
}

func mapResult(result *api.QueryTableResult, consumer RecordConsumer) error {
	defer result.Close()

	var meta *Q.FluxTableMetadata = nil
	tags := []string{}

	index := 0

	for result.Next() {
		if result.TableChanged() {
			meta = result.TableMetadata()

			tags = []string{}

			for _, c := range meta.Columns() {
				name := c.Name()
				if !strings.HasPrefix(name, "_") && name != "table" && name != "result" {
					tags = append(tags, name)
				}
			}
		}

		record := result.Record()

		r := &PointRecord{
			Measurement: record.Measurement(),
			Tags:        map[string]string{},
			Field:       record.Field(),
			Value:       record.Value(),
			Timestamp:   record.Time(),
		}

		values := record.Values()

		for _, t := range tags {
			r.Tags[t] = values[t].(string)
		}

		consumer(index, record.Field(), r)

		index++
	}

	return nil
}