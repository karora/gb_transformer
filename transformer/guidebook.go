package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// MultiResponse represents the top-level structure of the Guidebook API response for anything that returns a series of results.
type MultiResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []any  `json:"results"`
}

// GuidebookSession represents a single session item from the Guidebook API.
// It includes details about the session, its location, tracks, and speakers.
type GuidebookSession struct {
	ID                  int     `json:"id"`
	Name                string  `json:"name"`
	Description         string  `json:"description_html"`
	StartTime           string  `json:"start_time"`
	EndTime             string  `json:"end_time"`
	AllowRating         bool    `json:"allow_rating"`
	AddToScheduleEnable bool    `json:"add_to_schedule_enabled"`
	AllDay              bool    `json:"all_day"`
	Image               string  `json:"image"`
	Rank                float64 `json:"rank"`
	ModeratorNotes      string  `json:"moderator_notes"`
	Locations           []int   `json:"locations"`
	ScheduleTracks      []int   `json:"schedule_tracks"`
}

// 2017-08-31T20:18:28.038556+0000
const GUIDEBOOK_TIME_FORMAT string = "2006-01-02T15:04:05.999999+0000"

const GUESTS_OF_HONOR_ID = 1153959

// GuidebookLocation represents a location for a session.
type GuidebookLocation struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ScheduleTrack represents a track for a session.
type ScheduleTrack struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CustomList struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Items []int  `json:"items,omitempty"`
}

type ListItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Subtitle    string `json:"subtitle"`
	Thumbnail   string `json:"thumbnail"`
	Descripion  string `json:"description_html"`
	CustomLists []int  `json:"custom_lists"`
	Image       string `json:"image"`
}

type CatLink struct {
	ID         int     `json:"id"`
	Name       string  `json:"title"`
	SourceType string  `json:"source_content_type"`
	TargetType string  `json:"target_content_type"`
	SourceID   int     `json:"source_object_id"`
	TargetID   int     `json:"target_object_id"`
	Rank       float64 `json:"rank"`
	CategoryID int     `json:"category"`
	Category   any     `json:"category_detail"`
}

type ListCategory struct {
	ID    int       `json:"id"`
	Name  string    `json:"name"`
	Links []CatLink `json:"links,omitempty"`
}

type SessionLink struct {
	TargetType string `json:"target_content_type"`
	TargetID   int    `json:"target_object_id"`
}

const GB_TARGET_TYPE_LISTITEM = "custom_list.customlistitem"
const GB_TARGET_TYPE_WEBVIEW = "uri_resource.webview"
const GB_TARGET_TYPE_PERSON = "custom_list.customlistitem"
const GB_TARGET_TYPE_STREAM = "uri_resource.webview"

type SessionList struct {
	SessionID int                 `json:"id"`
	TargetIDs map[int]SessionLink `json:"linked_items"`
}

