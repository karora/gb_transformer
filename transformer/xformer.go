package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

type conf struct {
	OutputPath      string
	GuidebookAPIKey string
	GuidebookID     string
	Dump            bool
	Debug           bool
	SlowDown        time.Duration
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

	flag.BoolVar(&config.Dump, "dump", false, "dumps the full contents we've loaded from GuideBook as JSON")
	flag.Parse()

	if !config.Dump {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}
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

	log.Println("Started fetching from Guidebook")
	guidebook, err := loadGuidebook(config)
	log.Println("Guidebook fetch complete")
	if err != nil {
		log.Fatal(err.Error())
	}
	if err != nil {
		log.Fatal(err.Error())
	}
	if config.Dump {
		DumpJSON(guidebook)
	} else {

		watsonSessions, err := WatsonFromGuidebook(guidebook)
		if err != nil {
			log.Fatal(err.Error())
		}

		DumpJSON(watsonSessions)
	}

	// // When something is written into the config.TimeToGo channel we quit.
	// <-config.TimeToGo

	// log.Println("Xformer exiting.")
}
