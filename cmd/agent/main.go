package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"github.com/diamondlightsource/dra-usbip-driver/pkg/agentconfig"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/devicemetadata"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/kmod"
	"github.com/diamondlightsource/dra-usbip-driver/pkg/usbip"
)

var (
	bindAllDevices bool
	autoBind       bool
	configPath     string
	cfg            *agentconfig.Config
)

func init() {
	pflag.BoolVar(&bindAllDevices, "bind-all-devices", false, "bind all valid devices to usbip")
	pflag.BoolVar(&autoBind, "auto-bind", false, "poll for new USB devices every 5 seconds and bind them automatically")
	pflag.StringVar(&configPath, "config", "", "path to device filter config file")
}

// bindDevices discovers local USB devices, applies config filters,
// binds unbound devices if binding is enabled, and returns metadata
// for all non-excluded devices.
func bindDevices() ([]*devicemetadata.UdevadmInfo, error) {
	localDevices, err := usbip.GetLocalDevices()
	if err != nil {
		return nil, fmt.Errorf("could not list local usb devices: %w", err)
	}

	var devicesMetadata []*devicemetadata.UdevadmInfo
	for _, device := range localDevices {
		udevInfo, err := usbip.GetLocalDeviceInfo(device)
		if err != nil {
			klog.Errorf("Failed to get device info for %s: %s", device, err)
			continue
		}

		if cfg.ShouldExclude(device, udevInfo.VendorID, udevInfo.ModelID) {
			klog.Infof("Excluding device %s (%s:%s) by config", device, udevInfo.VendorID, udevInfo.ModelID)
			continue
		}

		if udevInfo.Driver != "usbip-host" && bindAllDevices {
			err := usbip.BindDevice(device)
			if err != nil {
				klog.Errorf("Failed to bind device %s: %s", device, err)
				continue
			}
			klog.Infof("Bound device %s", device)
		}

		devicesMetadata = append(devicesMetadata, udevInfo)
	}

	return devicesMetadata, nil
}

func listDevices(w http.ResponseWriter, req *http.Request) {
	devicesMetadata, err := bindDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(devicesMetadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode devices as json: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		klog.Errorf("Failed to write response: %s", err)
	}
}

// autoBindLoop periodically discovers and binds new USB devices.
func autoBindLoop() {
	klog.Info("Auto-bind enabled, polling every 5 seconds")
	for {
		if _, err := bindDevices(); err != nil {
			klog.Errorf("Auto-bind poll failed: %s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func main() {
	pflag.Parse()

	if configPath != "" {
		var err error
		cfg, err = agentconfig.Load(configPath)
		if err != nil {
			klog.Fatalf("Failed to load config: %s", err)
		}
		klog.Infof("Loaded device filter config from %s", configPath)
	}

	moduleLoaded, err := kmod.IsLoaded("usbip_host")
	if err != nil {
		klog.Fatalf("Unable to check kernel modules: %s", err)
	}
	if !moduleLoaded {
		klog.Fatal("usbip_host kernel module is not loaded")
	}

	if autoBind {
		go autoBindLoop()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", listDevices)

	klog.Fatal(http.ListenAndServe(":13240", mux))
}
