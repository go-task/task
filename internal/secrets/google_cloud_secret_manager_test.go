package secrets

import (
	"testing"
)

func TestGoogleCloudSecretManager_prepareReference(t *testing.T) {
	type fields struct {
		defaultProject string
		defaultVersion string
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "emptyReferenceInValid",
			args:    args{""},
			wantErr: true,
		},
		{
			name:    "referenceInValid",
			args:    args{"foo/bar"},
			wantErr: true,
		},
		{
			name: "shortReferenceValid",
			fields: fields{
				defaultProject: "my-project",
				defaultVersion: "my-version",
			},
			args: args{"test-secret"},
			want: "projects/my-project/secrets/test-secret/versions/my-version",
		},
		{
			name:    "shortReferenceInValid",
			args:    args{"test-secret"},
			wantErr: true,
		},
		{
			name:   "mediumReferenceValid",
			fields: fields{defaultVersion: "default-version"},
			args:   args{"projects/my-project/secrets/test-secret"},
			want:   "projects/my-project/secrets/test-secret/versions/default-version",
		},
		{
			name: "longReferenceValid",
			args: args{"projects/my-project/secrets/test-secret/versions/my-version"},
			want: "projects/my-project/secrets/test-secret/versions/my-version",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GoogleCloudSecretManager{
				defaultProject: tt.fields.defaultProject,
				defaultVersion: tt.fields.defaultVersion,
			}
			got, err := m.prepareReference(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("prepareReference() got = %v, want %v", got, tt.want)
			}
		})
	}
}
