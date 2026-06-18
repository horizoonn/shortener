package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/horizoonn/shortener/internal/analytics"
	"github.com/horizoonn/shortener/internal/links"
)

type CreateLinkRequest struct {
	OriginalURL string  `json:"original_url"`
	CustomAlias *string `json:"custom_alias"`
}

type CreateLinkResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	IsCustom    bool      `json:"is_custom"`
	CreatedAt   time.Time `json:"created_at"`
}

func createLinkResponseFromDomain(link links.Link, shortURL string) CreateLinkResponse {
	return CreateLinkResponse{
		ID:          link.ID,
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
		ShortURL:    shortURL,
		IsCustom:    link.IsCustom,
		CreatedAt:   link.CreatedAt,
	}
}

type AnalyticsResponse struct {
	Code              string                    `json:"code"`
	OriginalURL       string                    `json:"original_url"`
	TotalClicks       int64                     `json:"total_clicks"`
	ClicksByDay       []TimeBucketCountResponse `json:"clicks_by_day"`
	ClicksByMonth     []TimeBucketCountResponse `json:"clicks_by_month"`
	ClicksByUserAgent []UserAgentCountResponse  `json:"clicks_by_user_agent"`
	RecentClicks      []RecentClickResponse     `json:"recent_clicks"`
}

type TimeBucketCountResponse struct {
	Day    string `json:"day,omitempty"`
	Month  string `json:"month,omitempty"`
	Clicks int64  `json:"clicks"`
}

type UserAgentCountResponse struct {
	UserAgent string `json:"user_agent"`
	Clicks    int64  `json:"clicks"`
}

type RecentClickResponse struct {
	ClickedAt time.Time `json:"clicked_at"`
	UserAgent string    `json:"user_agent"`
	Referer   *string   `json:"referer"`
	IP        *string   `json:"ip"`
}

func analyticsResponseFromDomain(link links.Link, linkAnalytics analytics.LinkAnalytics) AnalyticsResponse {
	return AnalyticsResponse{
		Code:              link.Code,
		OriginalURL:       link.OriginalURL,
		TotalClicks:       linkAnalytics.TotalClicks,
		ClicksByDay:       dayCountsResponseFromDomain(linkAnalytics.ClicksByDay),
		ClicksByMonth:     monthCountsResponseFromDomain(linkAnalytics.ClicksByMonth),
		ClicksByUserAgent: userAgentCountsResponseFromDomain(linkAnalytics.ClicksByUserAgent),
		RecentClicks:      recentClicksResponseFromDomain(linkAnalytics.RecentClicks),
	}
}

func dayCountsResponseFromDomain(counts []analytics.TimeBucketCount) []TimeBucketCountResponse {
	response := make([]TimeBucketCountResponse, 0, len(counts))
	for _, count := range counts {
		response = append(response, TimeBucketCountResponse{
			Day:    count.Bucket.Format("2006-01-02"),
			Clicks: count.Count,
		})
	}

	return response
}

func monthCountsResponseFromDomain(counts []analytics.TimeBucketCount) []TimeBucketCountResponse {
	response := make([]TimeBucketCountResponse, 0, len(counts))
	for _, count := range counts {
		response = append(response, TimeBucketCountResponse{
			Month:  count.Bucket.Format("2006-01"),
			Clicks: count.Count,
		})
	}

	return response
}

func userAgentCountsResponseFromDomain(counts []analytics.UserAgentCount) []UserAgentCountResponse {
	response := make([]UserAgentCountResponse, 0, len(counts))
	for _, count := range counts {
		response = append(response, UserAgentCountResponse{
			UserAgent: count.UserAgent,
			Clicks:    count.Count,
		})
	}

	return response
}

func recentClicksResponseFromDomain(clicks []analytics.Click) []RecentClickResponse {
	response := make([]RecentClickResponse, 0, len(clicks))
	for _, click := range clicks {
		response = append(response, RecentClickResponse{
			ClickedAt: click.ClickedAt,
			UserAgent: click.UserAgent,
			Referer:   click.Referer,
			IP:        click.IP,
		})
	}

	return response
}
