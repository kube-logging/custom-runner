package config

import (
	"reflect"
	"testing"
)

func TestImap_SetIn_MapOnly(t *testing.T) {
	type args struct {
		keys  []string
		value string
	}
	tests := []struct {
		name string
		s    Imap
		args args
		want Imap
	}{
		{
			name: "nil keys",
			s:    Imap{},
			args: args{keys: nil, value: "myval"},
			want: Imap{},
		},
		{
			name: "empty keys",
			s:    Imap{},
			args: args{keys: []string{}, value: "myval"},
			want: Imap{},
		},
		{
			name: "single key - empty map",
			s:    Imap{},
			args: args{keys: []string{"foo"}, value: "myval"},
			want: Imap{"foo": "myval"},
		},
		{
			name: "multi key - empty map",
			s:    Imap{},
			args: args{keys: []string{"foo", "bar"}, value: "myval"},
			want: Imap{"foo": Imap{"bar": "myval"}},
		},
		{
			name: "single key - nonempty map",
			s:    Imap{"foo": Imap{}, "bar": "baz"},
			args: args{keys: []string{"foo"}, value: "myval"},
			want: Imap{"foo": "myval", "bar": "baz"},
		},
		{
			name: "multi key - nonempty map",
			s:    Imap{"foo": Imap{"bar": "otherval"}, "baz": "KEEP"},
			args: args{keys: []string{"foo", "bar"}, value: "myval"},
			want: Imap{"foo": Imap{"bar": "myval"}, "baz": "KEEP"},
		},
		{
			name: "long key - nonempty map",
			s:    Imap{"foo": Imap{"bar": "otherval", "baz": "should keep this"}},
			args: args{keys: []string{"foo", "bar", "baz"}, value: "myval"},
			want: Imap{"foo": Imap{"bar": Imap{"baz": "myval"}, "baz": "should keep this"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.SetIn(tt.args.keys, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Imap.SetIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImap_SetIn_With_Array(t *testing.T) {
	type args struct {
		keys  []string
		value string
	}
	tests := []struct {
		name string
		s    Imap
		args args
		want interface{}
	}{
		{
			name: "append to array",
			s:    Imap{"foo": ImapArray{"bar", "baz"}},
			args: args{keys: []string{"foo", "[+]"}, value: "myval"},
			want: Imap{"foo": ImapArray{"bar", "baz", "myval"}},
		},
		{
			name: "mixed type map elements",
			s:    Imap{"foo": ImapArray{"bar", "baz"}},
			args: args{keys: []string{"foo", "[+]", "bax"}, value: "myval"},
			want: Imap{"foo": ImapArray{"bar", "baz", Imap{"bax": "myval"}}},
		},
		{
			name: "nested array",
			s:    Imap{"foo": ImapArray{"bar", "baz"}},
			args: args{keys: []string{"foo", "[+]", "[+]"}, value: "myval"},
			want: Imap{"foo": ImapArray{"bar", "baz", ImapArray{"myval"}}},
		},
		{
			name: "extending nested array",
			s:    Imap{"foo": ImapArray{"bar", "baz", ImapArray{"myval"}}},
			args: args{keys: []string{"foo", "[-]", "[+]"}, value: "myval2"},
			want: Imap{"foo": ImapArray{"bar", "baz", ImapArray{"myval", "myval2"}}},
		},
		{
			name: "updating map behind array",
			s:    Imap{"events": Imap{"onStart": ImapArray{Imap{"exec": Imap{"command": "foobar"}}}}},
			args: args{keys: []string{"events", "onStart", "[-]", "exec", "key"}, value: "szelcso"},
			want: Imap{"events": Imap{"onStart": ImapArray{Imap{"exec": Imap{"command": "foobar", "key": "szelcso"}}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.SetIn(tt.args.keys, tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Imap.SetIn() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
