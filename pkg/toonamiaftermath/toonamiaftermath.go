package toonamiaftermath

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	m3u "github.com/chris102994/go-m3u/pkg/m3u/models"
	xmltv "github.com/chris102994/go-xmltv/pkg/xmltv/models"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// TODO:
// - Cleanup the code and methods
// - Make logging concise
// - Test the code

type ToonamiAftermath struct {
	M3UOutput   m3u.M3U
	XMLTVOutput xmltv.TV

	transport *http.Transport
	client    *http.Client

	EpisodeCache EpisodeCache
}

func New() *ToonamiAftermath {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	episodeCache := EpisodeCache{
		Episodes: make(map[string]EpisodeInfo),
	}

	this := &ToonamiAftermath{
		M3UOutput: m3u.M3U{
			PlaylistHeaders: make([]m3u.PlaylistHeader, 0),
			Channels:        make([]m3u.Channel, 0),
		},
		XMLTVOutput: xmltv.TV{
			Date:              time.Now().Format("2006-01-02"),
			SourceInfoURL:     "https://api.toonamiaftermath.com",
			SourceInfoName:    "Toonami Aftermath",
			SourceDataURL:     "https://api.toonamiaftermath.com",
			GeneratorInfoName: "Toonami Aftermath CLI",
			GeneratorInfoURL:  "https://github.com/chris102994/toonamiaftermath-cli",
			Channels:          make([]*xmltv.Channel, 0),
			Programmes:        make([]*xmltv.Programme, 0),
		},
		transport:    transport,
		client:       client,
		EpisodeCache: episodeCache,
	}

	this.M3UOutput.PlaylistHeaders = append(this.M3UOutput.PlaylistHeaders, m3u.PlaylistHeader{})

	return this
}

