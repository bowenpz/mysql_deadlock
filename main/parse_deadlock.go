package main

import (
	"fmt"
	"regexp"
)

var (
	regexpTrxID, _     = regexp.Compile(`TRANSACTION (\d+),`)
	regexpSQL, _       = regexp.Compile(`TRANSACTION:(?U)[\w\W]*\n(.*)\n\*\*\*`)
	regexpIndex, _     = regexp.Compile(`WAITING FOR THIS LOCK TO BE GRANTED:\n.*index (.*) of table`)
	regexpTable, _     = regexp.Compile(`WAITING FOR THIS LOCK TO BE GRANTED:\n.*of table (.*) trx`)
	regexpWaitLock, _  = regexp.Compile(`WAITING FOR THIS LOCK TO BE GRANTED:\n.*trx id \d+ (.*)`)
	regexpHoldLock1, _ = regexp.Compile(`\(1\) HOLDS THE LOCK\(S\):\n.*trx id \d+ (.*)`)
	regexpHoldLock2, _ = regexp.Compile(`\(2\) HOLDS THE LOCK\(S\):\n.*trx id \d+ (.*)`)
)

type TrxWithLock struct {
	TrxID    string `json:"-"`
	SQL      string `json:"sql"`
	Lock     string `json:"wait lock"`
	HoldLock string `json:"hold lock,omitempty"`
}

func (t TrxWithLock) print() {
	title := fmt.Sprintf("TRANSACTIONS  %s", t.TrxID)
	line := ""
	for i := 0; i < len(title); i++ {
		line += "-"
	}

	fmt.Printf("\n%s\n%s\n%s\n", line, title, line)
	fmt.Printf("sql:       %s\n", t.SQL)
	fmt.Printf("wait lock: %s\n", t.Lock)
	if t.HoldLock != "" {
		fmt.Printf("hold lock: %s\n", t.HoldLock)
	}
}

// PrintLatestDeadlock 打印出最近的死锁
func PrintLatestDeadlock() {
	innodbInfo := struct {
		Status string
	}{}
	db.Raw("show engine innodb status").Scan(&innodbInfo)
	ParseDeadlock(innodbInfo.Status)
}

// ParseDeadlock 解析死锁信息
func ParseDeadlock(info string) {
	info = deleteExtraSpace(info)

	// print table & index
	parseTableInfo(info)

	// parse deadlock info
	t1, t2 := new(TrxWithLock), new(TrxWithLock)

	trxIDs := regexpTrxID.FindAllStringSubmatch(info, 2)
	t1.TrxID = trxIDs[0][1]
	t2.TrxID = trxIDs[1][1]

	sqls := regexpSQL.FindAllStringSubmatch(info, 2)
	t1.SQL = sqls[0][1]
	t2.SQL = sqls[1][1]

	waitLocks := regexpWaitLock.FindAllStringSubmatch(info, 2)
	t1.Lock = waitLocks[0][1]
	t2.Lock = waitLocks[1][1]

	if holdLocks := regexpHoldLock1.FindAllStringSubmatch(info, 1); len(holdLocks) > 0 {
		t1.HoldLock = holdLocks[0][1]
	}

	if holdLocks := regexpHoldLock2.FindAllStringSubmatch(info, 1); len(holdLocks) > 0 {
		t2.HoldLock = holdLocks[0][1]
	}

	// print deadlock info
	t1.print()
	t2.print()
	fmt.Printf("\n\n")
}

func parseTableInfo(info string) {
	table := fmt.Sprintf("Table: %s", regexpTable.FindAllStringSubmatch(info, 1)[0][1])
	index := fmt.Sprintf("Index: %s", regexpIndex.FindAllStringSubmatch(info, 1)[0][1])
	line := ""

	maxLength := len(table)
	if maxLength < len(index) {
		maxLength = len(index)
	}

	for len(line) < maxLength {
		line += "="
	}
	for len(table) < maxLength {
		table += " "
	}
	for len(index) < maxLength {
		index += " "
	}

	fmt.Printf("\n╔=%s=╗\n║ %s ║\n║ %s ║\n╚=%s=╝\n", line, table, index, line)
}

func deleteExtraSpace(s string) string {
	reg, _ := regexp.Compile("\\s{2,}")
	tmpStr := make([]byte, len(s))
	copy(tmpStr, s)
	spcIndex := reg.FindStringIndex(string(tmpStr))
	for len(spcIndex) > 0 {
		tmpStr = append(tmpStr[:spcIndex[0]+1], tmpStr[spcIndex[1]:]...)
		spcIndex = reg.FindStringIndex(string(tmpStr))
	}
	return string(tmpStr)
}
