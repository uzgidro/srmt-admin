package data_convert

import "srmt-admin/internal/lib/model/data"

func Convert(resId int64, indicatorLevel, current, resistance float64) (data.Model, error) {

	var model data.Model
	var err error

	switch resId {
	case 1:
		model, err = ConvertAndijan(indicatorLevel, current, resistance)
	default:
		model = data.Model{}
	}
	if err != nil {
		return model, err
	}

	return model, nil
}
