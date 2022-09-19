package main

import (
	"fmt"
	"gorm.io/gorm"
	"testing"
)

var (
	initData []interface{}
)

func TestMain(m *testing.M) {
	// 初始化 gorm，创建 t_deadlock 表
	StartGorm(MYSQL_5_7_38, Deadlock{})

	// 初始化 t_deadlock 表数据
	for _, column := range initData {
		if column != nil {
			db.Create(column)
		}
	}

	// 运行具体的单测
	fmt.Printf("\n\n\n\n------------------------------------------------------------\n")
	m.Run()
	fmt.Printf("\n------------------------------------------------------------\n\n\n\n")

	// 结束前删掉 t_deadlock 表
	db.Exec(fmt.Sprintf("drop table %s", Deadlock{}.TableName())) // ignore_security_alert
}

func TestDeadlock1(t *testing.T) {
	c := ConcurrentTrx{IgnoreDeadlock: true}
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1210).Delete(&Deadlock{}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1220).Delete(&Deadlock{}).Error
	})
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1210}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1220}).Error
	})
	c.Execute()
}
