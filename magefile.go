//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/livekit/livekit-server/version"
	"github.com/livekit/mageutil"
	_ "github.com/livekit/psrpc"
)

const (
	goChecksumFile = ".checksumgo"
	imageName      = "livekit/livekit-server"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build
var checksummer = mageutil.NewChecksummer(".", goChecksumFile, ".go", ".mod")

func init() {
	checksummer.IgnoredPaths = []string{
		"pkg/service/wire_gen.go",
		"pkg/rtc/types/typesfakes",
	}
}

// explicitly reinstall all deps
func Deps() error {
	return installTools(true)
}

// builds LiveKit server
func Build() error {
	mg.Deps(generateWire)
	if !checksummer.IsChanged() {
		fmt.Println("up to date")
		return nil
	}

	fmt.Println("building...")
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	if err := mageutil.RunDir(context.Background(), "cmd/server", "go build -o ../../bin/livekit-server"); err != nil {
		return err
	}

	checksummer.WriteChecksum()
	return nil
}

// builds binary that runs on linux amd64
func BuildLinux() error {
	mg.Deps(generateWire)
	if !checksummer.IsChanged() {
		fmt.Println("up to date")
		return nil
	}

	fmt.Println("building...")
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	cmd := mageutil.CommandDir(context.Background(), "cmd/server", "go build -buildvcs=false -o ../../bin/livekit-server-amd64")
	cmd.Env = []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"HOME=" + os.Getenv("HOME"),
		"GOPATH=" + os.Getenv("GOPATH"),
	}
	if err := cmd.Run(); err != nil {
		return err
	}

	checksummer.WriteChecksum()
	return nil
}

func Deadlock() error {
	ctx := context.Background()
	if err := mageutil.InstallTool("golang.org/x/tools/cmd/goimports", "latest", false); err != nil {
		return err
	}
	if err := mageutil.Run(ctx, "go get github.com/sasha-s/go-deadlock"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.Mutex ./pkg", "xargs sed -i  -e s/sync.Mutex/deadlock.Mutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.RWMutex ./pkg", "xargs sed -i  -e s/sync.RWMutex/deadlock.RWMutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl deadlock.Mutex\\|deadlock.RWMutex ./pkg", "xargs goimports -w"); err != nil {
		return err
	}
	if err := mageutil.Run(ctx, "go mod tidy"); err != nil {
		return err
	}
	return nil
}

func Sync() error {
	if err := mageutil.Pipe("grep -rl deadlock.Mutex ./pkg", "xargs sed -i  -e s/deadlock.Mutex/sync.Mutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl deadlock.RWMutex ./pkg", "xargs sed -i  -e s/deadlock.RWMutex/sync.RWMutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.Mutex\\|sync.RWMutex ./pkg", "xargs goimports -w"); err != nil {
		return err
	}
	if err := mageutil.Run(context.Background(), "go mod tidy"); err != nil {
		return err
	}
	return nil
}

// builds and publish snapshot docker image
func PublishDocker() error {
	// don't publish snapshot versions as latest or minor version
	if !strings.Contains(version.Version, "SNAPSHOT") {
		return errors.New("Cannot publish non-snapshot versions")
	}

	versionImg := fmt.Sprintf("%s:v%s", imageName, version.Version)
	cmd := exec.Command("docker", "buildx", "build",
		"--push", "--platform", "linux/amd64,linux/arm64",
		"--tag", versionImg,
		".")
	mageutil.ConnectStd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// regenerate psrpc service definitions
func Psrpc() error {
	psrpcProtoFiles := []string{
		"pkg/service/rpc/egress.proto",
		"pkg/service/rpc/ingress.proto",
		"pkg/service/rpc/io.proto",
	}

	fmt.Println("generating psrpc")

	protocolDir, err := mageutil.GetPkgDir("github.com/livekit/protocol")
	if err != nil {
		return err
	}

	psrpcDir, err := mageutil.GetPkgDir("github.com/livekit/psrpc")
	if err != nil {
		return err
	}

	protoc, err := mageutil.GetToolPath("protoc")
	if err != nil {
		return err
	}
	protocGoPath, err := mageutil.GetToolPath("protoc-gen-go")
	if err != nil {
		return err
	}
	psrpcPath, err := mageutil.GetToolPath("protoc-gen-psrpc")
	if err != nil {
		return err
	}

	fmt.Println("generating psrpc protobuf")
	args := append([]string{
		"--go_out", ".",
		"--psrpc_out", ".",
		"--go_opt=paths=source_relative",
		"--psrpc_opt=paths=source_relative",
		"--plugin=go=" + protocGoPath,
		"--plugin=psrpc=" + psrpcPath,
		"-I" + protocolDir,
		"-I" + psrpcDir + "/protoc-gen-psrpc/options",
		"-I=.",
	}, psrpcProtoFiles...)
	cmd := exec.Command(protoc, args...)
	mageutil.ConnectStd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// run unit tests, skipping integration
func Test() error {
	mg.Deps(generateWire, setULimit)
	return mageutil.Run(context.Background(), "go test -short ./... -count=1")
}

// run all tests including integration
func TestAll() error {
	mg.Deps(generateWire, setULimit)
	return mageutil.Run(context.Background(), "go test ./... -count=1 -timeout=4m -v")
}

// cleans up builds
func Clean() {
	fmt.Println("cleaning...")
	os.RemoveAll("bin")
	os.Remove(goChecksumFile)
}

// regenerate code
func Generate() error {
	mg.Deps(installDeps, generateWire)

	fmt.Println("generating...")
	return mageutil.Run(context.Background(), "go generate ./...")
}

// code generation for wiring
func generateWire() error {
	mg.Deps(installDeps)
	if !checksummer.IsChanged() {
		return nil
	}

	fmt.Println("wiring...")

	wire, err := mageutil.GetToolPath("wire")
	if err != nil {
		return err
	}
	cmd := exec.Command(wire)
	cmd.Dir = "pkg/service"
	mageutil.ConnectStd(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// implicitly install deps
func installDeps() error {
	return installTools(false)
}

func installTools(force bool) error {
	tools := map[string]string{
		"github.com/google/wire/cmd/wire": "latest",
	}
	for t, v := range tools {
		if err := mageutil.InstallTool(t, v, force); err != nil {
			return err
		}
	}
	return nil
}
