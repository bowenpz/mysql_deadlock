# mysql_deadlock

本项目用于复现死锁。



## TL;DR

1. 启动 MySQL

   ```bash
   docker run --name mysql-5.7.38 -e MYSQL_ROOT_PASSWORD=123456 -p 33061:3306 -d mysql:5.7.38
   ```

2. 执行单测 `main/deadlock_test.go`

3. 查看日志中的死锁信息

   ```text
   ╔=============================╗
   ║ Table: `mysql`.`t_deadlock` ║
   ║ Index: uk                   ║
   ╚=============================╝
   
   -------------------
   TRANSACTIONS  47349
   -------------------
   sql:       INSERT INTO `t_deadlock` (`uk`,`idx`) VALUES (?,?)
   wait lock: lock_mode X insert intention waiting
   
   -------------------
   TRANSACTIONS  47350
   -------------------
   sql:       INSERT INTO `t_deadlock` (`uk`,`idx`) VALUES (?,?)
   wait lock: lock_mode X insert intention waiting
   hold lock: lock_mode X
   ```



## 项目代码

* `main/deadlock_test.go` 是测试用例，用于复现死锁，这是本项目的入口

* `main/config.go` 用于配置 gorm（一款 go 的 ORM框架）

  使用 `StartGorm()` 方法启动 gorm，调用方法前需要提前启动好 MySQL Server

* `main/models.go` 定义了 t_deadlock 表结构

* `main/concurrent_trx.go` 实现了并发事务，将按照 `AddSQL()` 的方法调用顺序执行 SQL

  例如：

  ```go
  c := ConcurrentTrx{}
  c.AddSQL(1, func(tx *gorm.DB) error {
    return tx.Find(&Deadlock{ID: 1}).Error
  })
  c.AddSQL(2, func(tx *gorm.DB) error {
    return tx.Find(&Deadlock{ID: 2}).Error
  })
  c.AddSQL(1, func(tx *gorm.DB) error {
    return tx.Find(&Deadlock{ID: 3}).Error
  })
  c.Execute()
  ```

  | 事务 1                                                 | 事务 2                                                 |
  | ------------------------------------------------------ | ------------------------------------------------------ |
  | SELECT * FROM `t_deadlock` WHERE `t_deadlock`.`id` = 1 |                                                        |
  |                                                        | SELECT * FROM `t_deadlock` WHERE `t_deadlock`.`id` = 2 |
  | SELECT * FROM `t_deadlock` WHERE `t_deadlock`.`id` = 3 |                                                        |

* `main/parse_deadlock.go` 用于解析 `show engine innodb status;` 里的死锁信息（可视化一些）



## 友情链接

https://github.com/aneasystone/mysql-deadlocks