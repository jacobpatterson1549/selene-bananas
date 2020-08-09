// +build js,wasm

package user

import (
	"reflect"
	"testing"
)

func TestParseUserInfoJSON(t *testing.T) {
	parseUserInfoJSONTests := []struct {
		json   string
		wantOk bool
		want   *userInfo
	}{
		{},
		{
			json: `{"Sub":"selene","points":18}`,
		},
		{
			json: `{"sub":18,"points":18}`,
		},
		{
			json: `{"sub":"selene","Points":18}`,
		},
		{
			json: `{"sub":"selene","points":"18"}`,
		},
		{
			json:   `{"sub":"selene","points":18}`,
			wantOk: true,
			want: &userInfo{
				Name: "selene",
				Points:   18,
			},
		},
	}
	for i, test := range parseUserInfoJSONTests {
		got, err := parseUserInfoJSON(test.json)
		switch {
		case err != nil:
			if test.wantOk {
				t.Errorf("Test %v: unwanted error: %v", i, err)
			}
		case !test.wantOk:
			t.Errorf("Test %v: wanted error", i)
		case !reflect.DeepEqual(test.want, got):
			t.Errorf("Test %v:\nwanted %v\ngot    %v", i, test.want, *got)
		}
	}
}
