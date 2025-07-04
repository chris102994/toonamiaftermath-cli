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
// - Test the code

type ToonamiAftermath struct {
	XMLTVBuilder *xmltv.TVBuilder
	M3UBuilder   *m3u.M3UBuilder

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
		M3UBuilder: m3u.NewM3UBuilder().
			AddPlaylistHeader(m3u.NewPlaylistHeaderBuilder().
				SetMetadata(m3u.NewPlaylistHeaderMetadataBuilder().
					Build()).
				Build()),
		XMLTVBuilder: xmltv.NewTVBuilder().
			SetDate(time.Now().Format("2006-01-02")).
			SetSourceInfoURL("https://api.toonamiaftermath.com").
			SetSourceInfoName("Toonami Aftermath").
			SetSourceDataURL("https://api.toonamiaftermath.com").
			SetGeneratorInfoName("Toonami Aftermath CLI").
			SetSourceInfoURL("https://github.com/chris102994/toonamiaftermath-cli"),
		transport:    transport,
		client:       client,
		EpisodeCache: episodeCache,
	}

	return this
}

func (t *ToonamiAftermath) Run() error {
	log.WithFields(log.Fields{
		"m3u":   t.M3UBuilder,
		"xmltv": t.XMLTVBuilder,
	}).Info("Scraping With...")

	baseUrl := "https://api.toonamiaftermath.com"

	// Get Channels
	channelsParams := url.Values{
		"startDate": {time.Now().Add(-2 * time.Hour).Format(time.RFC3339)},
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

		t.M3UBuilder.AddChannel(m3u.NewChannelBuilder().
			SetDuration(-0).
			SetTitle(taChannel.Name).
			SetURL(m3uUrl).
			SetMetadata(m3u.NewChannelMetadataBuilder().
				SetChannelID(channelId).
				SetGroupTitle("Toonami Aftermath").
				SetTvgCountry("us").
				SetTvgID(taChannel.Name).
				SetTvgLanguage("en").
				SetTvgName(taChannel.Name).
				Build()).
			Build())

		t.XMLTVBuilder.AddChannel(
			xmltv.NewChannelBuilder().
				SetID(fmt.Sprintf("%v", index+1)).
				AddDisplayName(xmltv.NewDisplayNameBuilder().
					SetLang("en").
					SetText(taChannel.Name).
					Build()).
				AddURL(xmltv.NewURLBuilder().
					SetText(m3uUrl).
					Build()).
				Build())

		log.WithFields(log.Fields{
			"channelName": taChannel.Name,
			"westOffset":  westOffset,
			"channelSlug": channelSlug,
			"m3uUrl":      m3uUrl,
		}).Info("Channel")

		scheduleNameString := strings.ReplaceAll(taChannel.Name, "East", "EST")
		scheduleNameString = strings.ReplaceAll(scheduleNameString, "West", "EST")

		timeOffset, _ := time.ParseDuration("-3h")
		guideParams := url.Values{
			"scheduleName": {scheduleNameString},
			"dateString":   {time.Now().Add(timeOffset).Format(time.RFC3339)},
			"count":        {"150"},
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
				"guideUrl":    guideUrl,
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

		for position, mediaItem := range taMedia {
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

			// Look forward to set the Stop time
			var stopTime string
			if position < len(taMedia)-1 {
				nextMediaItem := taMedia[position+1]
				nextParsedTime, err := time.Parse(time.RFC3339, nextMediaItem.StartDate)
				if err == nil {
					thisStopTime := nextParsedTime.Format("20060102150405")
					if westOffset {
						thisStopTime = fmt.Sprintf("%v %v", thisStopTime, "-0300")
					} else {
						thisStopTime = fmt.Sprintf("%v %v", thisStopTime, "+0000")
					}
					stopTime = thisStopTime
				}
			} else {
				log.WithFields(log.Fields{
					"position":  position,
					"mediaItem": mediaItem,
				}).Trace("Programme last index. No end time.")
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

			programmeBuilder := xmltv.NewProgrammeBuilder().
				SetChannel(channelId).
				SetStart(startTime).
				SetStop(stopTime)

			if mediaItem.Info.Fullname != "" {
				programmeBuilder.AddTitle(xmltv.NewTitleBuilder().
					SetLang("en").
					SetText(mediaItem.Info.Fullname).
					Build())
			} else if mediaItem.Name != "" {
				programmeBuilder.AddTitle(xmltv.NewTitleBuilder().
					SetLang("en").
					SetText(mediaItem.Name).
					Build())
			} else if mediaItem.BlockName != "" {
				programmeBuilder.AddTitle(xmltv.NewTitleBuilder().
					SetLang("en").
					SetText(mediaItem.BlockName).
					Build())
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

				episodeImage := ""
				if mediaItem.Info.Image != "" {
					episodeImage = mediaItem.Info.Image
				} else if episodeInfo.Episode.Image != "" {
					episodeImage = episodeInfo.Episode.Image
				} else if episodeInfo.Image != "" {
					episodeImage = episodeInfo.Image
				}

				if !IsEpisodeInfoEmpty(episodeInfo) {
					programmeBuilder.
						AddTitle(xmltv.NewTitleBuilder().
							SetLang("en").
							SetText(episodeInfo.Name).
							Build()).
						AddSubTitle(xmltv.NewSubTitleBuilder().
							SetLang("en").
							SetText(episodeInfo.Episode.Name).
							Build()).
						AddImage(xmltv.NewImageBuilder().
							SetText(episodeImage).
							Build()).
						AddIcon(xmltv.NewIconBuilder().
							SetSrc(episodeImage).
							Build()).
						AddDesc(xmltv.NewDescBuilder().
							SetLang("en").
							SetText(episodeInfo.Episode.Summary).
							Build())

					if episodeInfo.Episode.AirDate != "" {
						var parsedDate time.Time
						formats := []string{"2 Jan 2006", "2 Jan. 2006", "2006", "Jan. 2006", "Jan 2006"}
						for _, format := range formats {
							parsedDate, err = time.Parse(format, episodeInfo.Episode.AirDate)
							if err == nil {
								break
							}
						}

						if err != nil {
							log.Fatal(err)
						}
						programmeBuilder.SetDate(parsedDate.Format("20060102"))
					}

					unusableRatings := []string{"", "Not Rated"}
					if !slices.Contains(unusableRatings, episodeInfo.ContentRating) {
						programmeBuilder.AddRating(xmltv.NewRatingBuilder().
							SetSystem("VCHIP").
							SetValue(episodeInfo.ContentRating).
							Build())
					}

					if episodeInfo.Rating != 0 {
						programmeBuilder.AddStarRating(xmltv.NewStarRatingBuilder().
							SetSystem("imdb").
							SetValue(fmt.Sprintf("%v/10", episodeInfo.Rating)).
							Build())
					}

					if episodeInfo.Episode.Season != 0 && episodeInfo.Episode.EpNum != 0 {
						programmeBuilder.AddEpisodeNum(xmltv.NewEpisodeNumBuilder().
							SetSystem("xmltv_ns").
							SetText(fmt.Sprintf("%v.%v.0/1", episodeInfo.Episode.Season-1, episodeInfo.Episode.EpNum-1)).
							Build())
					}

					for _, category := range episodeInfo.Genres {
						programmeBuilder.AddCategory(xmltv.NewCategoryBuilder().
							SetLang("en").
							SetText(category).
							Build())
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

					creditsBuilder := xmltv.NewCreditsBuilder()
					for _, writer := range writers {
						creditsBuilder.AddWriter(writer)
					}
					for _, producer := range producers {
						creditsBuilder.AddProducer(producer)
					}

					programmeBuilder.SetCredits(creditsBuilder.Build())
				} else {
					log.WithFields(log.Fields{
						"episodeUrl": episodeUrl,
					}).Trace("No Episode Info")
				}

			}

			log.WithFields(log.Fields{
				"mediaItem": mediaItem,
			}).Trace("Adding Media Item")

			t.XMLTVBuilder.AddProgramme(programmeBuilder.Build())
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
