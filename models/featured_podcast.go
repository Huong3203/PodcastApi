package models

import "time"

type FeaturedPodcast struct {
	ID         string    `gorm:"type:char(36);primaryKey" json:"id"`
	PodcastID  string    `gorm:"type:char(36);not null" json:"podcast_id"`
	AvgRating  float64   `json:"avg_rating"`
	TotalVotes int64     `json:"total_votes"`
	FeaturedAt time.Time `json:"featured_at"`

	Podcast Podcast `gorm:"foreignKey:PodcastID;references:ID" json:"podcast"`
}
