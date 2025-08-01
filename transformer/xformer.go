package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

type conf struct {
	OutputPath      string
	GuidebookAPIKey string
	GuidebookID     string
	DumpLists       bool
	Debug           bool
	TimeToGo        chan (bool)
}

var (
	config conf
	ctx    context.Context
)

func getEnvWithDefault(key string, defaultValue string) string {
	result, present := os.LookupEnv(key)
	if !present {
		result = defaultValue
	}
	if config.Debug {
		log.Printf("%s is %q", key, result)
	}
	return result
}

func init() {
	config.Debug = os.Getenv("XFORMER_DEBUG") == "true"
	config.OutputPath = getEnvWithDefault("OUTPUT_PATH", "/opt/schedule.json")
	config.GuidebookAPIKey = getEnvWithDefault("GB_API_KEY", "not set")
	config.GuidebookID = getEnvWithDefault("GB_ID", "not set")

	flag.BoolVar(&config.DumpLists, "customlist", false, "dumps the list of custom lists as JSON")
	flag.Parse()
}

func DumpJSON(v any) {
	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(string(jsonBytes))

}

func main() {
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	links, err := fetchSessionLinks(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		log.Fatal(err.Error())
	}
	lists, listItems, err := fetchGuidebookLists(config.GuidebookAPIKey, config.GuidebookID)
	if err != nil {
		log.Fatal(err.Error())
	}
	if config.DumpLists {
		// Dumping like this gives an error in the JSON which can make it easier to find where that breakpoint is
		DumpJSON(lists)
		DumpJSON(listItems)
	} else {
		sessions, err := fetchGuidebookSessions(config.GuidebookAPIKey, config.GuidebookID)
		if err != nil {
			log.Fatal(err.Error())
		}

		locations, err := fetchGuidebookLocations(config.GuidebookAPIKey, config.GuidebookID)
		if err != nil {
			log.Fatal(err.Error())
		}

		if config.Debug {
			DumpJSON(sessions)
		}

		watsonSessions, err := WatsonFromGuidebook(sessions, locations, lists, listItems, links)
		if err != nil {
			log.Fatal(err.Error())
		}

		DumpJSON(watsonSessions)
	}

	// // When something is written into the config.TimeToGo channel we quit.
	// <-config.TimeToGo

	// log.Println("Xformer exiting.")
}
