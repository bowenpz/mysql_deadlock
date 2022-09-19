package main

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// docker 启动命令：
// docker run --name mysql-5.7.38 -e MYSQL_ROOT_PASSWORD=123456 -p 33061:3306 -d mysql:5.7.38
// docker run --name mysql-8.0.0 -e MYSQL_ROOT_PASSWORD=123456 -p 33062:3306 -d mysql:8.0.0
// docker run --name mysql-8.0.29 -e MYSQL_ROOT_PASSWORD=123456 -p 33063:3306 -d mysql:8.0.29

const (
	MYSQL_5_7_38 int = iota
	MYSQL_8_0_0
	MYSQL_8_0_29
)

var (
	db *gorm.DB

	dbUser     = "root"
	dbPasswd   = "123456"
	dbHost     = "127.0.0.1"
	dbPort5738 = "33061"
	dbPort800  = "33062"
	dbPort8029 = "33063"
	dbName     = "mysql"
)

func StartGorm(version int, migrateTables ...interface{}) {
	var dbPort string
	switch version {
	case MYSQL_5_7_38:
		dbPort = dbPort5738
	case MYSQL_8_0_0:
		dbPort = dbPort800
	case MYSQL_8_0_29:
		dbPort = dbPort8029
	}
	mysqlConfig := mysql.New(mysql.Config{
		DSN: fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPasswd, dbHost, dbPort, dbName),
	})
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Info),
	}
	var err error
	if db, err = gorm.Open(mysqlConfig, gormConfig); err != nil {
		panic(fmt.Errorf("failed to start gorm, err: %v", err))
	}

	// 自动迁移表
	if len(migrateTables) > 0 {
		autoDB := db.Set("gorm:table_options", "ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 ROW_FORMAT = Dynamic")
		if err = autoDB.AutoMigrate(migrateTables...); err != nil {
			panic(fmt.Errorf("failed to migrate tables, err: %v", err))
		}
	}
}
