package main

import (
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"sync"
	"time"
)

var ErrRollback = errors.New("rollback transaction")

type ConcurrentTrx struct {
	RollbackFinally bool                        // 是否在返回前回滚
	IgnoreDeadlock  bool                        // 是否忽略死锁
	ExecSeq         []int                       // 数组中的正数代表事务下标，负数代表等待时间（秒）
	TrxSQLList      [][]func(tx *gorm.DB) error // 每个事务要执行哪些 SQL
}

func (c *ConcurrentTrx) AddSQL(trxIndex int, sql func(tx *gorm.DB) error) {
	if len(c.TrxSQLList) == 0 {
		c.TrxSQLList = make([][]func(tx *gorm.DB) error, 10)
	}
	c.ExecSeq = append(c.ExecSeq, trxIndex)
	c.TrxSQLList[trxIndex] = append(c.TrxSQLList[trxIndex], sql)
}

func (c *ConcurrentTrx) Wait(t time.Duration) {
	c.ExecSeq = append(c.ExecSeq, -int(t.Seconds()))
}

func (c *ConcurrentTrx) Rollback(trxIndex int) {
	c.AddSQL(trxIndex, func(tx *gorm.DB) error {
		return ErrRollback
	})
}

func (c *ConcurrentTrx) Execute() {
	var (
		wg                  = sync.WaitGroup{}
		trxExecWaitChanList []chan int
		trxExecDoneChan     = make(chan int, 1)
		errChan             = make(chan error, 1)
	)

	trxExecWaitChanList = make([]chan int, 0)
	for i := 0; i < len(c.TrxSQLList); i++ {
		trxExecWaitChanList = append(trxExecWaitChanList, make(chan int, 1))
	}

	// 遍历所有事务
	for trxIndex, sqlList := range c.TrxSQLList {
		if len(sqlList) != 0 {
			wg.Add(1)

			sqlList := sqlList
			trxIndex := trxIndex

			// 每个事务起一个 goroutine
			go func() {
				err := db.Transaction(func(tx *gorm.DB) error {
					for sqlIndex, sqlFunc := range sqlList {
						// 阻塞等待，直到收到信号才执行，以此控制多个事务之间的先后顺序
						<-trxExecWaitChanList[trxIndex]

						err := sqlFunc(tx)
						if err != nil {
							return err
						}
						fmt.Printf("Success to exec SQL, trx index: %d, sql index: %d\n", trxIndex, sqlIndex+1)
						trxExecDoneChan <- 1
					}

					if c.RollbackFinally {
						return ErrRollback
					}
					return nil
				})
				if err != nil && err != ErrRollback {
					if IsDeadlock(err) {
						fmt.Printf("\n\nA deadlock has occurred!")
						PrintLatestDeadlock()
						if c.IgnoreDeadlock {
							wg.Done()
							return
						}
					}
					errChan <- fmt.Errorf("failed to exec transaction, trx index: %d, err:\n%s", trxIndex, err)
					return
				}
				wg.Done()
			}()
		}
	}

	// 按顺序让多个事务执行 SQL，一个事务执行 SQL 成功之后（或者超时），再让下一个事务执行 SQL
	for _, seq := range c.ExecSeq {
		if seq > 0 {
			trxExecWaitChanList[seq] <- 1
			select {
			case <-trxExecDoneChan:
			case err := <-errChan:
				panic(err)
			case <-time.After(time.Second):
				fmt.Printf("\nwait sql time out, jump it.\n")
			}
		} else {
			fmt.Printf("\nwait %d seconds\n", -seq)
			for i := 0; i < -seq; i++ {
				time.Sleep(time.Second)
			}
		}
	}

	// wait for all transactions to complete
	waitChan := make(chan int)
	go func() {
		wg.Wait()
		waitChan <- 1
	}()
	select {
	case <-waitChan:
	case err := <-errChan:
		panic(err)
	case <-time.After(time.Second):
		fmt.Printf("\nwait trx execute time out, return.\n")
	}
}

func IsDeadlock(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1213
}
