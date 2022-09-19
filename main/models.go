package main

type Deadlock struct {
	ID  int `gorm:"primaryKey"`     // 主键
	Uk  int `gorm:"uniqueIndex:uk"` // 唯一索引
	Idx int `gorm:"index:idx"`      // 普通索引
}

func (Deadlock) TableName() string {
	return "t_deadlock"
}