func (t *ToonamiAftermath) Run() error {
	log.WithFields(log.Fields{
		"m3u":   t.M3UOutput,
		"xmltv": t.XMLTVOutput,
	}).Info("Scraping With...")

	baseUrl := "https://api.toonamiaftermath.com"

	// Get Channels
	channelsParams := url.Values{
		"startDate": {time.Now().Format("2006-01-02T15:04:05Z")},
	}
	channelsUrl := baseUrl + "/channelsCurrentMedia" + "?" + channelsParams.Encode()

	taChannels, err := t.GetTAChannels(channelsUrl)
	if err != nil {
		return fmt.Errorf("failed to get Toonami Aftermath channels: %w", err)
	}
	log.WithFields(log.Fields{
		"channelsUrl": channelsUrl,
		"taChannels":  taChannels,
	}).Trace("Channels")

	for index, taChannel := range taChannels {
		westOffset := taChannel.GetWestOffset()
		channelSlug := taChannel.GetSlug()
		if channelSlug == "" {
			log.WithFields(log.Fields{
				"channelName": taChannel.Name,
			}).Error("Unknown Channel. Skipping.")
			continue
		}

		m3uUrlValues := url.Values{
			"channelName":    {channelSlug},
			"timezoneOffset": {"5"},
			"useHttps":       {"true"},
		}

		if westOffset {
			m3uUrlValues["streamDelay"] = []string{"180"}
		}

		m3uUrl := baseUrl + "/streamUrl" + "?" + m3uUrlValues.Encode()

		m3uUrlResp, err := t.client.Get(m3uUrl)
		if err != nil {
			return err
		}

		m3uUrlBytes, err := io.ReadAll(m3uUrlResp.Body)
		if err != nil {
			return err
		}

		m3uUrl = string(m3uUrlBytes)

		channelId := fmt.Sprintf("%v", index+1)

		t.M3UOutput.Channels = append(t.M3UOutput.Channels, m3u.Channel{
			Duration: -0,
			Metadata: m3u.ChannelMetadata{
				ChannelID:   channelId,
				GroupTitle:  "Toonami Aftermath",
				TvgCountry:  "us",
				TvgID:       taChannel.Name,
				TvgLanguage: "en",
				TvgName:     taChannel.Name,
			},
			Title: taChannel.Name,
			URL:   m3uUrl,
		})

		t.XMLTVOutput.Channels = append(t.XMLTVOutput.Channels, &xmltv.Channel{
			ID: fmt.Sprintf("%v", index+1),
			DisplayNames: append(make([]*xmltv.DisplayName, 0), &xmltv.DisplayName{
				Lang: "en",
				Text: taChannel.Name,
			}),
			URLs: append(make([]*xmltv.URL, 0), &xmltv.URL{
				Text: channelsUrl,
			}),
		})

		log.WithFields(log.Fields{
			"channelName": taChannel.Name,
			"westOffset":  westOffset,
			"channelSlug": channelSlug,
			"m3uUrl":      m3uUrl,
		}).Info("Channel")

		scheduleNameString := strings.ReplaceAll(taChannel.Name, "East", "EST")
		scheduleNameString = strings.ReplaceAll(scheduleNameString, "West", "EST")

		guideParams := url.Values{
			"scheduleName": {scheduleNameString},
			"dateString":   {time.Now().Format("2006-01-02T15:04:05Z")},
			"count":        {"200"},
		}
		guideUrl := baseUrl + "/media" + "?" + guideParams.Encode()

		guideResp, err := t.client.Get(guideUrl)
		if err != nil {
			return err
		}

		guideBytes, err := io.ReadAll(guideResp.Body)
		if err != nil {
			return err
		}

		if len(guideBytes) == 0 {
			log.WithFields(log.Fields{
				"channelName": taChannel.Name,
			}).Warn("No Guide Data. Skipping. . . ")
			continue
		}

		log.WithFields(log.Fields{
			"guideUrl":         guideUrl,
			"guideResp.Status": guideResp.Status,
		}).Info("Guide query")

		var taMedia []Media
		err = json.Unmarshal(guideBytes, &taMedia)
		if err != nil {
			return err
		}

		for _, mediaItem := range taMedia {
			log.WithFields(log.Fields{
				"mediaItem": mediaItem,
			}).Trace("Media Item")

			parsedTime, err := time.Parse(time.RFC3339, mediaItem.StartDate)
			if err != nil {
				log.Fatal(err)
			}
			startTime := parsedTime.Format("20060102150405")
			if westOffset {
				startTime = fmt.Sprintf("%v %v", startTime, "-0300")
			} else {
				startTime = fmt.Sprintf("%v %v", startTime, "+0000")
			}

			episodeUrlValues := url.Values{}

			if mediaItem.Info.Fullname != "" {
				episodeUrlValues["name"] = []string{mediaItem.Info.Fullname}
			} else if mediaItem.Name != "" {
				episodeUrlValues["name"] = []string{mediaItem.Name}
			} else if mediaItem.BlockName != "" {
				episodeUrlValues["name"] = []string{mediaItem.BlockName}
			}

			if mediaItem.Info.Year != 0 {
				episodeUrlValues["year"] = []string{fmt.Sprintf("%v", mediaItem.Info.Year)}
			}

			if mediaItem.EpisodeNumber != 0 {
				episodeUrlValues["episode"] = []string{fmt.Sprintf("%v", mediaItem.EpisodeNumber)}
			}

			programme := xmltv.Programme{
				Channel: channelId,
				Start:   startTime,
			}

			if mediaItem.Name != "" {
				programme.Title = []*xmltv.Title{
					{
						Lang: "en",
						Text: mediaItem.Name,
					},
				}
			} else if mediaItem.Info.Fullname != "" {
				programme.Title = []*xmltv.Title{
					{
						Lang: "en",
						Text: mediaItem.Info.Fullname,
					},
				}
			} else if mediaItem.BlockName != "" {
				programme.Title = []*xmltv.Title{
					{
						Lang: "en",
						Text: mediaItem.BlockName,
					},
				}
			} else {
				log.WithFields(log.Fields{
					"mediaItem": mediaItem,
				}).Trace("No Title")
			}

			if len(episodeUrlValues) != 0 {
				episodeUrl := baseUrl + "/mediaInfo" + "?" + episodeUrlValues.Encode()

				episodeInfo, err := t.getEpisodeInfo(episodeUrl)
				if err != nil {
					return fmt.Errorf("failed to get episode info: %w", err)
				}

				if !IsEpisodeInfoEmpty(episodeInfo) {
					programme.Title = []*xmltv.Title{
						{
							Lang: "en",
							Text: episodeInfo.Name,
						},
					}

					programme.SubTitle = []*xmltv.SubTitle{
						{
							Lang: "en",
							Text: episodeInfo.Episode.Name,
						},
					}

					programme.Image = []*xmltv.Image{
						{
							Text: episodeInfo.Image,
						},
					}

					programme.Icons = []*xmltv.Icon{
						{
							Src: episodeInfo.Image,
						},
					}

					programme.Desc = []*xmltv.Desc{
						{
							Lang: "en",
							Text: episodeInfo.Episode.Summary,
						},
					}

					if episodeInfo.Episode.AirDate != "" {
						var parsedDate time.Time
						formats := []string{"2 Jan 2006", "2 Jan. 2006", "2006", "Jan. 2006"}
						for _, format := range formats {
							parsedDate, err = time.Parse(format, episodeInfo.Episode.AirDate)
							if err == nil {
								break
							}
						}

						if err != nil {
							log.Fatal(err)
						}
						programme.Date = parsedDate.Format("20060102")
					}

					unusableRatings := []string{"", "Not Rated"}
					if !slices.Contains(unusableRatings, episodeInfo.ContentRating) {
						programme.Rating = []*xmltv.Rating{
							{
								System: "VCHIP",
								Value:  episodeInfo.ContentRating,
							},
						}
					}

					if episodeInfo.Rating != 0 {
						programme.StarRating = []*xmltv.StarRating{
							{
								System: "imdb",
								Value:  fmt.Sprintf("%v/10", episodeInfo.Rating),
							},
						}
					}

					if episodeInfo.Episode.Season != 0 && episodeInfo.Episode.EpNum != 0 {
						programme.EpisodeNum = []*xmltv.EpisodeNum{
							{
								System: "xmltv_ns",
								Text:   fmt.Sprintf("%v.%v.0/1", episodeInfo.Episode.Season-1, episodeInfo.Episode.EpNum-1),
							},
						}
					}

					programme.Category = []*xmltv.Category{}
					for _, category := range episodeInfo.Genres {
						programme.Category = append(programme.Category, &xmltv.Category{
							Lang: "en",
							Text: category,
						})
					}

					unwantedCredits := []string{
						"a.k.a. Cartoon",
						"IMDbPro",
						"See full cast & crew",
						"See more",
					}
					writers := slices.DeleteFunc(episodeInfo.Creators, func(s string) bool {
						return slices.Contains(unwantedCredits, s)
					})
					producers := slices.DeleteFunc(episodeInfo.ProductionCo, func(s string) bool {
						return slices.Contains(unwantedCredits, s)
					})
					programme.Credits = &xmltv.Credits{
						Writers:   writers,
						Producers: producers,
					}
				} else {
					log.WithFields(log.Fields{
						"episodeUrl": episodeUrl,
					}).Trace("No Episode Info")
				}

			}

			log.WithFields(log.Fields{
				"mediaItem": mediaItem,
			}).Trace("Adding Media Item")

			t.XMLTVOutput.Programmes = append(t.XMLTVOutput.Programmes, &programme)
		}
	}

	for position, programme := range t.XMLTVOutput.Programmes {
		if position < len(t.XMLTVOutput.Programmes)-1 {
			if programme.Channel == t.XMLTVOutput.Programmes[position+1].Channel {
				programme.Stop = t.XMLTVOutput.Programmes[position+1].Start
			} else {
				log.WithFields(log.Fields{
					"position":  position,
					"programme": programme,
				}).Trace("Programme next channel. No end time.")
			}
		} else {
			log.WithFields(log.Fields{
				"position":  position,
				"programme": programme,
			}).Trace("Programme last index. No end time.")
		}
	}

	return nil
}

