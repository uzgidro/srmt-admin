package data_convert

import (
	"fmt"
	"srmt-admin/internal/lib/model/data"
)

func Convert(resId int64, indicatorLevel, current, resistance float64) (data.Model, error) {

	var model data.Model
	var err error

	switch resId {
	case 1:
		model, err = ConvertAndijan(indicatorLevel, current, resistance)
	default:
		return data.Model{}, fmt.Errorf("unsupported reservoir id for data conversion: %d", resId)
	}
	if err != nil {
		return data.Model{}, fmt.Errorf("failed to convert data for reservoir %d: %w", resId, err)
	}

	model.ResID = resId

	return model, nil
}
