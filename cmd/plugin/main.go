package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/kubeconfig"
	"github.com/spf13/pflag"
)

var (
	kubeConfigFile string
	driverName     string
)

func init() {
	pflag.StringVar(&kubeConfigFile, "kubeconfig", "", "kube config file path")
	pflag.StringVar(&driverName, "driver-name", "usbip.diamond.ac.uk", "unique name for the driver to identify as")
}

func main() {
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	pluginPath := filepath.Join("/var/lib/kubelet/plugins/", driverName)
	err := os.MkdirAll(pluginPath, 0750)
	if err != nil {
		panic(err)
	}

	coreclient, err := kubeconfig.GetCoreClient(kubeConfigFile)
	if err != nil {
		panic(err)
	}

	driver, err := NewDriver(ctx, coreclient, driverName)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()
	stop()

	err = driver.Shutdown()
	if err != nil {
		panic(err)
	}
}
