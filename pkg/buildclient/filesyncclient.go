package buildclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/session/filesync"
	"github.com/sirupsen/logrus"
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

	tempDir, dirs, err := createFileMapInput(cwd, opts)
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

	logrus.Tracef("starting file sync client %s", sessionID)
	fsClient := &fileSyncClient{
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

func createFileMapInput(cwd string, opts *SyncOptions) (string, map[string]string, error) {
	var (
		tempDir    string
		err        error
		context    = filepath.Join(cwd, opts.Context)
		dockerfile = filepath.Dir(filepath.Join(cwd, opts.Dockerfile))
	)
	if opts.DockerfileContents != "" {
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
	return tempDir, map[string]string{
		"context":    context,
		"dockerfile": dockerfile,
	}, nil
}

func prepareSyncedDirs(localDirs map[string]string, dirNames []string, followPaths []string) ([]filesync.SyncedDir, error) {
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
			if dirName == "context" && localDirName == dirName {
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
	resetUIDAndGID := func(p string, st *fstypes.Stat) bool {
		st.Uid = 0
		st.Gid = 0
		return true
	}

	dirs := make([]filesync.SyncedDir, 0, len(localDirs))
	for name, d := range localDirs {
		dirs = append(dirs, filesync.SyncedDir{Name: name, Dir: d, Map: resetUIDAndGID})
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
		logrus.Tracef("fileSyncClient msg.fileSessionID=%s sessionID=%s packetNil=%v", nextMessage.FileSessionID, s.sessionID, nextMessage.Packet == nil)
		if nextMessage.Packet == nil || nextMessage.FileSessionID != s.sessionID {
			continue
		}
		n := m.(*types.Packet)
		*n = *nextMessage.Packet
		return nil
	}
}
