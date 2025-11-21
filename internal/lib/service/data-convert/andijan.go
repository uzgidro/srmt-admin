package data_convert

import (
	"math"
	"srmt-admin/internal/lib/model/data"
)

func ConvertAndijan(indicatorLevel, current, resistance float64) (data.Model, error) {

	virtualHeight := ((current - 4) / 16) * 60
	rawHeight := indicatorLevel - 9.3855 + virtualHeight
	height := math.Ceil(rawHeight*100) / 100

	temperature := (resistance - 100) / 0.385

	return data.Model{Level: height, Temperature: temperature}, nil
}
