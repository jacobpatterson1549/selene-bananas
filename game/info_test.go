package game

import "testing"

func TestCanJoinInfo(t *testing.T) {
	canJoinTests := []struct {
		info       Info
		playerName string
		want       bool
	}{
		{},
		{
			info: Info{
				Status: NotStarted,
			},
			want: true,
		},
		{
			info: Info{
				Status:  InProgress,
				Players: []string{"fred", "selene"},
			},
			playerName: "selene",
			want:       true,
		},
		{
			info: Info{
				Status:  InProgress,
				Players: []string{"selene"},
			},
			playerName: "fred",
		},
		{
			info: Info{
				Status:  Finished,
				Players: []string{"selene"},
			},
			playerName: "selene",
			want:       true,
		},
	}
	for i, test := range canJoinTests {
		got := test.info.CanJoin(test.playerName)
		if test.want != got {
			t.Errorf("Test %v: wanted %v, got %v", i, test.want, got)
		}
	}
}
