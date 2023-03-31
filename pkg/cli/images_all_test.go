package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/mocks"
	"github.com/acorn-io/acorn/pkg/project"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestImages(t *testing.T) {
	image1Untagged := apiv1.Image{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-1-9102390",
			Namespace: "NameOne",
			Labels: map[string]string{
				labels.AcornProject: "project1",
			},
		},
		Digest: "1234567890asdfghkl",
	}
	image1Tagged := image1Untagged
	image1Tagged.Tags = []string{"testtag:latest"}

	image2Untagged := apiv1.Image{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-2-2198031",
			Namespace: "NameTwo",
			Labels: map[string]string{
				labels.AcornProject: "project2",
			},
		},
	}

	image2Tagged := image2Untagged
	image2Tagged.Tags = []string{"testtag2:latest2"}

	image3Untagged := apiv1.Image{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-3-129394",
			Namespace: "Name3",
			Labels: map[string]string{
				labels.AcornProject: "acorn.com/acc/Name3",
			},
		},
	}

	image3Tagged := image3Untagged
	image3Tagged.Tags = []string{"testtag3:latest3"}

	allImagesHaveContaines := func(f *mocks.MockClient) {
		f.EXPECT().ImageDetails(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, imageName string, opts *client.ImageDetailsOptions) (*client.ImageDetails, error) {
			return &client.ImageDetails{
				AppImage: v1.AppImage{ID: imageName, ImageData: v1.ImagesData{
					Containers: map[string]v1.ContainerData{"test-image-running-container": {
						Image: "test-image-running-container",
					}},
				}},
			}, nil
		}).AnyTimes()
	}

	type fields struct {
		Quiet  bool
		Output string
		All    bool
	}
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	testCases := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		wantOut string
		prepare func(f *mocks.MockClient)
	}{
		{
			name: "acorn images -a -A, one untagged", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Tagged,
						image2Untagged,
					}, nil)
			},
			args: args{
				args: []string{"-a"},
			},
			wantErr: false,
			wantOut: "REPOSITORY          TAG       IMAGE-ID\nproject1::testtag   latest    image-1-9102\n<none>              <none>    image-2-2198\n",
		},
		{
			name: "acorn images -a -A, both tagged", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Tagged,
						image2Tagged,
					}, nil)
			},
			args: args{
				args: []string{"-a"},
			},
			wantErr: false,
			wantOut: "REPOSITORY           TAG       IMAGE-ID\nproject1::testtag    latest    image-1-9102\nproject2::testtag2   latest2   image-2-2198\n",
		},
		{
			name: "acorn images -A -c, all tagged", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Tagged,
						image2Tagged,
						image3Tagged,
					}, nil)
				// Any imageDetails call specifies that the image has a container
				allImagesHaveContaines(f)
			},
			args: args{
				args: []string{"-c"},
			},
			wantErr: false,
			wantOut: "REPOSITORY                      TAG       IMAGE-ID          CONTAINER                      DIGEST\nproject1::testtag               latest    image-1-9102390   test-image-running-container   test-image-running-container\nproject2::testtag2              latest2   image-2-2198031   test-image-running-container   test-image-running-container\nacorn.com/acc/Name3::testtag3   latest3   image-3-129394    test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn images -A -c, one tagged, both have containers", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Tagged,
						image2Untagged,
					}, nil)
				allImagesHaveContaines(f)
			},
			args: args{
				args: []string{"-c"},
			},
			wantErr: false,
			wantOut: "REPOSITORY          TAG       IMAGE-ID          CONTAINER                      DIGEST\nproject1::testtag   latest    image-1-9102390   test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn images -A -c -a, one tagged", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Tagged,
						image2Untagged,
					}, nil)
				allImagesHaveContaines(f)
			},
			args: args{
				args: []string{"-c", "-a"},
			},
			wantErr: false,
			wantOut: "REPOSITORY          TAG       IMAGE-ID          CONTAINER                      DIGEST\nproject1::testtag   latest    image-1-9102390   test-image-running-container   test-image-running-container\n<none>              <none>    image-2-2198031   test-image-running-container   test-image-running-container\n",
		},
		{
			name: "acorn images -A -c -a, no tags", fields: fields{
				All:    true,
				Quiet:  false,
				Output: "",
			},
			prepare: func(f *mocks.MockClient) {
				f.EXPECT().ImageList(gomock.Any()).Return(
					[]apiv1.Image{
						image1Untagged,
					}, nil)
				allImagesHaveContaines(f)
			},
			args: args{
				args: []string{"-c", "-a"},
			},
			wantErr: false,
			wantOut: "REPOSITORY   TAG       IMAGE-ID          CONTAINER                      DIGEST\n<none>       <none>    image-1-9102390   test-image-running-container   test-image-running-container\n",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			//Mocked client for cli's client calls.
			mClient := mocks.NewMockClient(ctrl)
			if tt.prepare != nil {
				tt.prepare(mClient)
			}

			r, w, _ := os.Pipe()
			os.Stdout = w
			// Mock client factory just returns the gomock client.
			command := CommandContext{
				ClientFactory: &testdata.MockClientFactoryManual{
					Client:         mClient,
					ProjectOptions: project.Options{AllProjects: tt.fields.All},
				},
				StdOut: w,
				StdErr: w,
				StdIn:  strings.NewReader(""),
			}
			tt.args.cmd = NewImage(command)
			tt.args.cmd.SetArgs(tt.args.args)

			err := tt.args.cmd.Execute()

			if err != nil && !tt.wantErr {
				assert.Failf(t, "got err when err not expected", "got err: %s", err.Error())
			} else if err != nil && tt.wantErr {
				assert.Equal(t, tt.wantOut, err.Error())
			} else {
				err := w.Close()
				if err != nil {
					t.Fatal(err)
				}
				out, _ := io.ReadAll(r)
				assert.Equal(t, tt.wantOut, string(out))
			}
		})
	}
}
