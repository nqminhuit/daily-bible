package dateutil

import (
	"log"
	"time"

	"github.com/minh/daily-bible/internal/constants"
)

func today() time.Time {
	loc, err := time.LoadLocation(constants.Timezone)
	if err != nil {
		log.Fatalf("fatal: load timezone %q: %v", constants.Timezone, err)
	}
	return time.Now().In(loc)
}

func TodayDate() string {
	return today().Format("2006-01-02")
}
