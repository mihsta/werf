package buildah

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/werf/werf/pkg/buildah/types"
	"github.com/werf/werf/pkg/util"
)

type BaseBuildah struct {
	Isolation           types.Isolation
	TmpDir              string
	InstanceTmpDir      string
	SignaturePolicyPath string
	Insecure            bool
}

type BaseBuildahOpts struct {
	Isolation types.Isolation
	Insecure  bool
}

func NewBaseBuildah(tmpDir string, opts BaseBuildahOpts) (*BaseBuildah, error) {
	b := &BaseBuildah{
		Isolation: opts.Isolation,
		TmpDir:    tmpDir,
		Insecure:  opts.Insecure,
	}

	if err := os.MkdirAll(b.TmpDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create dir %q: %s", b.TmpDir, err)
	}

	var err error
	b.InstanceTmpDir, err = ioutil.TempDir(b.TmpDir, "instance")
	if err != nil {
		return nil, fmt.Errorf("unable to create instance tmp dir: %s", err)
	}

	b.SignaturePolicyPath = filepath.Join(b.InstanceTmpDir, "policy.json")
	if err := ioutil.WriteFile(b.SignaturePolicyPath, []byte(DefaultSignaturePolicy), os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to write file %q: %s", b.SignaturePolicyPath, err)
	}

	return b, nil
}

func (b *BaseBuildah) NewSessionTmpDir() (string, error) {
	sessionTmpDir, err := ioutil.TempDir(b.TmpDir, "session")
	if err != nil {
		return "", fmt.Errorf("unable to create session tmp dir: %s", err)
	}

	return sessionTmpDir, nil
}

func (b *BaseBuildah) prepareBuildFromDockerfile(dockerfile []byte, contextTar io.Reader) (string, string, string, error) {
	sessionTmpDir, err := b.NewSessionTmpDir()
	if err != nil {
		return "", "", "", err
	}

	dockerfileTmpPath := filepath.Join(sessionTmpDir, "Dockerfile")
	if err := ioutil.WriteFile(dockerfileTmpPath, dockerfile, os.ModePerm); err != nil {
		return "", "", "", fmt.Errorf("error writing %q: %s", dockerfileTmpPath, err)
	}

	contextTmpDir := filepath.Join(sessionTmpDir, "context")
	if err := os.MkdirAll(contextTmpDir, os.ModePerm); err != nil {
		return "", "", "", fmt.Errorf("unable to create dir %q: %s", contextTmpDir, err)
	}

	if contextTar != nil {
		if err := util.ExtractTar(contextTar, contextTmpDir); err != nil {
			return "", "", "", fmt.Errorf("unable to extract context tar to tmp context dir: %s", err)
		}
	}

	return sessionTmpDir, contextTmpDir, dockerfileTmpPath, nil
}
