package main

import "time"

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

func WatsonFromGuidebook(sessions []GuidebookSession, locations map[int]string,
	lists map[int]CustomList, listItems map[int]ListItem, links map[int]SessionLink) ([]WatsonSession, error) {

	watson := make([]WatsonSession, 0, len(sessions))

	for _, v := range sessions {
		session := WatsonSession{
			ID:          v.ID,
			Name:        v.Name,
			Description: v.Description,
			StartTime:   v.StartTime,
			Tags:        make([]Tag, 0),
			Links:       Links{},
		}
		for _, w := range v.Locations {
			session.Locations = append(session.Locations, locations[w])
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
		personLinks, exists := links[session.ID]
		if exists {
			people := make([]Person, 0, len(personLinks.TargetIDs))
			for pl := range personLinks.TargetIDs {
				person := Person{
					ID:   pl,
					Name: listItems[pl].Name,
				}
				people = append(people, person)
			}
			session.People = people
		}

		watson = append(watson, session)
	}
	return watson, nil
}