func (t *ToonamiAftermath) GetTAChannels(channelsUrl string) ([]TAChannel, error) {
	resp, err := t.client.Get(channelsUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var taChannels []TAChannel
	err = json.Unmarshal(bodyBytes, &taChannels)
	if err != nil {
		return nil, err
	}

	return taChannels, nil
}

func (t *ToonamiAftermath) getEpisodeInfo(episodeInfoUrl string) (EpisodeInfo, error) {
	thisEpisodeInfo := EpisodeInfo{}

	if episodeInfo, ok := t.EpisodeCache.Episodes[episodeInfoUrl]; ok {
		log.WithFields(log.Fields{
			"episodeInfoUrl": episodeInfoUrl,
		}).Trace("Using Cached Episode Info")
		return episodeInfo, nil
	} else {
		episodeResp, err := t.client.Get(episodeInfoUrl)
		if err != nil {
			return thisEpisodeInfo, fmt.Errorf("failed to get episode response: %w", err)
		}

		episodeBytes, err := io.ReadAll(episodeResp.Body)
		if err != nil {
			return thisEpisodeInfo, fmt.Errorf("failed to read episode response body: %w", err)
		}

		log.WithFields(log.Fields{
			"episodeUrl":         episodeInfoUrl,
			"episodeResp.Status": episodeResp.Status,
		}).Trace("Episode query")

		if len(episodeBytes) > 0 {
			err = json.Unmarshal(episodeBytes, &thisEpisodeInfo)
			if err != nil {
				return thisEpisodeInfo, fmt.Errorf("failed to unmarshal episode response: %w", err)
			}
		} else {
			log.WithFields(log.Fields{
				"episodeUrl": episodeInfoUrl,
			}).Warn("No Episode Data was returned.")
		}
	}

	if !IsEpisodeInfoEmpty(thisEpisodeInfo) {
		log.WithFields(log.Fields{
			"episodeInfoUrl": episodeInfoUrl,
		}).Trace("Caching Episode Info")
		t.EpisodeCache.Episodes[episodeInfoUrl] = thisEpisodeInfo
	}

	return thisEpisodeInfo, nil
}
