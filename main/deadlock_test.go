package main

import (
	"fmt"
	"gorm.io/gorm"
	"testing"
)

func TestMain(m *testing.M) {
	// 初始化 gorm，创建 t_deadlock 表
	StartGorm(MYSQL_8_0_30, Deadlock{})

	// 运行具体的单测
	fmt.Printf("\n\n\n\n------------------------------------------------------------\n")
	m.Run()
	fmt.Printf("\n------------------------------------------------------------\n\n\n\n")

	// 结束前删掉 t_deadlock 表
	db.Exec(fmt.Sprintf("drop table %s", Deadlock{}.TableName())) // ignore_security_alert
}

// 以下用例来自：https://github.com/aneasystone/mysql-deadlocks

/*
| TRX 1                                      | TRX 2                                      |
|--------------------------------------------|--------------------------------------------|
| DELETE FROM `t_deadlock` WHERE uk = 1      |                                            |
|                                            | DELETE FROM `t_deadlock` WHERE uk = 2      |
| INSERT INTO `t_deadlock` (`uk`) VALUES (3) |                                            |
|                                            | INSERT INTO `t_deadlock` (`uk`) VALUES (4) |
*/
func TestDeadlock1(t *testing.T) {
	c := ConcurrentTrx{IgnoreDeadlock: true}
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1).Delete(&Deadlock{}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 2).Delete(&Deadlock{}).Error
	})
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 3}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 4}).Error
	})
	c.Execute()
}

/*
| TRX 1                                      | TRX 2                                      | TRX 3                                      |
|--------------------------------------------|--------------------------------------------|--------------------------------------------|
| INSERT INTO `t_deadlock` (`uk`) VALUES (1) |                                            |                                            |
|                                            | INSERT INTO `t_deadlock` (`uk`) VALUES (2) |                                            |
|                                            |                                            | INSERT INTO `t_deadlock` (`uk`) VALUES (3) |
| ROLLBACK                                   |                                            |                                            |
*/
func TestDeadlock2(t *testing.T) {
	c := ConcurrentTrx{IgnoreDeadlock: true}
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1}).Error
	})
	c.AddSQL(3, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1}).Error
	})
	c.Rollback(1)
	c.Execute()
}

/*
| TRX 1                                      | TRX 2                                 |
|--------------------------------------------|---------------------------------------|
| DELETE FROM `t_deadlock` WHERE uk = 1      |                                       |
|                                            | DELETE FROM `t_deadlock` WHERE uk = 1 |
| INSERT INTO `t_deadlock` (`uk`) VALUES (1) |                                       |

注：MySQL 8.0.30 不会出现死锁
*/
func TestDeadlock4(t *testing.T) {
	db.Create(&Deadlock{Uk: 1})

	c := ConcurrentTrx{IgnoreDeadlock: true}
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1).Delete(&Deadlock{}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1).Delete(&Deadlock{}).Error
	})
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Create(&Deadlock{Uk: 1}).Error
	})
	c.Execute()
}

/*
| TRX 1                                 | TRX 2                                 |
|---------------------------------------|---------------------------------------|
| DELETE FROM `t_deadlock` WHERE uk = 1 |                                       |
|                                       | DELETE FROM `t_deadlock` WHERE uk = 2 |
| DELETE FROM `t_deadlock` WHERE uk = 2 |                                       |
|                                       | DELETE FROM `t_deadlock` WHERE uk = 1 |
*/
func TestDeadlock8(t *testing.T) {
	db.Create(&[]Deadlock{{Uk: 1}, {Uk: 2}})

	c := ConcurrentTrx{IgnoreDeadlock: true}
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1).Delete(&Deadlock{}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 2).Delete(&Deadlock{}).Error
	})
	c.AddSQL(1, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 2).Delete(&Deadlock{}).Error
	})
	c.AddSQL(2, func(tx *gorm.DB) error {
		return tx.Where("uk = ?", 1).Delete(&Deadlock{}).Error
	})
	c.Execute()
}
