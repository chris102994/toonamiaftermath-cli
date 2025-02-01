package toonamiaftermath

import (
	"reflect"
	"strings"
	"time"
)

type TAChannel struct {
	Name string `json:"name"`
}

func (c *TAChannel) GetSlug() string {
	channelSlug := ""

	switch {
	case strings.Contains(c.Name, "Toonami Aftermath") && !strings.Contains(c.Name, "Radio"):
		channelSlug = "est"
	case strings.Contains(c.Name, "Snickelodeon"):
		channelSlug = "snick-est"
	case strings.Contains(c.Name, "Movies"):
		channelSlug = "movies"
	case strings.Contains(c.Name, "MTV97"):
		channelSlug = "mtv97"
	case strings.Contains(c.Name, "Radio"):
		channelSlug = "radio"
	}
	return channelSlug
}

func (c *TAChannel) GetWestOffset() bool {
	westOffset := false
	if strings.Contains(strings.ToLower(c.Name), "west") {
		westOffset = true
	}
	return westOffset
}

type Media struct {
	Name          string    `json:"name"`
	StartDate     string    `json:"startDate"`
	Info          MediaInfo `json:"info"`
	BlockName     string    `json:"blockName"`
	MediaType     string    `json:"mediaType"`
	EpisodeNumber int       `json:"episodeNumber"`
}

type MediaInfo struct {
	Fullname string `json:"fullname"`
	Image    string `json:"image"`
	Episode  string `json:"episode"`
	Year     int    `json:"year"`
}

type Episode struct {
	Image   string `json:"image"`
	Summary string `json:"summary"`
	AirDate string `json:"airDate"`
	EpNum   int    `json:"epNum"`
	Season  int    `json:"season"`
	Name    string `json:"name"`
}

type EpisodeInfo struct {
	Genres        []string  `json:"genres"`
	Creators      []string  `json:"creators"`
	ProductionCo  []string  `json:"productionCo"`
	ID            string    `json:"_id"`
	UpdatedAt     time.Time `json:"updatedAt"`
	CreatedAt     time.Time `json:"createdAt"`
	ImdbID        string    `json:"imdbId"`
	Name          string    `json:"name"`
	ContentRating string    `json:"contentRating"`
	Release       string    `json:"release"`
	Tagline       string    `json:"tagline"`
	Aka           string    `json:"aka"`
	Rating        float64   `json:"rating"`
	Image         string    `json:"image"`
	ReleaseDate   string    `json:"releaseDate"`
	Summary       string    `json:"summary"`
	Storyline     string    `json:"storyline"`
	Series        bool      `json:"series"`
	SearchedName  string    `json:"searchedName"`
	Version       int       `json:"__v"`
	Episode       Episode   `json:"episode"`
}

func IsEpisodeInfoEmpty(ei EpisodeInfo) bool {
	return reflect.DeepEqual(ei, EpisodeInfo{})
}
