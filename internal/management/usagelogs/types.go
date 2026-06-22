package usagelogs

import "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"

type ManagementLogQueryInput struct {
	Page            int
	Size            int
	Days            int
	APIKeys         []string
	Models          []string
	Statuses        []string
	Channels        []string
	MatchNoAPIKeys  bool
	MatchNoModels   bool
	MatchNoStatuses bool
	MatchNoChannels bool
}

type PublicLogQueryInput struct {
	APIKey string
	Model  string
	Status string
	Page   int
	Size   int
	Days   int
}

type LogContentResponse struct {
	Status      int
	ContentType string
	Headers     map[string]string
	Payload     any
	Text        string
}

type AuthFileGroupTrendResponse struct {
	Days        int                     `json:"days"`
	Group       string                  `json:"group"`
	Points      []usage.DailyCountPoint `json:"points"`
	QuotaPoints []usage.DailyQuotaPoint `json:"quota_points"`
}

type AuthFileTrendResponse struct {
	AuthIndex         string                      `json:"auth_index"`
	Days              int                         `json:"days"`
	Hours             int                         `json:"hours"`
	RequestTotal      int64                       `json:"request_total"`
	CycleRequestTotal int64                       `json:"cycle_request_total"`
	CycleCostTotal    float64                     `json:"cycle_cost_total"`
	WeeklyQuotaUsed   *float64                    `json:"weekly_quota_used_percent"`
	CycleKnown        bool                        `json:"cycle_known"`
	CycleStart        string                      `json:"cycle_start"`
	DailyUsage        []usage.DailyUsagePoint     `json:"daily_usage"`
	HourlyUsage       []usage.HourlyUsagePoint    `json:"hourly_usage"`
	QuotaSeries       []usage.QuotaSnapshotSeries `json:"quota_series"`
}

// AuthFileWindowCostItem asks for one account's request cost since each quota
// window's start. The portal supplies the start instant of every quota window
// (5-hour, weekly, …) shown on a card; the service sums request cost from that
// instant up to now so the UI can estimate a window's total budget from
// "cost so far ÷ utilisation".
type AuthFileWindowCostItem struct {
	AuthIndex string                     `json:"auth_index"`
	Windows   []AuthFileWindowCostWindow `json:"windows"`
}

type AuthFileWindowCostWindow struct {
	Key   string `json:"key"`
	Since string `json:"since"` // RFC3339, inclusive lower bound
}
