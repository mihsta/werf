package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/werf/werf/pkg/buildah"
	"github.com/werf/werf/pkg/buildah/types"
	"github.com/werf/werf/pkg/container_runtime"
	"github.com/werf/werf/pkg/docker"
	"github.com/werf/werf/pkg/util"
)

func ContainerRuntimeProcessStartupHook() (bool, error) {
	buildahMode := GetContainerRuntimeBuildahMode()

	switch {
	case buildahMode != "":
		return buildah.ProcessStartupHook(buildahMode)
	case strings.HasPrefix(os.Args[0], "buildah-") || strings.HasPrefix(os.Args[0], "chrootuser-") || strings.HasPrefix(os.Args[0], "storage-"):
		return buildah.ProcessStartupHook("native-rootless")
	}

	return false, nil
}

func GetContainerRuntimeBuildahMode() buildah.Mode {
	return buildah.Mode(os.Getenv("WERF_CONTAINER_RUNTIME_BUILDAH"))
}

func GetContainerRuntimeBuildahIsolation() (*types.Isolation, error) {
	isolationRaw := os.Getenv("WERF_CONTAINER_RUNTIME_BUILDAH_ISOLATION")
	var isolation types.Isolation
	switch isolationRaw {
	case "rootless", "oci-rootless":
		if isInContainer, err := util.IsInContainer(); err != nil {
			return nil, fmt.Errorf("unable to determine if is in container: %s", err)
		} else if isInContainer {
			return nil, fmt.Errorf("rootless isolation is not available in container: %s", err)
		}
		isolation = types.IsolationOCIRootless
	case "chroot":
		isolation = types.IsolationChroot
	case "default", "":
		var err error
		isolation, err = buildah.GetDefaultIsolation()
		if err != nil {
			return nil, fmt.Errorf("unable to determine default isolation: %s", err)
		}
	default:
		return nil, fmt.Errorf("unexpected isolation specified: %s", isolationRaw)
	}
	return &isolation, nil
}

func GetContainerRuntimeBuildahStorageDriver() (*buildah.StorageDriver, error) {
	storageDriverRaw := os.Getenv("WERF_CONTAINER_RUNTIME_BUILDAH_STORAGE_DRIVER")
	var storageDriver buildah.StorageDriver
	switch storageDriverRaw {
	case string(buildah.StorageDriverOverlay), string(buildah.StorageDriverVFS):
		storageDriver = buildah.StorageDriver(storageDriverRaw)
	case "default", "":
		storageDriver = buildah.DefaultStorageDriver
	default:
		return nil, fmt.Errorf("unexpected driver specified: %s", storageDriverRaw)
	}
	return &storageDriver, nil
}

func wrapContainerRuntime(containerRuntime container_runtime.ContainerRuntime) container_runtime.ContainerRuntime {
	if os.Getenv("WERF_PERF_TEST_CONTAINER_RUNTIME") == "1" {
		return container_runtime.NewPerfCheckContainerRuntime(containerRuntime)
	}
	return containerRuntime
}

func InitProcessContainerRuntime(ctx context.Context, cmdData *CmdData) (container_runtime.ContainerRuntime, context.Context, error) {
	buildahMode := GetContainerRuntimeBuildahMode()
	if buildahMode != "" {
		resolvedMode := buildah.ResolveMode(buildahMode)
		if resolvedMode == buildah.ModeDockerWithFuse {
			newCtx, err := InitProcessDocker(ctx, cmdData)
			if err != nil {
				return nil, ctx, fmt.Errorf("unable to init process docker for buildah container runtime: %s", err)
			}
			ctx = newCtx
		}

		isolation, err := GetContainerRuntimeBuildahIsolation()
		if err != nil {
			return nil, ctx, fmt.Errorf("unable to determine buildah container runtime isolation: %s", err)
		}

		storageDriver, err := GetContainerRuntimeBuildahStorageDriver()
		if err != nil {
			return nil, ctx, fmt.Errorf("unable to determine buildah container runtime storage driver: %s", err)
		}

		insecure := *cmdData.InsecureRegistry || *cmdData.SkipTlsVerifyRegistry
		b, err := buildah.NewBuildah(resolvedMode, buildah.BuildahOpts{
			CommonBuildahOpts: buildah.CommonBuildahOpts{
				Insecure:      insecure,
				Isolation:     isolation,
				StorageDriver: storageDriver,
			},
		})
		if err != nil {
			return nil, ctx, fmt.Errorf("unable to get buildah client: %s", err)
		}

		return wrapContainerRuntime(container_runtime.NewBuildahRuntime(b)), ctx, nil
	}

	newCtx, err := InitProcessDocker(ctx, cmdData)
	if err != nil {
		return nil, ctx, fmt.Errorf("unable to init process docker for docker-server container runtime: %s", err)
	}
	ctx = newCtx

	return wrapContainerRuntime(container_runtime.NewDockerServerRuntime()), ctx, nil
}

func InitProcessDocker(ctx context.Context, cmdData *CmdData) (context.Context, error) {
	if err := docker.Init(ctx, *cmdData.DockerConfig, *cmdData.LogVerbose, *cmdData.LogDebug, *cmdData.Platform); err != nil {
		return ctx, fmt.Errorf("unable to init docker for buildah container runtime: %s", err)
	}

	ctxWithDockerCli, err := docker.NewContext(ctx)
	if err != nil {
		return ctx, fmt.Errorf("unable to init context for docker: %s", err)
	}
	ctx = ctxWithDockerCli

	return ctx, nil
}
