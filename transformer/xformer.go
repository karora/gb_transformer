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
	SchedulePath    string
	StreamPath      string
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
	config.SchedulePath = getEnvWithDefault("SCHEDULE_PATH", "/var/www/html/schedule.json")
	config.StreamPath = getEnvWithDefault("STREAM_PATH", "/var/www/html/streaming.csv")
	config.GuidebookAPIKey = getEnvWithDefault("GB_API_KEY", "not set")
	config.GuidebookID = getEnvWithDefault("GB_ID", "not set")

	flag.BoolVar(&config.Dump, "dump", false, "dumps the full contents we've loaded from GuideBook as JSON")
	flag.Parse()

	if !config.Dump {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}
	if config.SchedulePath == config.StreamPath {
		log.Fatal("SCHEDULE_PATH and STREAM_PATH must be set to different values.")
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

		f, err := os.OpenFile(config.StreamPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Error opening file %q for writing streaming CSV: %s", config.StreamPath, err.Error())
		} else {
			StreamingCSV(f, watsonSessions)
			f.Close()
		}
	}

	// // When something is written into the config.TimeToGo channel we quit.
	// <-config.TimeToGo

	// log.Println("Xformer exiting.")
}
