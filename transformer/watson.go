package main

import (
	"strings"
	"time"
)

type WatsonSession struct {
	ID              int      `json:"id"`
	Locations       []string `json:"loc"`
	Name            string   `json:"title"`
	Description     string   `json:"desc"`
	StartTime       string   `json:"datetime"`
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

const WatsonTimeFormat string = "2006-01-02T15:04:05.999Z07:00"

func makeTag(category, label string) Tag {
	return Tag{
		Label: label,
		// Probably really want to just regexp.ReplaceAll, if we need more than this...
		Value:    category + "_" + strings.ReplaceAll(strings.ReplaceAll(label, " ", ""), "&", ""),
		Category: category,
	}
}

func BuildSessionTags(ws WatsonSession, gs GuidebookSession, gb GuideBook) []Tag {
	// This will at worst return an empty set - it will not return an error
	tags := make([]Tag, 0)
	for _, st := range gs.ScheduleTracks {
		tags = append(tags, makeTag("track", gb.Tracks[st]))
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
		session.StartTime = start.Format(WatsonTimeFormat)
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
