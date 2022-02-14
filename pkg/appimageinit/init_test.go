package appimageinit

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrint(t *testing.T) {
	tests := []struct {
		pathName   string
		wantOutput string
		wantErr    bool
	}{
		{
			pathName: "onlyherd",
			wantOutput: `{"id":"onlyherd","herdfile":"containers:{}","imageData":{}}
`,
		},
		{
			pathName: "herdandimages",
			wantOutput: `{"id":"herdandimages","herdfile":"containers:{}","imageData":{"containers":{"foo":{"image":"image-name"}}}}
`,
		},
		{
			pathName:   "missingherd",
			wantErr:    true,
			wantOutput: "{\"error\":\"herd.cue is missing in app image: missingherd\"}\n",
		},
		{
			pathName:   "badimages",
			wantErr:    true,
			wantOutput: "{\"error\":\"decoding images.json in badimages: unexpected EOF\"}\n",
		},
		{
			pathName: "badherd",
			wantOutput: `{"id":"badherd","herdfile":"containers:{","imageData":{"containers":{"foo":{"image":"image-name"}}}}
`,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.pathName, func(t *testing.T) {
			output := &bytes.Buffer{}
			err := Print(filepath.Join("./testdata", tt.pathName), tt.pathName, output)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.wantOutput, output.String())
		})
	}
}
