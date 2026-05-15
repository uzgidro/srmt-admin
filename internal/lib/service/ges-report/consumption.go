package gesreportservice

import (
	"fmt"
	"strings"

	model "srmt-admin/internal/lib/model/ges-report"
)

// ReportValidationErrorCode is the stable machine-readable code returned in
// the error envelope for the report-side consumption-vs-idle violation.
const ReportValidationErrorCode = "report.consumption_exceeds_idle"

// ConsumptionViolation is one station's contribution to a
// ReportValidationError. The handler renders these as the `details` array of
// the structured 400 response so the frontend can list every offending
// station with its name, computed idle, and configured consumption.
type ConsumptionViolation struct {
	OrganizationID   int64
	OrganizationName string
	Date             string
	IdleM3s          float64 // total_outflow_m3s - ges_flow_m3s, before subtracting consumption
	ConsumptionM3s   float64
}

// ReportValidationError is returned by BuildDailyReport when one or more
// stations have consumption_m3_s greater than the computed idle discharge
// (total_outflow - ges_flow). All violations across the report are collected;
// the handler maps Code/Violations to the structured 400 wire format.
type ReportValidationError struct {
	Code       string
	Violations []ConsumptionViolation
}

func (e *ReportValidationError) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "report validation failed"
	}
	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, fmt.Sprintf(
			"organization_id=%d (%s): consumption=%g > idle=%g on %s",
			v.OrganizationID, v.OrganizationName, v.ConsumptionM3s, v.IdleM3s, v.Date,
		))
	}
	return "useful consumption exceeds idle discharge for: " + strings.Join(parts, "; ")
}

// validateConsumptionAgainstIdle scans today's rows and returns a
// *ReportValidationError when one or more stations have consumption_m3_s
// greater than the computable idle (total_outflow - ges_flow). Stations
// without a complete idle (any of outflow/ges_flow nil) are skipped — the
// check is inapplicable, not a violation. Returns nil when every row passes
// (or has nothing to check).
func validateConsumptionAgainstIdle(rows []model.RawDailyRow) *ReportValidationError {
	var violations []ConsumptionViolation
	for _, row := range rows {
		if row.ConsumptionM3s == nil {
			continue
		}
		if row.TotalOutflowM3s == nil || row.GESFlowM3s == nil {
			continue
		}
		idle := *row.TotalOutflowM3s - *row.GESFlowM3s
		cons := *row.ConsumptionM3s
		if cons > idle {
			// Round to match what computeDaySnapshot would have rendered, so
			// the figure shown in the structured 400 matches the figure the
			// frontend would otherwise have seen in current.idle_discharge_m3s.
			violations = append(violations, ConsumptionViolation{
				OrganizationID:   row.OrganizationID,
				OrganizationName: row.OrganizationName,
				Date:             row.Date,
				IdleM3s:          roundTo2(idle),
				ConsumptionM3s:   cons,
			})
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return &ReportValidationError{
		Code:       ReportValidationErrorCode,
		Violations: violations,
	}
}
