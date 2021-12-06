package contruntime

import (
	"encoding/json"

	. "github.com/onsi/gomega"

	"github.com/werf/werf/integration/pkg/utils"
	"github.com/werf/werf/test/pkg/thirdparty/contruntime/manifest"
)

func NewDockerRuntime() ContainerRuntime {
	return &DockerRuntime{}
}

type DockerRuntime struct {
	BaseContainerRuntime
}

func (r *DockerRuntime) ExpectCmdsToSucceed(image string, cmds ...string) {
	expectCmdsToSucceed(r, image, cmds...)
}

func (r *DockerRuntime) RunSleepingContainer(containerName, image string) {
	args := append(r.CommonCliArgs, "run", "--rm", "-d", "--entrypoint=", "--name", containerName, image, "tail", "-f", "/dev/null")
	utils.RunSucceedCommand("/", "docker", args...)
}

func (r *DockerRuntime) Exec(containerName string, cmds ...string) {
	for _, cmd := range cmds {
		args := append(r.CommonCliArgs, "exec", containerName, "sh", "-ec", cmd)
		utils.RunSucceedCommand("/", "docker", args...)
	}
}

func (r *DockerRuntime) Rm(containerName string) {
	args := append(r.CommonCliArgs, "rm", "-fv", containerName)
	utils.RunSucceedCommand("/", "docker", args...)
}

func (r *DockerRuntime) Pull(image string) {
	args := append(r.CommonCliArgs, "pull", image)
	utils.RunSucceedCommand("/", "docker", args...)
}

func (r *DockerRuntime) GetImageInspectConfig(image string) (config manifest.Schema2Config) {
	args := append(r.CommonCliArgs, "image", "inspect", "-f", "{{ json .Config }}", image)
	configRaw, err := utils.RunCommand("/", "docker", args...)
	Expect(err).NotTo(HaveOccurred())
	Expect(json.Unmarshal(configRaw, &config)).To(Succeed())

	return config
}
