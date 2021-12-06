package buildah

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/werf/werf/pkg/buildah/types"
	"github.com/werf/werf/pkg/util"
	"github.com/werf/werf/pkg/werf"
)

const (
	DefaultShmSize              = "65536k"
	BuildahImage                = "ghcr.io/werf/buildah:v1.22.3-1"
	BuildahStorageContainerName = "werf-buildah-storage"

	DefaultStorageDriver StorageDriver = StorageDriverOverlay
)

type CommonOpts struct {
	LogWriter io.Writer
}

type BuildFromDockerfileOpts struct {
	CommonOpts
	ContextTar io.Reader
	BuildArgs  map[string]string
}

type RunMount struct {
	Type        string
	TmpfsSize   string
	Source      string
	Destination string
}

type RunCommandOpts struct {
	CommonOpts
	Args   []string
	Mounts []specs.Mount
}

type RmiOpts struct {
	CommonOpts
	Force bool
}

type (
	FromCommandOpts CommonOpts
	PushOpts        CommonOpts
	PullOpts        CommonOpts
	TagOpts         CommonOpts
	MountOpts       CommonOpts
	UmountOpts      CommonOpts
)

type Buildah interface {
	Tag(ctx context.Context, ref, newRef string, opts TagOpts) error
	Push(ctx context.Context, ref string, opts PushOpts) error
	BuildFromDockerfile(ctx context.Context, dockerfile []byte, opts BuildFromDockerfileOpts) (string, error)
	RunCommand(ctx context.Context, container string, command []string, opts RunCommandOpts) error
	FromCommand(ctx context.Context, container string, image string, opts FromCommandOpts) (string, error)
	Pull(ctx context.Context, ref string, opts PullOpts) error
	Inspect(ctx context.Context, ref string) (*types.BuilderInfo, error)
	Rmi(ctx context.Context, ref string, opts RmiOpts) error
	Mount(ctx context.Context, container string, opts MountOpts) (string, error)
	Umount(ctx context.Context, container string, opts UmountOpts) error
}

type Mode string

const (
	ModeAuto           Mode = "auto"
	ModeNativeRootless Mode = "native-rootless"
	ModeDockerWithFuse Mode = "docker-with-fuse"
)

func ProcessStartupHook(mode Mode) (bool, error) {
	switch ResolveMode(mode) {
	case ModeNativeRootless:
		return NativeRootlessProcessStartupHook(), nil
	case ModeDockerWithFuse:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported mode %q", mode)
	}
}

type StorageDriver string

const (
	StorageDriverOverlay StorageDriver = "overlay"
	StorageDriverVFS     StorageDriver = "vfs"
)

type CommonBuildahOpts struct {
	Isolation     *types.Isolation
	StorageDriver *StorageDriver
	TmpDir        string
	Insecure      bool
}

type NativeRootlessModeOpts struct{}

type DockerWithFuseModeOpts struct{}

type BuildahOpts struct {
	CommonBuildahOpts
	DockerWithFuseModeOpts
	NativeRootlessModeOpts
}

func NewBuildah(mode Mode, opts BuildahOpts) (b Buildah, err error) {
	if opts.CommonBuildahOpts.Isolation == nil {
		defIsolation, err := GetDefaultIsolation()
		if err != nil {
			return b, fmt.Errorf("unable to determine default isolation: %s", err)
		}
		opts.CommonBuildahOpts.Isolation = &defIsolation
	}

	if opts.CommonBuildahOpts.StorageDriver == nil {
		defStorageDriver := DefaultStorageDriver
		opts.CommonBuildahOpts.StorageDriver = &defStorageDriver
	}

	if opts.CommonBuildahOpts.TmpDir == "" {
		opts.CommonBuildahOpts.TmpDir = filepath.Join(werf.GetHomeDir(), "buildah", "tmp")
	}

	switch ResolveMode(mode) {
	case ModeNativeRootless:
		switch runtime.GOOS {
		case "linux":
			b, err = NewNativeRootlessBuildah(opts.CommonBuildahOpts, opts.NativeRootlessModeOpts)
			if err != nil {
				return nil, fmt.Errorf("unable to create new Buildah instance with mode %q: %s", mode, err)
			}
		default:
			panic("ModeNativeRootless can't be used on this OS")
		}
	case ModeDockerWithFuse:
		b, err = NewDockerWithFuseBuildah(opts.CommonBuildahOpts, opts.DockerWithFuseModeOpts)
		if err != nil {
			return nil, fmt.Errorf("unable to create new Buildah instance with mode %q: %s", mode, err)
		}
	default:
		return nil, fmt.Errorf("unsupported mode %q", mode)
	}

	return b, nil
}

func ResolveMode(mode Mode) Mode {
	switch mode {
	case ModeAuto:
		switch runtime.GOOS {
		case "linux":
			return ModeNativeRootless
		default:
			return ModeDockerWithFuse
		}
	default:
		return mode
	}
}

func GetOverlayOptions() ([]string, error) {
	fuseOverlayBinPath, err := exec.LookPath("fuse-overlayfs")
	if err != nil {
		return nil, fmt.Errorf("\"fuse-overlayfs\" binary not found in PATH: %s", err)
	}

	result := []string{fmt.Sprintf("overlay.mount_program=%s", fuseOverlayBinPath)}

	if isInContainer, err := util.IsInContainer(); err != nil {
		return nil, fmt.Errorf("unable to determine whether we are in the container: %s", err)
	} else if isInContainer {
		result = append(result, fmt.Sprintf("overlay.mountopt=%s", "nodev,fsync=0"))
	}

	return result, nil
}

func GetDefaultIsolation() (types.Isolation, error) {
	if isInContainer, err := util.IsInContainer(); err != nil {
		return 0, fmt.Errorf("unable to determine if is in container: %s", err)
	} else if isInContainer {
		return types.IsolationChroot, nil
	} else {
		return types.IsolationOCIRootless, nil
	}
}

func debug() bool {
	return os.Getenv("WERF_BUILDAH_DEBUG") == "1"
}
