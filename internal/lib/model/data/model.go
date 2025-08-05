package data

import "time"

type Model struct {
	ID          int64
	Time        time.Time
	Level       float64
	Volume      float64
	Income      float64
	Release     float64
	Temperature float64
	ResID       int64
}
