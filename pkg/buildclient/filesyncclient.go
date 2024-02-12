package buildclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/acorn-io/function-builder/pkg/factory"
	"github.com/acorn-io/function-builder/pkg/templates"
	"github.com/moby/buildkit/session/filesync"
	"github.com/sirupsen/logrus"
	"github.com/tonistiigi/fsutil"
	"github.com/tonistiigi/fsutil/types"
	fstypes "github.com/tonistiigi/fsutil/types"
	"google.golang.org/grpc/metadata"
)

var (
	ignoreLocalFiles = map[string]bool{
		".dockerignore": true,
	}
)

type fileSyncClient struct {
	compress  bool
	sessionID string
	messages  Messages
	msg       <-chan *Message
	close     func()
	tempDir   string
	ctx       context.Context
}

func newFileSyncClient(ctx context.Context, cwd, sessionID string, messages Messages, opts *SyncOptions) (*fileSyncClient, error) {
	if opts == nil {
		return nil, fmt.Errorf("options can not be nil")
	}

	tempDir, dirs, err := createFileMapInput(ctx, cwd, opts)
	if err != nil {
		return nil, err
	}

	synced, err := prepareSyncedDirs(dirs, opts.DirName, opts.FollowPaths)
	if err != nil {
		return nil, err
	}

	md := metadata.MD{
		keyOverrideExcludes:   opts.OverrideExcludes,
		keyIncludePatterns:    opts.IncludePatterns,
		keyExcludePatterns:    opts.ExcludePatterns,
		keyFollowPaths:        opts.FollowPaths,
		keyDirName:            opts.DirName,
		keyExporterMetaPrefix: opts.ExporterMetaPrefix,
	}

	logrus.Tracef("starting file sync client %s, compress=%v", sessionID, opts.Compress)
	fsClient := &fileSyncClient{
		compress:  opts.Compress,
		sessionID: sessionID,
		messages:  messages,
		tempDir:   tempDir,
		ctx:       metadata.NewIncomingContext(ctx, md),
	}
	fsClient.msg, fsClient.close = messages.Recv()

	server := filesync.NewFSSyncProvider(synced).(filesync.FileSyncServer)
	go func() {
		defer logrus.Tracef("closed file sync client %s", sessionID)
		defer fsClient.Close()
		err := server.DiffCopy(fsClient)
		if err != nil {
			logrus.Errorf("file sync failed: %T", err)
			messages.Close()
		}
		_ = messages.Send(&Message{
			FileSessionID:    sessionID,
			FileSessionClose: true,
		})
	}()
	return fsClient, nil
}

func createFileMapInput(ctx context.Context, cwd string, opts *SyncOptions) (string, map[string]string, error) {
	var (
		tempDir    string
		err        error
		context    = filepath.Join(cwd, opts.Context)
		dockerfile = filepath.Dir(filepath.Join(cwd, opts.Dockerfile))
		dirs       = map[string]string{}
	)

	if strings.HasSuffix(opts.Dockerfile, templates.Suffix) {
		data, err := factory.Load(ctx, filepath.Join(cwd, opts.Dockerfile))
		if err != nil {
			return "", nil, err
		}
		tempDir, err = os.MkdirTemp("", "acorn")
		if err != nil {
			return "", nil, err
		}
		dockerfile = tempDir
		if err := os.MkdirAll(filepath.Dir(filepath.Join(tempDir, opts.Dockerfile)), 0655); err != nil {
			return "", nil, err
		}
		err = os.WriteFile(filepath.Join(dockerfile, filepath.Base(opts.Dockerfile)), data, 0600)
		if err != nil {
			return "", nil, err
		}
	} else if opts.DockerfileContents != "" {
		tempDir, err = os.MkdirTemp("", "acorn")
		if err != nil {
			return "", nil, err
		}
		dockerfile = tempDir
		err := os.WriteFile(filepath.Join(dockerfile, "Dockerfile"), []byte(opts.DockerfileContents), 0600)
		if err != nil {
			return "", nil, err
		}
	}

	if len(opts.DirName) > 0 {
		if opts.DirName[0] == "context" {
			dirs["context"] = context
		} else if dir, ok := opts.AdditionalContexts[opts.DirName[0]]; ok {
			dirs[opts.DirName[0]] = filepath.Join(cwd, dir)
		}
	}

	dirs["dockerfile"] = dockerfile
	return tempDir, dirs, nil
}

func prepareSyncedDirs(localDirs map[string]string, dirNames []string, followPaths []string) (filesync.StaticDirSource, error) {
	for localDirName, d := range localDirs {
		fi, err := os.Stat(d)
		if os.IsNotExist(err) {
			// don't blindly mkdirall because this could actually be a file
			err := os.MkdirAll(d, 0755)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, fmt.Errorf("could not find %s: %w", d, err)
		} else if !fi.IsDir() {
			return nil, fmt.Errorf("%s not a directory", d)
		}
		for _, dirName := range dirNames {
			if dirName != "dockerfile" && localDirName == dirName {
				for _, followPath := range followPaths {
					if ignoreLocalFiles[followPath] {
						continue
					}
					f := filepath.Join(d, followPath)
					if _, err := os.Stat(f); os.IsNotExist(err) {
						if strings.Contains(f, "*") || strings.Contains(f, "?") {
							err = nil
						} else {
							err = os.MkdirAll(f, 0755)
						}
						if err != nil {
							return nil, err
						}
					} else if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	resetUIDAndGID := func(_ string, st *fstypes.Stat) fsutil.MapResult {
		st.Uid = 0
		st.Gid = 0
		return fsutil.MapResultKeep
	}

	dirs := make(filesync.StaticDirSource, len(localDirs))
	for name, d := range localDirs {
		dirs[name] = filesync.SyncedDir{Dir: d, Map: resetUIDAndGID}
	}

	return dirs, nil
}

func (s *fileSyncClient) Send(obj *types.Packet) error {
	return s.SendMsg(obj)
}

func (s *fileSyncClient) Recv() (*types.Packet, error) {
	obj := &types.Packet{}
	return obj, s.RecvMsg(obj)
}

func (s *fileSyncClient) SetHeader(metadata.MD) error {
	panic("not implemented")
}

func (s *fileSyncClient) SendHeader(metadata.MD) error {
	panic("not implemented")
}

func (s *fileSyncClient) SetTrailer(metadata.MD) {
	panic("not implemented")
}

func (s *fileSyncClient) Context() context.Context {
	return s.ctx
}

func (s *fileSyncClient) Close() {
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
	s.close()
}

func (s *fileSyncClient) SendMsg(m interface{}) error {
	return s.messages.Send(&Message{
		Compress:      s.compress,
		FileSessionID: s.sessionID,
		Packet:        m.(*types.Packet),
	})
}

func (s *fileSyncClient) RecvMsg(m interface{}) error {
	for {
		nextMessage, ok := <-s.msg
		if !ok {
			return io.EOF
		}
		if nextMessage.Packet == nil || nextMessage.FileSessionID != s.sessionID {
			continue
		}
		logrus.Tracef("fileSyncClient msg.fileSessionID=%s sessionID=%s packetNil=%v", nextMessage.FileSessionID, s.sessionID, nextMessage.Packet == nil)
		n := m.(*types.Packet)
		*n = *nextMessage.Packet
		return nil
	}
}
