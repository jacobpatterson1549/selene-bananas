package db

import (
	"reflect"
	"testing"
)

func TestNewSQLQueryFunction(t *testing.T) {
	want := sqlQueryFunction{
		name: "read_hobbits",
		cols: []string{
			"first_name",
			"last_name",
		},
		arguments: []interface{}{
			"baggins",
			"gamgee",
			"brandybuck",
			"took",
		},
	}
	got := newSQLQueryFunction(
		"read_hobbits",
		[]string{"first_name", "last_name"},
		"baggins", "gamgee", "brandybuck", "took")
	if !reflect.DeepEqual(want, *got) {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestNewSQLExecFunction(t *testing.T) {
	want := sqlExecFunction{
		name: "delete_rings",
		arguments: []interface{}{
			"elf",
			"dwarf",
			"man",
		},
	}
	got := newSQLExecFunction(
		"delete_rings",
		"elf",
		"dwarf",
		"man")
	if !reflect.DeepEqual(want, *got) {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLQueryFunctionCmd(t *testing.T) {
	q := sqlQueryFunction{
		name: "read_hobbits",
		cols: []string{
			"whole_name",
			"age",
		},
		arguments: []interface{}{
			33,
			111,
		},
	}
	want := "SELECT whole_name, age FROM read_hobbits($1, $2)"
	got := q.cmd()
	if want != got {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLExecFunctionCmd(t *testing.T) {
	e := sqlExecFunction{
		name: "kill_orcs",
		arguments: []interface{}{
			"minas ithil",
			"minas tirith",
			"minas morgul",
		},
	}
	want := "SELECT kill_orcs($1, $2, $3)"
	got := e.cmd()
	if want != got {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLRawCmd(t *testing.T) {
	r := sqlExecRaw{"DELETE FROM rings"}
	want := "DELETE FROM rings"
	got := r.cmd()
	if want != got {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLQueryArgs(t *testing.T) {
	q := sqlQueryFunction{
		arguments: []interface{}{
			111,
			"hobbit",
		},
	}
	want := []interface{}{
		111,
		"hobbit",
	}
	got := q.args()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLExecArgs(t *testing.T) {
	q := sqlQueryFunction{
		arguments: []interface{}{
			false,
			"hobbit",
			33,
		},
	}
	want := []interface{}{
		false,
		"hobbit",
		33,
	}
	got := q.args()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestSQLRawArgs(t *testing.T) {
	r := sqlExecRaw{"DELETE FROM rings"}
	got := r.args()
	if got != nil {
		t.Errorf("raw sql should not have arguments, got %v", got)
	}
}
