package applesciprt

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	start1, end1 := "2023-11", "2024-10" // 12个月，跨年
	phases := SplitPhases(start1, end1)

	fmt.Println("原始阶段划分:")
	for _, phase := range phases {
		for _, t := range phase {
			fmt.Print(t.Format("2006-01"), " ")
		}
		fmt.Println()
	}

	mergedPhases := MergePhases(phases)

	fmt.Println("合并后的阶段:")
	for _, phase := range mergedPhases {
		for _, t := range phase {
			fmt.Print(t.Format("2006-01"), " ")
		}
		fmt.Println()
	}

	start2, end2 := "2023-11", "2024-04" // 6个月
	phases2 := SplitPhases(start2, end2)

	fmt.Println("原始阶段划分:")
	for _, phase := range phases2 {
		for _, t := range phase {
			fmt.Print(t.Format("2006-01"), " ")
		}
		fmt.Println()
	}

	mergedPhases2 := MergePhases(phases2)

	fmt.Println("合并后的阶段:")
	for _, phase := range mergedPhases2 {
		for _, t := range phase {
			fmt.Print(t.Format("2006-01"), " ")
		}
		fmt.Println()
	}
}
func TestName2(t *testing.T) {
	data := []float64{1.2, 2.3, 3.4, 4.5, 5.6}
	fmt.Printf("平均值: %.2f\n", Mean(data))
	fmt.Printf("标准差: %.2f\n", StdDev(data))
}

// SplitPhases 根据开始时间和结束时间划分阶段，返回 time.Time 类型的结果
func SplitPhases(start, end string) [][]time.Time {
	layout := "2006-01"
	startTime, err1 := time.Parse(layout, start)
	endTime, err2 := time.Parse(layout, end)

	if err1 != nil || err2 != nil {
		fmt.Println("时间格式错误，请使用YYYY-MM格式")
		return nil
	}

	// 确保 startTime 早于 endTime
	if startTime.After(endTime) {
		startTime, endTime = endTime, startTime
	}

	var months []time.Time
	curTime := startTime

	for !curTime.After(endTime) {
		months = append(months, curTime)
		curTime = curTime.AddDate(0, 1, 0)
	}

	n := len(months)
	var phases [][]time.Time

	if n == 12 { // 12个月，分4个阶段
		for i := 0; i < 4; i++ {
			phases = append(phases, months[i*3:(i+1)*3])
		}
	} else if n == 6 { // 6个月，分2个阶段
		for i := 0; i < 2; i++ {
			phases = append(phases, months[i*3:(i+1)*3])
		}
	} else {
		fmt.Println("输入时间范围不符合要求，仅支持12个月或6个月的划分")
	}

	return phases
}

// MergePhases 结合相邻阶段，并将前一个数组的末尾元素放置到当前数组的开头
func MergePhases(phases [][]time.Time) [][]time.Time {
	if len(phases) == 0 {
		return nil
	}

	var merged [][]time.Time
	merged = append(merged, phases[0]) // 先加入第一个阶段

	for i := 1; i < len(phases); i++ {
		if len(phases[i-1]) > 0 {
			newPhase := append([]time.Time{phases[i-1][len(phases[i-1])-1]}, phases[i]...)
			merged = append(merged, newPhase)
		} else {
			merged = append(merged, phases[i])
		}
	}

	return merged
}

// Mean 计算平均值
func Mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

// StdDev 计算总体标准差
func StdDev(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	mean := Mean(data)
	var sumSquares float64
	for _, v := range data {
		sumSquares += (v - mean) * (v - mean)
	}
	return math.Sqrt(sumSquares / float64(len(data)))
}
func TestSplitPhases_12Months(t *testing.T) {
	phases := SplitPhases("2023-01", "2023-12")
	if len(phases) != 4 {
		t.Errorf("expected 4 phases, got %d", len(phases))
	}
	for i, phase := range phases {
		if len(phase) != 3 {
			t.Errorf("phase %d: expected 3 months, got %d", i, len(phase))
		}
	}
}

func TestSplitPhases_6Months(t *testing.T) {
	phases := SplitPhases("2023-01", "2023-06")
	if len(phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(phases))
	}
	for i, phase := range phases {
		if len(phase) != 3 {
			t.Errorf("phase %d: expected 3 months, got %d", i, len(phase))
		}
	}
}

func TestSplitPhases_InvalidRange(t *testing.T) {
	phases := SplitPhases("2023-01", "2023-03")
	if phases != nil {
		t.Errorf("expected nil for invalid range, got %v", phases)
	}
}

func TestSplitPhases_ReversedOrder(t *testing.T) {
	phases := SplitPhases("2023-06", "2023-01")
	if len(phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(phases))
	}
}

func TestMergePhases(t *testing.T) {
	phases := [][]time.Time{
		{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
	merged := MergePhases(phases)
	if len(merged) != 2 {
		t.Errorf("expected 2 merged phases, got %d", len(merged))
	}
	if len(merged[1]) != 3 {
		t.Errorf("expected merged phase to have 3 elements, got %d", len(merged[1]))
	}
	if !merged[1][0].Equal(phases[0][1]) {
		t.Errorf("expected first element of merged[1] to be %v, got %v", phases[0][1], merged[1][0])
	}
}

func TestMean(t *testing.T) {
	data := []float64{2, 4, 6, 8}
	expected := 5.0
	if got := Mean(data); got != expected {
		t.Errorf("expected mean %.2f, got %.2f", expected, got)
	}
	if Mean([]float64{}) != 0 {
		t.Error("expected mean of empty slice to be 0")
	}
}

func TestStdDev(t *testing.T) {
	data := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	expected := 2.0
	got := StdDev(data)
	if math.Abs(got-expected) > 1e-6 {
		t.Errorf("expected stddev %.2f, got %.6f", expected, got)
	}
	if StdDev([]float64{}) != 0 {
		t.Error("expected stddev of empty slice to be 0")
	}
}
