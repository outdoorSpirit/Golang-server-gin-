package lib

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"gopkg.in/gorp.v2"
)

const (
	WriteDBKey string = "write_db"
	ReadDBKey         = "read_db"
)

// Database データース設定。
type DatabaseConfiguration struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	Maxconns int `envconfig:"MAX_CONNS"`
	Maxidles int `envconfig:"MAX_IDLES"`
	Lifetime int
	Debug    bool
}

func (db *DatabaseConfiguration) String() string {
	return fmt.Sprintf(`[Database]
Host:           %v
Port:           %v
Dbname:         %v
MaxConnections: %v
MaxIdles:       %v`, db.Host, db.Port, db.Name, db.Maxconns, db.Maxidles)
}

func (db *DatabaseConfiguration) connection() string {
	return fmt.Sprintf(
		"host=%v port=%v user=%v password=%v dbname=%v sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.Name,
	)
}

var databases = map[string]*gorp.DbMap{}

func SetupDatabase(name string, db *DatabaseConfiguration) error {
	if _, ok := databases[name]; ok {
		return nil
	}

	conn, err := sql.Open("postgres", db.connection())
	if err != nil {
		log.Fatalln(err)
	}
	if db.Maxconns > 0 {
		conn.SetMaxOpenConns(db.Maxconns)
	}
	if db.Maxidles > 0 {
		conn.SetMaxIdleConns(db.Maxidles)
	}
	if err := conn.Ping(); err != nil {
		log.Fatalf("database connection error: %v", err)
	}

	dbmap := &gorp.DbMap{Db: conn, Dialect: gorp.PostgresDialect{}, ExpandSliceArgs: true}

	if db.Debug {
		dbmap.TraceOn("", log.New(os.Stdout, "[Gorp] ", log.Ltime))
	}

	databases[name] = dbmap

	return nil
}

func GetDB(name string) *gorp.DbMap {
	return databases[name]
}
