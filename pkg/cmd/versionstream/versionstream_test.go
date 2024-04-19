package versionstream

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"

	"github.com/stretchr/testify/assert"
)

func TestMoreThanOneFlagSet(t *testing.T) {
	o := Options{}
	assert.False(t, o.moreThanOneFlagSet())

	o.LTS = true
	assert.False(t, o.moreThanOneFlagSet())

	o.Latest = true
	assert.True(t, o.moreThanOneFlagSet())

	o.Custom = true
	assert.True(t, o.moreThanOneFlagSet())

	o.Latest = false
	assert.True(t, o.moreThanOneFlagSet())

	o.Custom = false
	assert.False(t, o.moreThanOneFlagSet())
}

func TestAtLeastOneFlagSet(t *testing.T) {
	o := Options{}

	assert.False(t, o.atLeastOneFlagSet())

	o.LTS = true
	assert.True(t, o.atLeastOneFlagSet())

	o.Latest = true
	assert.True(t, o.atLeastOneFlagSet())

	o.Custom = true
	assert.True(t, o.atLeastOneFlagSet())
}

func TestOptions_Run(t *testing.T) {
	type fields struct {
		LTS    bool
		Latest bool
		Custom bool
		GitURL string
		GitRef string
		GitDir string
		Dir    string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "lts", fields: fields{LTS: true}, wantErr: false},
		{name: "latest", fields: fields{Latest: true}, wantErr: false},
		{name: "custom", fields: fields{Custom: true, GitURL: "https://github.com/foo/bar", GitRef: "cheese", GitDir: "wine"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			err := files.CopyDir(filepath.Join("testdata", tt.name), tmpDir, true)
			assert.NoError(t, err)

			o := &Options{
				LTS:    tt.fields.LTS,
				Latest: tt.fields.Latest,
				Custom: tt.fields.Custom,
				GitURL: tt.fields.GitURL,
				GitRef: tt.fields.GitRef,
				GitDir: tt.fields.GitDir,
				Dir:    tmpDir,
			}

			if err := o.switchVersionStream(); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				testhelpers.AssertTextFilesEqual(t, filepath.Join("testdata", tt.name, "expected"), filepath.Join(tmpDir, "versionStream", "Kptfile"), "updated Kptfile does not match expected")
			}
		})
	}
}
