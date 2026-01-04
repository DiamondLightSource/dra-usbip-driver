package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/kubeconfig"
)

var (
	kubeConfigFile string
)

func init() {
	pflag.StringVar(&kubeConfigFile, "kubeconfig", "", "kube config file path")
}

func main() {
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	err := os.MkdirAll("/var/lib/kubelet/plugins/usbip", 0750)
	if err != nil {
		panic(err)
	}

	coreclient, err := kubeconfig.GetCoreClient(kubeConfigFile)
	if err != nil {
		panic(err)
	}

	driver, err := NewDriver(ctx, coreclient)
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
