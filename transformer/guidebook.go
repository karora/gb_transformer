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

func loadGuidebook() (gb GuideBook, err error) {
	gb.Sessions, err = fetchGuidebookSessions(config)
	if err != nil {
		return gb, fmt.Errorf("failed to load sessions from GuideBook: %w", err)
	}

	gb.Locations, err = fetchGuidebookLocations(config)
	if err != nil {
		return gb, fmt.Errorf("failed to load session locations from GuideBook: %w", err)
	}

	gb.Tracks, err = fetchGuidebookTracks(config)
	if err != nil {
		return gb, fmt.Errorf("failed to load schedule tracks from GuideBook: %w", err)
	}

	gb.SessionLinks, gb.OtherLinks, err = fetchSessionLists(config)
	if err != nil {
		return gb, fmt.Errorf("failed to load session links from GuideBook: %w", err)
	}

	gb.Lists, gb.ListItems, err = fetchGuidebookLists(config)
	if err != nil {
		return gb, fmt.Errorf("failed to load lists and listitems from GuideBook: %w", err)
	}

	// gb.WebViews, err = fetchWebViews(config)

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

// fetchGuidebookSessions fetches all sessions from a specific guide in Guidebook.
// It requires an API key and the ID of the guide.
// It handles pagination automatically to retrieve all session records.
func fetchGuidebookSessions(c conf) ([]GuidebookSession, error) {
	var allSessions []GuidebookSession

	response, err := multiFetch(c, "sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allSessions); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	return allSessions, nil
}

// fetchGuidebookLocations fetches all locations from a specific guide in Guidebook.
func fetchGuidebookLocations(c conf) (map[int]string, error) {
	allLocations := make([]GuidebookLocation, 0)
	response, err := multiFetch(c, "locations")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allLocations); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	locationMap := make(map[int]string)
	for _, v := range allLocations {
		locationMap[v.ID] = v.Name
	}

	return locationMap, nil
}

// fetchGuidebookTracks fetches all schedule tracks from a specific guide in Guidebook.
func fetchGuidebookTracks(c conf) (map[int]string, error) {
	allTracks := make([]ScheduleTrack, 0)
	response, err := multiFetch(c, "schedule-tracks")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allTracks); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	trackMap := make(map[int]string)
	for _, v := range allTracks {
		trackMap[v.ID] = v.Name
	}

	return trackMap, nil
}

func fetchAllListItems(c conf) (map[int]ListItem, error) {
	allItems := make([]ListItem, 0, 1000)
	response, err := multiFetch(c, "custom-list-items")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&allItems); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}
	itemMap := make(map[int]ListItem)
	for _, v := range allItems {
		itemMap[v.ID] = v
	}
	return itemMap, nil

}

// fetchGuidebookLists fetches all custom-lists from a specific guide in Guidebook.
func fetchGuidebookLists(c conf) (map[int]CustomList, map[int]ListItem, error) {
	customLists := make([]CustomList, 0)
	response, err := multiFetch(c, "custom-lists")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&customLists); err != nil {
		fmt.Println(string(response))
		return nil, nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	listsMap := make(map[int]CustomList)
	for _, v := range customLists {
		listsMap[v.ID] = v
	}

	listItems, err := fetchAllListItems(c)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch all custom list items: %w", err)
	}
	for id, v := range listItems {
		for _, w := range v.CustomLists {
			list := listsMap[w]
			list.Items = append(list.Items, id)
			listsMap[w] = list
		}
	}

	return listsMap, listItems, nil
}

// fetchSessionLists fetches the customlists related to a session
func fetchSessionLists(c conf) (map[int]SessionList, map[int][]CatLink, error) {
	listCats := make([]ListCategory, 0)
	response, err := multiFetch(c, "link-categories")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&listCats); err != nil {
		fmt.Println(string(response))
		return nil, nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	otherListsMap := make(map[int][]CatLink)
	sessionListsMap := make(map[int]SessionList)
	for _, v := range listCats {
		for _, w := range v.Links {
			if w.SourceType == "schedule.session" {
				list, exists := sessionListsMap[w.SourceID]
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
				sessionListsMap[w.SourceID] = list
			} else {
				list, exists := otherListsMap[w.SourceID]
				if !exists {
					list = make([]CatLink, 0)
				}
				list = append(list, w)
				otherListsMap[w.SourceID] = list
			}
		}
	}
	return sessionListsMap, otherListsMap, nil
}

// fetchWebViews fetches the webviews related to a session
func fetchWebViews(c conf) (map[int]WebView, error) {
	response, err := multiFetch(c, "webviews")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch webviews results: %w", err)
	}

	webViews := make([]WebView, 0)
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&webViews); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	viewsMap := make(map[int]WebView)
	for _, wv := range webViews {
		viewsMap[wv.ID] = wv
	}
	return viewsMap, nil
}
