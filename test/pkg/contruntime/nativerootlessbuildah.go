package contruntime

import (
	"encoding/json"

	. "github.com/onsi/gomega"
	"github.com/werf/werf/pkg/buildah"

	"github.com/werf/werf/integration/pkg/utils"
	"github.com/werf/werf/test/pkg/thirdparty/contruntime/manifest"
)

func NewNativeRootlessBuildahRuntime(storageDriver buildah.StorageDriver) ContainerRuntime {
	var commonCliArgs []string

	commonBuildahCliArgs, err := buildah.GetCommonBuildahCliArgs(storageDriver)
	Expect(err).NotTo(HaveOccurred())

	commonCliArgs = append(commonCliArgs, commonBuildahCliArgs...)

	return &NativeRootlessBuildahRuntime{
		BaseContainerRuntime: BaseContainerRuntime{
			CommonCliArgs: commonCliArgs,
		},
	}
}

type NativeRootlessBuildahRuntime struct {
	BaseContainerRuntime
}

func (r *NativeRootlessBuildahRuntime) ExpectCmdsToSucceed(image string, cmds ...string) {
	expectCmdsToSucceed(r, image, cmds...)
}

func (r *NativeRootlessBuildahRuntime) RunSleepingContainer(containerName, image string) {
	args := append(r.CommonCliArgs, "from", "--tls-verify=false", "--format", "docker", "--name", containerName, image)
	utils.RunSucceedCommand("/", "buildah", args...)
}

func (r *NativeRootlessBuildahRuntime) Exec(containerName string, cmds ...string) {
	for _, cmd := range cmds {
		args := append(r.CommonCliArgs, "run", containerName, "--", "sh", "-ec", cmd)
		utils.RunSucceedCommand("/", "buildah", args...)
	}
}

func (r *NativeRootlessBuildahRuntime) Rm(containerName string) {
	args := append(r.CommonCliArgs, "rm", containerName)
	utils.RunSucceedCommand("/", "buildah", args...)
}

func (r *NativeRootlessBuildahRuntime) Pull(image string) {
	args := append(r.CommonCliArgs, "pull", "--tls-verify=false", image)
	utils.RunSucceedCommand("/", "buildah", args...)
}

func (r *NativeRootlessBuildahRuntime) GetImageInspectConfig(image string) (config manifest.Schema2Config) {
	r.Pull(image)

	args := append(r.CommonCliArgs, "inspect", "--type", "image", image)
	inspectRaw, err := utils.RunCommand("/", "buildah", args...)
	Expect(err).NotTo(HaveOccurred())

	var inspect BuildahInspect
	Expect(json.Unmarshal(inspectRaw, &inspect)).To(Succeed())

	return inspect.Docker.Config
}
