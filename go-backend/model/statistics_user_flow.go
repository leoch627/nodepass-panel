package model

type StatisticsUserFlow struct {
	ID         int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId     int64 `gorm:"column:user_id;index" json:"userId"`
	GostFlow   int64 `gorm:"column:gost_flow" json:"gostFlow"`
	XrayFlow   int64 `gorm:"column:xray_flow" json:"vFlow"`
	RecordTime int64 `gorm:"column:record_time;index" json:"recordTime"`
}

func (StatisticsUserFlow) TableName() string {
	return "statistics_user_flow"
}
