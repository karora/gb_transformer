package main

import (
	"regexp"
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
}

type Tag struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

type Links struct {
	Label string `json:"label,omitempty"`
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

func BuildSessionTags(ws WatsonSession, gs GuidebookSession, gb GuideBook) []Tag {
	// This will at worst return an empty set - it will not return an error
	tags := make([]Tag, 0)
	for _, st := range gs.ScheduleTracks {
		tags = append(tags, makeTag(gb.Tracks[st], "track_"+gb.Tracks[st], "Track"))
	}
	virtual := false
	in_person := false
	for _, loc := range gs.Locations {
		if loc == VIRTUAL_ROOM_1 || loc == VIRTUAL_ROOM_2 {
			virtual = true
		} else {
			in_person = true
		}
	}
	if virtual && in_person {
		tags = append(tags, makeTag("In Person and Virtual Session", "session_in_person", "Environment"))
	} else if virtual {
		tags = append(tags, makeTag("Virtual Session", "session_virtual", "Environment"))
	} else {
		tags = append(tags, makeTag("In Person Session", "session_in_person", "Environment"))
	}
	return tags
}

func WatsonFromGuidebook(gb GuideBook) ([]WatsonSession, error) {

	watson := make([]WatsonSession, 0, len(gb.Sessions))

	for _, v := range gb.Sessions {
		session := WatsonSession{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			StartTime:   v.StartTime,
			Tags:        make([]Tag, 0),
			Links:       Links{},
		}
		for _, w := range v.Locations {
			session.Locations = append(session.Locations, gb.Locations[w])
		}
		start, err := time.Parse(GuidebookTimeFormat, v.StartTime)
		if err != nil {
			return watson, err
		}
		finish, err := time.Parse(GuidebookTimeFormat, v.EndTime)
		if err != nil {
			return watson, err
		}
		session.StartTime = start.Format(WATSON_TIME_FORMAT)
		session.DurationMinutes = int(finish.Sub(start) / time.Minute)

		// People in he session are in CustomLinks :-/
		personLinks, exists := gb.SessionLinks[session.ID]
		if exists {
			people := make([]Person, 0, len(personLinks.TargetIDs))
			for pl := range personLinks.TargetIDs {
				person := Person{
					ID:   pl,
					Name: gb.ListItems[pl].Name,
				}
				people = append(people, person)
			}
			session.People = people
		}

		session.Tags = BuildSessionTags(session, v, gb)

		watson = append(watson, session)
	}
	return watson, nil
}
