package reservoirhourly

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"srmt-admin/internal/lib/dto"
	model "srmt-admin/internal/lib/model/reservoir-hourly"
	"time"
)

// OrgNameResolver resolves organization names by IDs
type OrgNameResolver interface {
	GetOrganizationNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
}

// DataFetcher fetches raw reservoir data
type DataFetcher interface {
	FetchLast12(ctx context.Context, date string) (map[int64][]*dto.ReservoirData, error)
	GetIDs() []int64
}

// Service transforms raw fetcher data into HourlyReport
type Service struct {
	fetcher  DataFetcher
	resolver OrgNameResolver
	log      *slog.Logger
}

// NewService creates a new reservoir-hourly service
func NewService(fetcher DataFetcher, resolver OrgNameResolver, log *slog.Logger) *Service {
	return &Service{
		fetcher:  fetcher,
		resolver: resolver,
		log:      log,
	}
}

// BuildReport fetches data and builds the hourly report
func (s *Service) BuildReport(ctx context.Context, date string) (*model.HourlyReport, error) {
	const op = "service.reservoir-hourly.BuildReport"

	// 1. Fetch last 12 records per reservoir
	rawData, err := s.fetcher.FetchLast12(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch: %w", op, err)
	}

	// 2. Resolve org names
	orgIDs := s.fetcher.GetIDs()
	nameMap, err := s.resolver.GetOrganizationNamesByIDs(ctx, orgIDs)
	if err != nil {
		s.log.Error("failed to resolve org names", slog.String("op", op), slog.Any("error", err))
		nameMap = make(map[int64]string)
	}

	// 3. Transform each reservoir's data
	var reservoirs []model.ReservoirData
	var globalLatest time.Time
	var periodSource []*dto.ReservoirData // records from the reservoir with latest timestamp

	for _, orgID := range orgIDs {
		records := rawData[orgID]
		if len(records) == 0 {
			continue
		}

		// Records from API are oldest-first; reverse so index 0 = newest
		reversed := make([]*dto.ReservoirData, len(records))
		for i, r := range records {
			reversed[len(records)-1-i] = r
		}

		rd := s.transformRecords(orgID, nameMap[orgID], reversed)
		reservoirs = append(reservoirs, rd)

		// Track which reservoir has the most recent timestamp (for period calc)
		if reversed[0].Time != nil && reversed[0].Time.After(globalLatest) {
			globalLatest = *reversed[0].Time
			periodSource = reversed
		}
	}

	// 4. Compute period from the reservoir with the most recent timestamp
	period := s.computePeriod(periodSource)

	return &model.HourlyReport{
		Date:       date,
		LatestTime: globalLatest,
		Period:     int(period.Hours()),
		Reservoirs: reservoirs,
	}, nil
}

// transformRecords converts reversed records (index 0 = newest) into ReservoirData
func (s *Service) transformRecords(orgID int64, orgName string, records []*dto.ReservoirData) model.ReservoirData {
	current := records[0]

	// Find day-begin: first record starting from index 1 where hour == 6
	dayBegin := records[len(records)-1] // fallback to oldest
	for i := 1; i < len(records); i++ {
		if records[i].Time != nil && records[i].Time.Hour() == 6 {
			dayBegin = records[i]
			break
		}
	}

	rd := model.ReservoirData{
		OrganizationID:   orgID,
		OrganizationName: orgName,
	}

	// Level
	if current.Level != nil {
		rd.Level.Current = *current.Level
	}
	if dayBegin.Level != nil {
		rd.Level.DayBegin = *dayBegin.Level
	}

	// Volume
	if current.Volume != nil {
		rd.Volume.Current = *current.Volume
	}
	if dayBegin.Volume != nil {
		rd.Volume.DayBegin = *dayBegin.Volume
	}

	// Weather
	if current.Weather != nil {
		rd.Weather.Current = *current.Weather
	}
	if dayBegin.Weather != nil {
		rd.Weather.DayBegin = *dayBegin.Weather
	}

	// Release (latest)
	if current.Release != nil {
		rd.Release = *current.Release
	}

	// Income at day begin
	if dayBegin.Income != nil {
		rd.IncomeAtDayBegin = *dayBegin.Income
	}

	// Income: last 6 values in chronological order (oldest -> newest)
	// records[5..0] reversed = records[0..5] but we want chronological, so records[5],records[4],...,records[0]
	incomeCount := 6
	if len(records) < incomeCount {
		incomeCount = len(records)
	}
	rd.Income = make([]float64, incomeCount)
	for i := 0; i < incomeCount; i++ {
		// chronological: oldest first → records[incomeCount-1-i] maps to rd.Income[i... wait
		// records[0] = newest, records[5] = oldest (of the 6)
		// we want chronological: oldest→newest = records[incomeCount-1], records[incomeCount-2], ..., records[0]
		idx := incomeCount - 1 - i
		if records[idx].Income != nil {
			rd.Income[i] = *records[idx].Income
		}
	}

	return rd
}

// computePeriod calculates the observation interval from the last 4 records
func (s *Service) computePeriod(records []*dto.ReservoirData) time.Duration {
	if len(records) < 4 {
		return 0
	}

	// Take last 4 records (records[0..3], already newest-first)
	// Compute 3 time diffs, return the minimum
	minDiff := time.Duration(math.MaxInt64)
	for i := 0; i < 3; i++ {
		if records[i].Time == nil || records[i+1].Time == nil {
			continue
		}
		diff := records[i].Time.Sub(*records[i+1].Time)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
		}
	}

	if minDiff == time.Duration(math.MaxInt64) {
		return 0
	}
	return minDiff
}
