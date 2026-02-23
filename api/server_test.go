package api

import (
	"fmt"
	"testing"
)

func TestFillInTrendCycle(t *testing.T) {
	rangeDays := 7
	rangeDayList := FillInTrendCycle(rangeDays)
	fmt.Println("rangeDayList", rangeDayList)
	fmt.Println("rangeDayList len ", len(rangeDayList))
}
