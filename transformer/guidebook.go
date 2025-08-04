package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

type LinkCategory struct {
	ID    int       `json:"id"`
	Name  string    `json:"name"`
	Links []CatLink `json:"links,omitempty"`
}

type SessionLink struct {
	SessionID int            `json:"id"`
	TargetIDs map[int]string `json:"linked_items"`
}

// GuideBook a structure with everything we know from the guidebook
type GuideBook struct {
	Sessions     []GuidebookSession  `json:"sessions"`
	Locations    map[int]string      `json:"locations"`
	SessionLinks map[int]SessionLink `json:"session_links"`
	Lists        map[int]CustomList  `json:"custom_lists"`
	ListItems    map[int]ListItem    `json:"custom_list_items"`
	Tracks       map[int]string      `json:"tracks"`
}

func loadGuidebook() (gb GuideBook, err error) {
	gb.Sessions, err = fetchGuidebookSessions(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		return gb, fmt.Errorf("failed to load sessions from GuideBook: %w", err)
	}

	gb.Locations, err = fetchGuidebookLocations(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		return gb, fmt.Errorf("failed to load session locations from GuideBook: %w", err)
	}

	gb.Tracks, err = fetchGuidebookTracks(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		return gb, fmt.Errorf("failed to load schedule tracks from GuideBook: %w", err)
	}

	gb.SessionLinks, err = fetchSessionLinks(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		return gb, fmt.Errorf("failed to load session links from GuideBook: %w", err)
	}

	gb.Lists, gb.ListItems, err = fetchGuidebookLists(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		return gb, fmt.Errorf("failed to load lists and listitems from GuideBook: %w", err)
	}

	return gb, nil
}

func multiFetch(apiKey, guideID, fetchWhat string) ([]byte, error) {
	var allResults []any
	client := &http.Client{}

	nextURL := fmt.Sprintf("https://builder.guidebook.com/open-api/v1.1/%s/?guide=%s", fetchWhat, guideID)

	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "JWT "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("guidebook API request failed with status %s: %s", resp.Status, string(bodyBytes))
		}

		var response MultiResponse
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&response); err != nil {
			fmt.Println(string(bodyBytes))
			return nil, fmt.Errorf("failed to decode multi response: %w", err)
		}

		allResults = append(allResults, response.Results...)
		nextURL = response.Next
	}

	return json.Marshal(allResults)
}

// fetchGuidebookSessions fetches all sessions from a specific guide in Guidebook.
// It requires an API key and the ID of the guide.
// It handles pagination automatically to retrieve all session records.
func fetchGuidebookSessions(apiKey, guideID string) ([]GuidebookSession, error) {
	var allSessions []GuidebookSession

	response, err := multiFetch(apiKey, guideID, "sessions")
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
func fetchGuidebookLocations(apiKey, guideID string) (map[int]string, error) {
	allLocations := make([]GuidebookLocation, 0)
	response, err := multiFetch(apiKey, guideID, "locations")
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
func fetchGuidebookTracks(apiKey, guideID string) (map[int]string, error) {
	allTracks := make([]ScheduleTrack, 0)
	response, err := multiFetch(apiKey, guideID, "schedule-tracks")
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

func fetchAllListItems(apiKey, guideID string) (map[int]ListItem, error) {
	allItems := make([]ListItem, 0, 1000)
	response, err := multiFetch(apiKey, guideID, "custom-list-items")
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
func fetchGuidebookLists(apiKey, guideID string) (map[int]CustomList, map[int]ListItem, error) {
	customLists := make([]CustomList, 0)
	response, err := multiFetch(apiKey, guideID, "custom-lists")
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

	listItems, err := fetchAllListItems(apiKey, guideID)
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

// fetchSessionLinks(apiKey, gu)
func fetchSessionLinks(apiKey, guideID string) (map[int]SessionLink, error) {
	linkCats := make([]LinkCategory, 0)
	response, err := multiFetch(apiKey, guideID, "link-categories")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch results: %w", err)
	}
	if err := json.NewDecoder(bytes.NewReader(response)).Decode(&linkCats); err != nil {
		fmt.Println(string(response))
		return nil, fmt.Errorf("failed to decode guidebook response: %w", err)
	}

	listsMap := make(map[int]SessionLink)
	for _, v := range linkCats {
		for _, w := range v.Links {
			if w.SourceType == "schedule.session" && w.TargetType == "custom_list.customlistitem" {
				list, exists := listsMap[w.SourceID]
				if !exists {
					list = SessionLink{
						SessionID: w.SourceID,
						TargetIDs: make(map[int]string, 0),
					}
				}
				list.TargetIDs[w.TargetID] = w.Name
				listsMap[w.SourceID] = list
			}
		}
	}
	return listsMap, nil
}
