package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gousb/usbid"
	"github.com/spf13/pflag"

	"gitlab.diamond.ac.uk/sysadmin/container-tools/dra-usbip-driver/pkg/devicemetadata"

	resourceapi "k8s.io/api/resource/v1"
	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/utils/ptr"
)

var (
	agents         []string
	kubeConfigFile string
)

func init() {
	pflag.StringSliceVar(&agents, "agent", nil, "one or more agents (host:port) to find devices from")
	pflag.StringVar(&kubeConfigFile, "kubeconfig", "", "kuibe config file path")
}

func getDevices(agent string) ([]resourceapi.Device, error) {
	resp, err := http.Get("http://" + agent + "/devices")
	if err != nil {
		return nil, fmt.Errorf("could not fetch devices from %s: %w", agent, err)
	}

	var devicesMetadata []devicemetadata.Metadata
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&devicesMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not decode json from %s: %w", agent, err)
	}

	var devices []resourceapi.Device

	for _, meta := range devicesMetadata {
		attrs := map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
			"vendor": {
				StringValue: ptr.To(fmt.Sprintf("%04x", int64(meta.Vendor))),
			},
			"product": {
				StringValue: ptr.To(fmt.Sprintf("%04x", int64(meta.Product))),
			},
		}
		if meta.Serial != "" {
			attrs["serial"] = resourceapi.DeviceAttribute{
				StringValue: ptr.To(meta.Serial),
			}
		}

		if vendorInfo, ok := usbid.Vendors[meta.Vendor]; ok {
			attrs["vendorName"] = resourceapi.DeviceAttribute{
				StringValue: ptr.To(vendorInfo.Name),
			}
			if productInfo, ok := vendorInfo.Product[meta.Product]; ok {
				attrs["productName"] = resourceapi.DeviceAttribute{
					StringValue: ptr.To(productInfo.Name),
				}
			}
		}

		d := resourceapi.Device{
			Name:       fmt.Sprintf("bus-%03d-%03d", meta.Bus, meta.Address),
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
			panic(err)
		}
	} else {
		csconfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			panic(err)
		}
	}

	coreclient, err := coreclientset.NewForConfig(csconfig)
	if err != nil {
		panic(err)
	}

	devices, err := getDevices(agents[0])
	if err != nil {
		panic(err)
	}

	resources := &resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			"usbip": {
				Slices: []resourceslice.Slice{
					{
						Devices: devices,
					},
				},
			},
		},
	}

	resourceSliceController, err := resourceslice.StartController(
		ctx,
		resourceslice.Options{
			DriverName: "usbip",
			KubeClient: coreclient,
			Resources:  resources,
			ErrorHandler: func(ctx context.Context, err error, msg string) {
				fmt.Println(err, msg)
			},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("Started controller")

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-time.After(2 * time.Second):
			for _, agent := range agents {
				devices, err := getDevices(agent)
				if err != nil {
					fmt.Println(err)
					continue
				}
				resources := &resourceslice.DriverResources{
					Pools: map[string]resourceslice.Pool{
						"usbip": {
							Slices: []resourceslice.Slice{
								{
									Devices: devices,
								},
							},
						},
					},
				}
				resourceSliceController.Update(resources)
			}
		}
	}
}
