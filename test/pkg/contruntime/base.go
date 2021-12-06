package contruntime

import (
	"github.com/google/uuid"
)

type BaseContainerRuntime struct {
	CommonCliArgs []string
}

func expectCmdsToSucceed(r ContainerRuntime, image string, cmds ...string) {
	containerName := uuid.New().String()
	r.RunSleepingContainer(containerName, image)
	r.Exec(containerName, cmds...)
	r.Rm(containerName)
}
