package ai

import (
	"fmt"
	"testing"
)

func TestGetTrackInsightSchema(t *testing.T) {
	schema := GetTrackInsightSchema()
	fmt.Println(schema)
}
