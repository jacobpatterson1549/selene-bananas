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
		},
		{
			info: Info{
				Status:   NotStarted,
				Capacity: 1,
			},
			want: true,
		},
		{
			info: Info{
				Status:   InProgress,
				Players:  []string{"fred", "selene"},
				Capacity: 2,
			},
			playerName: "selene",
			want:       true,
		},
		{
			info: Info{
				Status:   InProgress,
				Players:  []string{"selene"},
				Capacity: 1,
			},
			playerName: "fred",
		},
		{
			info: Info{
				Status:   InProgress,
				Players:  []string{"selene"},
				Capacity: 4,
			},
			playerName: "fred",
		},
		{
			info: Info{
				Status:   Finished,
				Players:  []string{"selene"},
				Capacity: 1,
			},
			playerName: "selene",
			want:       true,
		},
	}
	for i, test := range canJoinTests {
		got := test.info.CanJoin(test.playerName)
		if test.want != got {
			t.Errorf("Test %v: wanted CanJoin() = %v, got %v when info is %v", i, test.want, got, test.info)
		}
	}
}

func TestCapacityRatio(t *testing.T) {
	capacityRatioTests := []struct {
		players  []string
		capacity int
		want     string
		Info
	}{
		{
			want: "0/0",
		},
		{
			want: "0/4",
			Info: Info{
				Capacity: 4,
			},
		},
		{
			want: "1/2",
			Info: Info{
				Players:  []string{"selene"},
				Capacity: 2,
			},
		},
		{
			want: "3/3",
			Info: Info{
				Players:  []string{"selene", "fred", "barney"},
				Capacity: 3,
			},
		},
		{
			want: "3/4",
			Info: Info{
				Players:  []string{"selene", "fred", "barney"},
				Capacity: 4,
			},
		},
	}
	for i, test := range capacityRatioTests {
		got := test.Info.CapacityRatio()
		if test.want != got {
			t.Errorf("Test %v: wanted capacity ratio of '%v', got '%v' when info is %v", i, test.want, got, test.Info)
		}
	}
}
