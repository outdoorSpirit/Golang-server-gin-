package config

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/model"
)

const (
	// dataBasePath 設定ファイルのベースパス。
	dataBasePath = "data/config"
)

var appConfig *configuration

// appConfiguration アプリケーション設定
//  `.env.{SERVER_ENV}` ファイルに含まれる設定値を取得し管理する
type configuration struct {
	Server     ServerConfiguration
	S3Config   AwsS3Config
	DB         lib.DatabaseConfiguration
	ReadDB     lib.DatabaseConfiguration
	Lang       lib.LanguageConfiguration
	JWT        lib.JWTConfiguration
	InfluxDB   lib.InfluxDBConfiguration
	Assessment AssessmentConfiguration
}

// ServerConfig サーバ設定情報。
type ServerConfiguration struct {
	Port       string
	Dump       bool
	ApiVersion string `envconfig:"API_VERSION"`
}

// AwsS3Config S3設定値
type AwsS3Config struct {
	Region         string
	Bucket         string
	Endpoint       string
	Expires        int `envconfig:"AWS_S3_PRESIGNED_EXPIRES"`
	ExpiresForPush int `envconfig:"AWS_S3_PRESINGED_EXPIRES_FOR_PUSH"`
}

type AssessmentConfiguration struct {
	Root       string
	Command    string
	Parameters string
	Algorithm  string
	Version    string
	Duration   int
	Interval   int
	Cutoff     int
	Delay      int
}

func SetupAll() {
	if appConfig == nil {
		env := strings.ToLower(os.Getenv("SERVER_ENV"))
		if len(env) == 0 {
			env = "test"
		}

		root := os.Getenv("SERVER_ROOT")

		paths := []string{path.Join(root, dataBasePath, ".env."+env)}
		if env != "test" {
			paths = append(paths, path.Join(root, dataBasePath, ".env.local"))
		} else {
			paths = append(paths, path.Join(root, dataBasePath, ".env.local.test"))
		}
		if err := godotenv.Load(paths...); err != nil {
			log.Fatalf("Failed to load %v: %v\n", paths, err)
		}

		load := func(prefix string, config interface{}) {
			err := envconfig.Process(prefix, config)
			if err != nil {
				log.Printf("An error occured during loading %#v\n", err)
			}
		}

		appConfig = &configuration{}
		load("server", &appConfig.Server)
		load("aws_s3", &appConfig.S3Config)
		load("db", &appConfig.DB)
		load("read_db", &appConfig.ReadDB)
		load("lang", &appConfig.Lang)
		load("jwt", &appConfig.JWT)
		load("influxdb", &appConfig.InfluxDB)
		load("ctg_assessment", &appConfig.Assessment)

		if env != "test" {
			log.Println(&appConfig.DB)
			log.Println(&appConfig.ReadDB)
			log.Println(&appConfig.JWT)
			log.Println(&appConfig.InfluxDB)
		}

		// Read/Write用DBの設定
		if err := lib.SetupDatabase(lib.WriteDBKey, &appConfig.DB); err != nil {
			log.Fatalf("Failed to setup default database %v\n", err.Error())
		}
		// Read用DBの設定
		if err := lib.SetupDatabase(lib.ReadDBKey, &appConfig.ReadDB); err != nil {
			log.Fatalf("Failed to setup read database %v\n", err.Error())
		}

		if err := lib.SetupInfluxDB(&appConfig.InfluxDB); err != nil {
			log.Fatalf("Failed to setup influxDB %v\n", err.Error())
		}
		if err := lib.SetupAuthentication(&appConfig.JWT); err != nil {
			log.Fatalf("Failed to setup authentication %v\n", err.Error())
		}

		lib.SetupI18n(&appConfig.Lang)

		model.SetupModels()

		setLogger()
	}
}

type ContextHook struct{}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(10); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Data["source"] = fmt.Sprintf("%s:%v:%s", path.Base(file), line, path.Base(funcName))
	}

	return nil
}

func setLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}

func ServerConfig() *ServerConfiguration {
	return &appConfig.Server
}

func AssessmentConfig() *AssessmentConfiguration {
	return &appConfig.Assessment
}