package sql

import (
	"reflect"
	"testing"
)

func TestNewQueryFunction(t *testing.T) {
	want := QueryFunction{
		name: "read_hobbits",
		cols: []string{
			"first_name",
			"last_name",
		},
		arguments: []any{
			"baggins",
			"gamgee",
			"brandybuck",
			"took",
		},
	}
	got := NewQueryFunction(
		"read_hobbits",
		[]string{"first_name", "last_name"},
		"baggins", "gamgee", "brandybuck", "took")
	if !reflect.DeepEqual(want, got) {
		t.Errorf("queries not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestNewExecFunction(t *testing.T) {
	want := ExecFunction{
		name: "delete_rings",
		arguments: []any{
			"elf",
			"dwarf",
			"man",
		},
	}
	got := NewExecFunction(
		"delete_rings",
		"elf",
		"dwarf",
		"man")
	if !reflect.DeepEqual(want, got) {
		t.Errorf("exec functions not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestQueryFunctionCmd(t *testing.T) {
	q := QueryFunction{
		name: "read_hobbits",
		cols: []string{
			"whole_name",
			"age",
		},
		arguments: []any{
			33,
			111,
		},
	}
	want := "SELECT whole_name, age FROM read_hobbits($1, $2)"
	got := q.Cmd()
	if want != got {
		t.Errorf("cmd functions not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestExecFunctionCmd(t *testing.T) {
	e := ExecFunction{
		name: "kill_orcs",
		arguments: []any{
			"barad-dur",
			"minas tirith",
			"minas morgul",
		},
	}
	want := "SELECT kill_orcs($1, $2, $3)"
	got := e.Cmd()
	if want != got {
		t.Errorf("exec function commands not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestRawQueryCmd(t *testing.T) {
	r := RawQuery("DELETE FROM wings")
	want := "DELETE FROM wings"
	got := r.Cmd()
	if want != got {
		t.Errorf("raw query commands not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestQueryFunctionArgs(t *testing.T) {
	q := QueryFunction{
		arguments: []any{
			111,
			"hobbit",
		},
	}
	want := []any{
		111,
		"hobbit",
	}
	got := q.Args()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("query args not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestExecFunctionArgs(t *testing.T) {
	e := ExecFunction{
		arguments: []any{
			false,
			"hobbit",
			33,
		},
	}
	want := []any{
		false,
		"hobbit",
		33,
	}
	got := e.Args()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("exec function args not equal\nwanted %v\ngot    %v", want, got)
	}
}

func TestRawQueryArgs(t *testing.T) {
	r := RawQuery("DELETE FROM rings")
	got := r.Args()
	if got != nil {
		t.Errorf("raw sql should not have arguments, got %v", got)
	}
}
