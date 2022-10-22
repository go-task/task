package exporter

import (
	"reflect"
	"testing"
)

func TestUnmarshalTypes(t *testing.T) {
	type args struct {
		values []string
	}
	tests := []struct {
		name    string
		args    args
		want    *[]Type
		wantErr bool
	}{
		{name: "validTypes", args: args{values: []string{"github_actions"}}, want: &AllowedExporterTypeEnumValues},
		{name: "invalidTypes", args: args{[]string{"whatever"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalTypes(tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalTypes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalTypes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unmarshalType(t *testing.T) {
	validType := GithubActions
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    *Type
		wantErr bool
	}{
		{name: "validType", args: args{"github_actions"}, want: &validType},
		{name: "invalidType", args: args{"whatever"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalType(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshalType() got = %v, want %v", got, tt.want)
			}
		})
	}
}
