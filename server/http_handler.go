package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kaytu-io/pennywise/server/internal/mysql"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type HttpHandler struct {
	backend *mysql.Backend
	logger  *zap.Logger
}

func InitializeHttpHandler(
	logger *zap.Logger,
	mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDb string,
) (*HttpHandler, error) {
	logger.Info("Initializing http handler")

	logger.Info("Connecting to database and creating backend")
	dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true", mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDb)
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		logger.Error("Error while connecting to db", zap.Error(err))
	}
	err = mysql.Migrate(context.Background(), db, "pricing_migrations")

	backend := mysql.NewBackend(db)

	return &HttpHandler{
		logger:  logger,
		backend: backend,
	}, nil
}
