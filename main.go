package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/jamespfennell/gtfs"
	"github.com/rodaine/table"
)

func main() {

	var data []byte

	if _, err := os.Stat("./transitData.zip"); errors.Is(err, os.ErrNotExist) {
		fmt.Println("Downloading transit data...")

		gtfsResp, err := http.Get("https://www.wroclaw.pl/open-data/87b09b32-f076-4475-8ec9-6020ed1f9ac0/OtwartyWroclaw_rozklad_jazdy_GTFS.zip")

		if err != nil {
			panic(err)
		}

		defer gtfsResp.Body.Close()

		data, err = io.ReadAll(gtfsResp.Body)

		if err != nil {
			panic(err)
		}

		os.WriteFile("./transitData.zip", data, 0644)
	} else {
		data, err = os.ReadFile("./transitData.zip")

		if err != nil {
			panic(err)
		}
	}

	staticData, err := gtfs.ParseStatic(data, gtfs.ParseStaticOptions{})

	if err != nil {
		panic(err)
	}

	switch os.Args[1] {
	case "stop":
		stopName := os.Args[2]

		stopIds := make([]string, 0)
		stopArrivals := make([]StopArrival, 0)

		for _, stop := range staticData.Stops {
			if strings.Contains(stop.Name, stopName) {
				stopIds = append(stopIds, stop.Id)
			}
		}

		for _, trip := range staticData.Trips {
			for _, stopTime := range trip.StopTimes {
				for _, stopId := range stopIds {
					if stopTime.Stop.Id == stopId {
						stopArrivals = append(stopArrivals, StopArrival{
							Line:     trip.Route.ShortName,
							Head:     trip.Headsign,
							Time:     stopTime.ArrivalTime,
							StopName: stopTime.Stop.Name,
						})
						break
					}
				}
			}
		}

		if len(stopArrivals) == 0 {
			fmt.Println("No arrivals found")
		}

		earliestArrivals := make(map[string]StopArrival)

		currentDuration, _ := time.ParseDuration(fmt.Sprintf("%dh%dm", time.Now().Hour(), time.Now().Minute()))
		minuteDuration := int(currentDuration.Minutes())

		for _, arrival := range stopArrivals {
			tag := arrival.Head + arrival.Line

			if stopData, ok := earliestArrivals[tag]; ok {
				oldDiff := int(stopData.Time.Minutes()) - minuteDuration
				newDiff := int(arrival.Time.Minutes()) - minuteDuration

				if newDiff < 0 {
					continue
				}

				if newDiff < oldDiff {
					earliestArrivals[tag] = arrival
				} else {
					continue
				}
			} else {
				if int(arrival.Time.Minutes())-minuteDuration < 0 {
					continue
				}
				earliestArrivals[tag] = arrival
			}
		}

		tbl := table.New("Linia", "Kierunek", "Przyjazd", "Przystanek")
		for _, arrival := range earliestArrivals {
			tbl.AddRow(arrival.Line, ToTitleCase(arrival.Head), SimplifyGTFSDuration(arrival.Time), arrival.StopName)
		}

		tbl.Print()
	}
}

func SimplifyGTFSDuration(dur time.Duration) string {
	totalMinutes := int(dur.Minutes())
	hours := (totalMinutes / 60) % 24
	minutes := totalMinutes % 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

func SimpleDuration(dur time.Duration) string {
	totalMinutes := int(dur.Minutes())
	hours := (totalMinutes / 60) % 24
	minutes := totalMinutes % 60
	return fmt.Sprintf("%02dh:%02dm", hours, minutes)
}

type StopArrival struct {
	StopName string
	Line     string
	Head     string
	Time     time.Duration
}

func ToTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
