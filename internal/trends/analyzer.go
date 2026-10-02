package trends

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/marcus/nightshift/internal/db"
)

const (
	defaultLookbackDays = 14
	maxLookbackDays     = 30
)

// Profile captures hourly usage averages for a provider.
type Profile struct {
	Provider       string
	LookbackDays   int
	HourlyAverages map[int]float64
	DailyTotal     float64
}

// Analyzer builds usage profiles and predicts near-term usage.
type Analyzer struct {
	db           *db.DB
	lookbackDays int
	nowFunc      func() time.Time
}

// NewAnalyzer constructs a trend analyzer with a bounded lookback.
func NewAnalyzer(database *db.DB, lookbackDays int) *Analyzer {
	if lookbackDays <= 0 {
		lookbackDays = defaultLookbackDays
	}
	if lookbackDays > maxLookbackDays {
		lookbackDays = maxLookbackDays
	}
	return &Analyzer{
		db:           database,
		lookbackDays: lookbackDays,
		nowFunc:      time.Now,
	}
}

// BuildProfile aggregates hourly averages over the lookback window.
func (a *Analyzer) BuildProfile(provider string, lookbackDays int) (Profile, error) {
	if a == nil || a.db == nil {
		return Profile{}, errors.New("trend analyzer not initialized")
	}

	if lookbackDays <= 0 {
		lookbackDays = a.lookbackDays
	}

	averages, err := a.getHourlyAverages(provider, lookbackDays)
	if err != nil {
		return Profile{}, err
	}

	hourly := make(map[int]float64, len(averages))
	dailyTotal := 0.0
	for _, avg := range averages {
		hourly[avg.hour] = avg.avg
		if avg.avg > dailyTotal {
			dailyTotal = avg.avg
		}
	}

	return Profile{
		Provider:       provider,
		LookbackDays:   lookbackDays,
		HourlyAverages: hourly,
		DailyTotal:     dailyTotal,
	}, nil
}

type hourlyAverage struct {
	hour int
	avg  float64
}

func (a *Analyzer) getHourlyAverages(provider string, lookbackDays int) ([]hourlyAverage, error) {
	if a.nowFunc == nil {
		a.nowFunc = time.Now
	}
	cutoff := a.nowFunc().AddDate(0, 0, -lookbackDays)
	rows, err := a.db.SQL().Query(
		`SELECT hour_of_day, AVG(local_daily)
		 FROM snapshots
		 WHERE provider = ? AND timestamp >= ?
		 GROUP BY hour_of_day
		 ORDER BY hour_of_day ASC`,
		strings.ToLower(provider),
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query hourly averages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	values := make([]hourlyAverage, 0)
	for rows.Next() {
		var avg hourlyAverage
		if err := rows.Scan(&avg.hour, &avg.avg); err != nil {
			return nil, fmt.Errorf("scan hourly average: %w", err)
		}
		values = append(values, avg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly averages: %w", err)
	}
	return values, nil
}

// PredictDaytimeUsage estimates remaining usage for today.
func (a *Analyzer) PredictDaytimeUsage(provider string, now time.Time, weeklyBudget int64) (int64, error) {
	profile, err := a.BuildProfile(provider, a.lookbackDays)
	if err != nil {
		return 0, err
	}
	if len(profile.HourlyAverages) == 0 || profile.DailyTotal <= 0 {
		return 0, nil
	}

	currentHour := now.Hour()
	currentAvg := profile.HourlyAverages[currentHour]
	if currentAvg == 0 {
		for hour := currentHour; hour >= 0; hour-- {
			if value, ok := profile.HourlyAverages[hour]; ok {
				currentAvg = value
				break
			}
		}
	}

	remaining := profile.DailyTotal - currentAvg
	if remaining < 0 {
		remaining = 0
	}

	if weeklyBudget > 0 {
		dailyCap := float64(weeklyBudget) / 7
		if remaining > dailyCap {
			remaining = dailyCap
		}
	}

	return int64(math.Round(remaining)), nil
}
