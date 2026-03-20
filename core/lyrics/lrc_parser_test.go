package lyrics

import "testing"

func TestParseLRC(t *testing.T) {
	t.Parallel()

	input := `[Verse]
[00:03.010]自然赠予你
[00:04.50]树冠
[00:04.5][00:06.000]微风
[ar:Radiohead]`

	lines := ParseLRC(input)
	if len(lines) != 4 {
		t.Fatalf("expected 4 parsed lines, got %d", len(lines))
	}

	cases := []struct {
		index  int
		timeMs int64
		text   string
	}{
		{index: 0, timeMs: 3010, text: "自然赠予你"},
		{index: 1, timeMs: 4500, text: "树冠"},
		{index: 2, timeMs: 4500, text: "微风"},
		{index: 3, timeMs: 6000, text: "微风"},
	}

	for _, tc := range cases {
		line := lines[tc.index]
		if line.TimeMs != tc.timeMs || line.Text != tc.text {
			t.Fatalf("line %d mismatch: got (%d, %q)", tc.index, line.TimeMs, line.Text)
		}
	}
}

func TestIsSyncedLRC(t *testing.T) {
	t.Parallel()

	if !IsSyncedLRC("[00:04.5]hello") {
		t.Fatal("expected timed lyric to be treated as synced")
	}
	if IsSyncedLRC("[Verse]\n[ar:Radiohead]\nhello") {
		t.Fatal("expected metadata-only lyric to be treated as plain text")
	}
}
