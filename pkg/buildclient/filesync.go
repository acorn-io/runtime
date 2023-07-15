package buildclient

import (
	"github.com/google/uuid"
	"github.com/moby/buildkit/session/filesync"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	keyOverrideExcludes   = "override-excludes"
	keyIncludePatterns    = "include-patterns"
	keyExcludePatterns    = "exclude-patterns"
	keyFollowPaths        = "followpaths"
	keyDirName            = "dir-name"
	keyExporterMetaPrefix = "exporter-md-"
)

type FileServer struct {
	messages           Messages
	context            string
	additionalContexts map[string]string
	dockerfilePath     string
	dockerfileContents string
}

func NewFileServer(messages Messages, context string, additionalContexts map[string]string, dockerfilePath, dockerFileContents string) *FileServer {
	return &FileServer{
		messages:           messages,
		context:            context,
		additionalContexts: additionalContexts,
		dockerfilePath:     dockerfilePath,
		dockerfileContents: dockerFileContents,
	}
}

func (f *FileServer) DiffCopy(server filesync.FileSync_DiffCopyServer) error {
	sessionID := uuid.New().String()
	logrus.Tracef("Starting diffcopy [%s]", sessionID)
	defer logrus.Tracef("Finished diffcopy [%s]", sessionID)

	ctx, _ := metadata.FromIncomingContext(server.Context())

	// subscribe early to not miss any messages
	msgs, cancel := f.messages.Recv()
	defer cancel()

	err := f.messages.Send(&Message{
		FileSessionID: sessionID,
		SyncOptions: &SyncOptions{
			Compress:           true,
			Context:            f.context,
			AdditionalContexts: f.additionalContexts,
			Dockerfile:         f.dockerfilePath,
			DockerfileContents: f.dockerfileContents,
			OverrideExcludes:   ctx.Get(keyOverrideExcludes),
			IncludePatterns:    ctx.Get(keyIncludePatterns),
			ExcludePatterns:    ctx.Get(keyExcludePatterns),
			FollowPaths:        ctx.Get(keyFollowPaths),
			DirName:            ctx.Get(keyDirName),
			ExporterMetaPrefix: ctx.Get(keyExporterMetaPrefix),
		},
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := server.Recv()
			if err != nil {
				break
			}
			_ = f.messages.Send(&Message{
				FileSessionID: sessionID,
				Packet:        msg,
			})
		}
	}()

	for msg := range msgs {
		if msg.FileSessionID == sessionID {
			logrus.Tracef("file sync message msg.FileSessionID=%s sessionID=%s close=%v", msg.FileSessionID, sessionID, msg.FileSessionClose)
			if msg.FileSessionClose {
				cancel()
			} else {
				_ = server.Send(msg.Packet)
			}
		}
	}

	return nil
}

func (f *FileServer) TarStream(server filesync.FileSync_TarStreamServer) error {
	panic("unsupported")
}

func (f *FileServer) Register(server *grpc.Server) {
	filesync.RegisterFileSyncServer(server, f)
}
