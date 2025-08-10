package main

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

type WatsonSession struct {
	ID              int      `json:"id"`
	Locations       []string `json:"loc"`
	Name            string   `json:"title"`
	Description     string   `json:"desc"`
	StartTime       string   `json:"dateTime"`
	DurationMinutes int      `json:"mins"`
	Format          string   `json:"format"`
	Tags            []Tag    `json:"tags"`
	Links           Links    `json:"links"`
	People          []Person `json:"people,omitempty"`
	in_person       bool     `json:"-"`
	virtual         bool     `json:"-"`
}

type Tag struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

// Links related to the session
type Links struct {
	Session string `json:"session,omitempty"`
	Stage   string `json:"stage,omitempty"`
	Replay  string `json:"replay,omitempty"`
	Chat    string `json:"chat,omitempty"`
}

type Link struct {
	Label    string `json:"label"`
	URL      string `json:"URL"`
	Category string `json:"category"`
}

type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

const WATSON_TIME_FORMAT string = "2006-01-02T15:04:05.999Z07:00"
const VIRTUAL_ROOM_1 = 5074259
const VIRTUAL_ROOM_2 = 5074260

var notAlphaNumeric = regexp.MustCompile("[^a-zA-Z0-9_]")

func makeTag(label, value, category string) Tag {
	return Tag{
		Label:    label,
		Value:    strings.ToLower(notAlphaNumeric.ReplaceAllLiteralString(strings.ReplaceAll(value, " ", "_"), "")),
		Category: category,
	}
}

// BuildSessionTags builds tags for this session
func (ws *WatsonSession) BuildSessionTags(gs GuidebookSession, gb GuideBook) {
	// This will at worst return an empty set - it will not return an error
	ws.Tags = make([]Tag, 0)

	for _, st := range gs.ScheduleTracks {
		ws.Tags = append(ws.Tags, makeTag(gb.Tracks[st], "track_"+gb.Tracks[st], "Track"))
		if gb.Tracks[st] == "virtual" {
			ws.virtual = true
		}
	}

	for _, loc := range gs.Locations {
		if loc == VIRTUAL_ROOM_1 || loc == VIRTUAL_ROOM_2 {
			ws.virtual = true
		} else {
			ws.in_person = true
		}
	}
	if ws.in_person {
		ws.Tags = append(ws.Tags, makeTag("In Person Session", "session_in_person", "Environment"))
	}
	if ws.virtual {
		ws.Tags = append(ws.Tags, makeTag("Virtual Session", "session_virtual", "Environment"))
	}
}

// BuildSessionLinks builds the "Links" structure for this session
func (ws *WatsonSession) BuildSessionLinks(gs GuidebookSession, gb GuideBook) {
	if ws.virtual {
		ws.Links.Session = fmt.Sprintf("https://virtual.seattlein2025.org/deep-link/session?item_id=%d", ws.ID)
	}
	ws.Links.Chat = fmt.Sprintf("https://virtual.seattlein2025.org/deep-link/chat?item_id=%d", ws.ID)

	// sessionLinks, exists := gb.SessionLinks[gs.ID]
	// if !exists {
	// 	return
	// }
	// for _, cl := range sessionLinks.TargetIDs {
	// 	switch cl.TargetType {
	// 	case GB_TARGET_TYPE_STREAM:
	// 		if strings.HasPrefix(gb.WebViews[cl.TargetID].URL, "https://virtual.seattlein2025.org") {
	// 			ws.Links.Session = gb.WebViews[cl.TargetID].URL
	// 		}
	// 	}
	// }
}

// WatsonFromGuidebook converts everything from the Guidebook structure into an array of WatsonSession.
func WatsonFromGuidebook(gb GuideBook) ([]WatsonSession, error) {

	watson := make([]WatsonSession, 0, len(gb.Sessions))

	for _, gs := range gb.Sessions {
		session := WatsonSession{
			ID:          gs.ID,
			Name:        gs.Name,
			Description: gs.Description,
			StartTime:   gs.StartTime,
			Tags:        make([]Tag, 0),
			Links:       Links{},
		}
		for _, loc := range gs.Locations {
			session.Locations = append(session.Locations, gb.Locations[loc])
		}
		if len(session.Locations) == 0 {
			session.Locations = append(session.Locations, "Discord") // All Hail Eris!
		}
		start, err := time.Parse(GUIDEBOOK_TIME_FORMAT, gs.StartTime)
		if err != nil {
			return watson, err
		}
		finish, err := time.Parse(GUIDEBOOK_TIME_FORMAT, gs.EndTime)
		if err != nil {
			return watson, err
		}
		session.StartTime = start.Format(WATSON_TIME_FORMAT)
		session.DurationMinutes = int(finish.Sub(start) / time.Minute)

		// People in the session are in CustomLinks :-/
		personLinks, exists := gb.SessionLinks[session.ID]
		if exists {
			people := make([]Person, 0, len(personLinks.TargetIDs))
			for _, pl := range personLinks.TargetIDs {
				if pl.TargetType != GB_TARGET_TYPE_PERSON {
					continue
				}
				person := Person{
					ID:   pl.TargetID,
					Name: gb.ListItems[pl.TargetID].Name,
				}
				_, exists := gb.GuestsOfHonor[pl.TargetID]
				if exists {
					person.Role = "Guest of Honor"
				}
				people = append(people, person)
			}
			session.People = people
		}

		session.BuildSessionTags(gs, gb)
		if !(session.in_person || session.virtual) {
			log.Printf("Somehow we have a session (%d, %s) which is neither virtual nor in person: assuming in person", session.ID, session.Name)
			session.in_person = true
		}
		session.BuildSessionLinks(gs, gb)

		watson = append(watson, session)
	}

	sort.Slice(watson, func(i, j int) bool {
		return watson[i].StartTime < watson[j].StartTime
	})

	return watson, nil
}
