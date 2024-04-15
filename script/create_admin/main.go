package main

import (
	"flag"
	"fmt"
	"log"
	//"os"
	//"time"

	"gopkg.in/gorp.v2"

	"github.com/spiker/spiker-server/config"
	"github.com/spiker/spiker-server/lib"
	"github.com/spiker/spiker-server/route/shared"
	//"github.com/spiker/spiker-server/model"
	S "github.com/spiker/spiker-server/service"
)

func createAdmin(tx *gorp.Transaction, loginId string, password string) (string, error) {
	service := &S.AdministratorTxService{nil, tx}

	admin, err := service.Create(loginId, password)

	if err != nil {
		return "", err
	}

	return shared.CreateTokenWithStandardClaims(admin.LoginId, admin.TokenVersion), nil
}

func main() {
	config.SetupAll()

	// コマンドライン引数
	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		log.Fatal("usage: go run main.go [loginId] [password]")
	}

	loginId := args[0]
	password := args[1]

	// 管理者作成
	tx, err := lib.GetDB(lib.WriteDBKey).Begin()

	if err != nil {
		log.Fatal("Failed to open transaction: %v", err)
	}

	status := true

	defer func() {
		if e := recover(); e != nil {
			log.Println("Rollback database due to unhandled exception: %v", e)
			tx.Rollback()
		} else if !status {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if token, e := createAdmin(tx, loginId, password); e != nil {
		log.Println("Failed to create administrator: %v", e)
		status = false
	} else {
		fmt.Println("OK")
		fmt.Println(token)
	}
}