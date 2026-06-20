package usagelogs

import "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"

func (s *Service) PublicChartData(apiKey string, days int) (map[string]any, error) {
	win := usage.WindowFromDays(days)
	daily, err := usage.QueryDailySeries(apiKey, win)
	if err != nil {
		return nil, err
	}
	if daily == nil {
		daily = []usage.DailySeriesPoint{}
	}

	models, err := usage.QueryModelDistribution(apiKey, win)
	if err != nil {
		return nil, err
	}
	if models == nil {
		models = []usage.ModelDistributionPoint{}
	}

	stats, err := usage.QueryStats(usage.LogQueryParams{APIKey: apiKey, Days: days})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"daily_series":       daily,
		"model_distribution": models,
		"stats":              stats,
	}, nil
}

func (s *Service) UsageChartData(apiKey string, win usage.TimeWindow) (map[string]any, error) {
	daily, err := usage.QueryDailySeries(apiKey, win)
	if err != nil {
		return nil, err
	}
	if daily == nil {
		daily = []usage.DailySeriesPoint{}
	}

	models, err := usage.QueryModelDistribution(apiKey, win)
	if err != nil {
		return nil, err
	}
	if models == nil {
		models = []usage.ModelDistributionPoint{}
	}

	// Hourly is a fixed "last 24h" real-time window unrelated to a historical
	// range. For custom ranges (explicit End) skip it and return empty series so
	// the frontend hides the hourly charts.
	hourlyTokens := []usage.HourlyTokenPoint{}
	hourlyModels := []usage.HourlyModelPoint{}
	if win.End.IsZero() {
		hourlyTokens, hourlyModels, err = usage.QueryHourlySeries(apiKey, 24)
		if err != nil {
			return nil, err
		}
		if hourlyTokens == nil {
			hourlyTokens = []usage.HourlyTokenPoint{}
		}
		if hourlyModels == nil {
			hourlyModels = []usage.HourlyModelPoint{}
		}
	}

	var apikeyDist []usage.APIKeyDistributionPoint
	if apiKey == "" {
		apikeyDist, err = usage.QueryAPIKeyDistribution(win)
		if err != nil {
			return nil, err
		}
		keyNameMap, _, _, _ := s.buildNameMaps()
		for i := range apikeyDist {
			if apikeyDist[i].Name == "" {
				if name, ok := keyNameMap[apikeyDist[i].APIKey]; ok {
					apikeyDist[i].Name = name
				}
			}
		}
	}
	if apikeyDist == nil {
		apikeyDist = []usage.APIKeyDistributionPoint{}
	}

	// Latency (TTFB) and throughput (tokens/sec) follow the selected window,
	// unlike the real-time hourly series, so they are queried for both preset
	// and custom ranges.
	latencyThroughput, err := usage.QueryLatencyThroughput(apiKey, win)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"daily_series":        daily,
		"model_distribution":  models,
		"hourly_tokens":       hourlyTokens,
		"hourly_models":       hourlyModels,
		"apikey_distribution": apikeyDist,
		"latency_throughput":  latencyThroughput,
	}, nil
}

func (s *Service) EntityUsageStats(apiKey string, days int, authIndexes, sources []string) (map[string]any, error) {
	sourceStats, err := usage.QueryEntityStats(apiKey, days, "source", sources)
	if err != nil {
		return nil, err
	}
	if sourceStats == nil {
		sourceStats = []usage.EntityStatPoint{}
	}

	authIndexStats, err := usage.QueryEntityStats(apiKey, days, "auth_index", authIndexes)
	if err != nil {
		return nil, err
	}
	if authIndexStats == nil {
		authIndexStats = []usage.EntityStatPoint{}
	}

	return map[string]any{
		"source":     sourceStats,
		"auth_index": authIndexStats,
	}, nil
}
