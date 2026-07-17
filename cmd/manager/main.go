package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/devicemetadata"

	resourceapi "k8s.io/api/resource/v1"
	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

var (
	agents         []string
	kubeConfigFile string
	driverName     string
)

func init() {
	pflag.StringSliceVar(&agents, "agent", nil, "one or more agents (host:port) to find devices from")
	pflag.StringVar(&kubeConfigFile, "kubeconfig", "", "kube config file path")
	pflag.StringVar(&driverName, "driver-name", "usbip.diamond.ac.uk", "unique name for the driver to identify as")
}

func getDevices(agent string) ([]resourceapi.Device, error) {
	resp, err := http.Get("http://" + agent + ":13240/devices")
	if err != nil {
		return nil, fmt.Errorf("could not fetch devices from %s: %w", agent, err)
	}

	var devicesMetadata []devicemetadata.UdevadmInfo
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&devicesMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not decode json from %s: %w", agent, err)
	}

	var devices []resourceapi.Device

	for _, meta := range devicesMetadata {
		attrs := map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
			"vendor": {
				StringValue: ptr.To(meta.VendorID),
			},
			"product": {
				StringValue: ptr.To(meta.ModelID),
			},
			"host": {
				StringValue: ptr.To(agent),
			},
			"bus": {
				StringValue: ptr.To(meta.SysName),
			},
		}

		if meta.SerialShort != "" {
			attrs["serial"] = resourceapi.DeviceAttribute{
				StringValue: ptr.To(meta.SerialShort),
			}
		}

		if meta.VendorName != "" {
			attrs["vendorName"] = resourceapi.DeviceAttribute{
				StringValue: ptr.To(meta.VendorName),
			}
		}
		if meta.ModelName != "" {
			attrs["productName"] = resourceapi.DeviceAttribute{
				StringValue: ptr.To(meta.ModelName),
			}
		}

		// Valid device names must have only lowercase alphanumeric
		// and dash characters. The bus IDs of devices connected to
		// USB hubs may have dots in, e.g "3-1.5", so are not valid.
		// Replace the dot with a lowercase "d" which can be turned
		// back later when needed for attaching with usbip.
		deviceName := strings.ReplaceAll(meta.SysName, ".", "d")

		d := resourceapi.Device{
			Name:       deviceName,
			Attributes: attrs,
		}

		devices = append(devices, d)
	}
	return devices, nil
}

func main() {
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	var csconfig *rest.Config
	var err error
	if kubeConfigFile == "" {
		csconfig, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalf("Unable to create in-cluster config: %s", err)
		}
	} else {
		csconfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			klog.Fatalf("Unable to create config from kubeconfig file: %s", err)
		}
	}

	coreclient, err := coreclientset.NewForConfig(csconfig)
	if err != nil {
		klog.Fatalf("Unable to create core API client: %s", err)
	}

	resourceSliceController, err := resourceslice.StartController(
		ctx,
		resourceslice.Options{
			DriverName: driverName,
			KubeClient: coreclient,
			ErrorHandler: func(ctx context.Context, err error, msg string) {
				klog.Errorf("Error updating resource slices: %s (%s)", err, msg)
			},
		},
	)
	if err != nil {
		klog.Fatalf("Failed starting resource slice controller: %s", err)
	}

	klog.Info("Started resource slice controller")

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(2 * time.Second):
			resources := &resourceslice.DriverResources{
				Pools: map[string]resourceslice.Pool{},
			}
			for _, agent := range agents {
				devices, err := getDevices(agent)
				if err != nil {
					klog.Errorf("Failed to fetch devices from %s: %s", agent, err)
					continue
				}
				resources.Pools[agent] = resourceslice.Pool{
					Slices: []resourceslice.Slice{
						{
							Devices: devices,
						},
					},
				}
			}
			resourceSliceController.Update(resources)
		}
	}
}