type WebView struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"webview_type"`
	URL      string `json:"url"`
	HtmlFile string `json:"html_file"`
}

// GuideBook a structure with everything we know from the guidebook
type GuideBook struct {
	config        conf                `json:"-"`
	Sessions      []GuidebookSession  `json:"sessions"`
	Locations     map[int]string      `json:"locations"`
	SessionLinks  map[int]SessionList `json:"session_links"`
	OtherLinks    map[int][]CatLink   `json:"other_links"`
	Lists         map[int]CustomList  `json:"custom_lists"`
	ListItems     map[int]ListItem    `json:"custom_list_items"`
	Tracks        map[int]string      `json:"tracks"`
	GuestsOfHonor map[int]string      `json:"guests_of_honor"`
	WebViews      map[int]WebView     `json:"webviews"`
}

var guideBookRequestCounter = 0

func loadGuidebook(c conf) (gb GuideBook, err error) {
	gb.config = c
	if err = gb.FetchSessions(); err != nil {
		return gb, fmt.Errorf("failed to load sessions from GuideBook: %w", err)
	}

	if err = gb.FetchLocations(); err != nil {
		return gb, fmt.Errorf("failed to load session locations from GuideBook: %w", err)
	}

	if err = gb.FetchTracks(); err != nil {
		return gb, fmt.Errorf("failed to load schedule tracks from GuideBook: %w", err)
	}

	if err = gb.FetchLists(); err != nil {
		return gb, fmt.Errorf("failed to load lists and listitems from GuideBook: %w", err)
	}

	if err = gb.FetchSessionLinks(); err != nil {
		return gb, fmt.Errorf("failed to load session links from GuideBook: %w", err)
	}

	// err = gb.FetchWebViews()

	gb.GuestsOfHonor = make(map[int]string)
	for _, goh := range gb.Lists[GUESTS_OF_HONOR_ID].Items {
		gb.GuestsOfHonor[goh] = gb.ListItems[goh].Name
	}

	return gb, nil
}

func multiFetch(c conf, fetchWhat string) ([]byte, error) {
	var allResults []any
	client := &http.Client{}

	nextURL := fmt.Sprintf("https://builder.guidebook.com/open-api/v1.1/%s/?guide=%s", fetchWhat, c.GuidebookID)

	for nextURL != "" {
	retryAfterWait:
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for %s: %w", fetchWhat, err)
		}

		req.Header.Set("Authorization", "JWT "+c.GuidebookAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request for %s: %w", fetchWhat, err)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == 429 {
				retryWait, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
				if retryWait > 0 {
					log.Printf("We got a 429 on request %d and are now waiting for %d seconds before our next request...", guideBookRequestCounter+1, retryWait)
					time.Sleep(time.Duration(1+retryWait) * time.Second)
					goto retryAfterWait
				}
				log.Println("Well, we got rate limited.  Here's the headers...")
				for key, value := range resp.Header {
					log.Printf("%s: %s", key, value)
				}
			}
			return nil, fmt.Errorf("guidebook API request for %s failed with status %s: %s", fetchWhat, resp.Status, string(bodyBytes))
		}
		guideBookRequestCounter++ // Only successful ones count

		var response MultiResponse
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response); err != nil {
			fmt.Println(string(bodyBytes))
			return nil, fmt.Errorf("failed to decode multi response: %w", err)
		}

		allResults = append(allResults, response.Results...)
		nextURL = response.Next
	}

	log.Printf("Fetched %s chain - %d requests so far.", fetchWhat, guideBookRequestCounter)

	return json.Marshal(allResults)
}

// FetchSessions fetches all sessions from a specific guide in Guidebook.
// It requires an API key and the ID of the guide.
// It handles pagination automatically to retrieve all session records.
func (gb *GuideBook) FetchSessions() error {
	response, err := multiFetch(gb.config, "sessions")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&gb.Sessions); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	return nil
}

// FetchLocations fetches all locations from a specific guide in Guidebook.
func (gb *GuideBook) FetchLocations() error {
	allLocations := make([]GuidebookLocation, 0)
	response, err := multiFetch(gb.config, "locations")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allLocations); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	gb.Locations = make(map[int]string)
	for _, v := range allLocations {
		gb.Locations[v.ID] = v.Name
	}

	return nil
}

// FetchTracks fetches all schedule tracks from a specific guide in Guidebook.
func (gb *GuideBook) FetchTracks() error {
	allTracks := make([]ScheduleTrack, 0)
	response, err := multiFetch(gb.config, "schedule-tracks")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allTracks); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	gb.Tracks = make(map[int]string)
	for _, v := range allTracks {
		gb.Tracks[v.ID] = v.Name
	}

	return nil
}

// FetchLists fetches all custom-lists from a specific guide in Guidebook.
func (gb *GuideBook) FetchLists() error {
	customLists := make([]CustomList, 0)
	response, err := multiFetch(gb.config, "custom-lists")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&customLists); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	gb.Lists = make(map[int]CustomList)
	for _, cl := range customLists {
		gb.Lists[cl.ID] = cl
	}

	allItems := make([]ListItem, 0, 1000)
	response, err = multiFetch(gb.config, "custom-list-items")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allItems); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	gb.ListItems = make(map[int]ListItem)
	for _, v := range allItems {
		gb.ListItems[v.ID] = v
	}

	for id, li := range gb.ListItems {
		for _, w := range li.CustomLists {
			list := gb.Lists[w]
			list.Items = append(list.Items, id)
			gb.Lists[w] = list
		}
	}

	return nil
}

// FetchSessionLinks fetches the link categories related to a session
func (gb *GuideBook) ExFetchSessionLinks() error {
	listCats := make([]ListCategory, 0)
	response, err := multiFetch(gb.config, "link-categories")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&listCats); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	gb.OtherLinks = make(map[int][]CatLink)
	gb.SessionLinks = make(map[int]SessionList)
	for _, v := range listCats {
		for _, w := range v.Links {
			if w.SourceType == "schedule.session" {
				list, exists := gb.SessionLinks[w.SourceID]
				if !exists {
					list = SessionList{
						SessionID: w.SourceID,
						TargetIDs: make(map[int]SessionLink, 0),
					}
				}
				list.TargetIDs[w.TargetID] = SessionLink{
					TargetType: w.TargetType,
					TargetID:   w.TargetID,
				}
				gb.SessionLinks[w.SourceID] = list
			} else {
				list, exists := gb.OtherLinks[w.SourceID]
				if !exists {
					list = make([]CatLink, 0)
				}
				list = append(list, w)
				gb.OtherLinks[w.SourceID] = list
			}
		}
	}
	return nil
}

// FetchSessionLinks fetches the link categories related to a session
func (gb *GuideBook) FetchSessionLinks() error {
	listCats := make([]CatLink, 0)
	response, err := multiFetch(gb.config, "links")
	if err != nil {
		return fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&listCats); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	gb.OtherLinks = make(map[int][]CatLink)
	gb.SessionLinks = make(map[int]SessionList)
	for _, w := range listCats {
		if w.SourceType == "schedule.session" {
			list, exists := gb.SessionLinks[w.SourceID]
			if !exists {
				list = SessionList{
					SessionID: w.SourceID,
					TargetIDs: make(map[int]SessionLink, 0),
				}
			}
			list.TargetIDs[w.TargetID] = SessionLink{
				TargetType: w.TargetType,
				TargetID:   w.TargetID,
			}
			gb.SessionLinks[w.SourceID] = list
		} else {
			list, exists := gb.OtherLinks[w.SourceID]
			if !exists {
				list = make([]CatLink, 0)
			}
			list = append(list, w)
			gb.OtherLinks[w.SourceID] = list
		}
	}
	return nil
}

// FetchWebViews fetches the webviews related to a session
func (gb *GuideBook) FetchWebViews() error {
	response, err := multiFetch(gb.config, "webviews")
	if err != nil {
		return fmt.Errorf("failed to fetch webviews results: %w", err)
	}

	webViews := make([]WebView, 0)
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&webViews); err != nil {
		fmt.Println(string(response))
		return fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	gb.WebViews = make(map[int]WebView)
	for _, wv := range webViews {
		gb.WebViews[wv.ID] = wv
	}
	return nil
}
