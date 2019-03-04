package main

import (
	"context"
	"flag"
	"runtime"

	"github.com/embercsi/ember-csi-operator/pkg/apis"
	"github.com/embercsi/ember-csi-operator/pkg/controller"
	"github.com/embercsi/ember-csi-operator/pkg/controller/embercsi"
	"github.com/embercsi/ember-csi-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"github.com/golang/glog"
)

// var log = logf.Log.WithName("cmd")

func printVersion() {
	glog.Infof("ember-csi-operator Version: %v", version.Version)
	glog.Infof("Go Version: %s", runtime.Version())
	glog.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	glog.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		glog.Fatal(err, "failed to get watch namespace")
	}

	// Read Config File if provided
	configFile := flag.String("config", "", "Config file. (Optional)")
	flag.Parse()

	// Config File
	glog.Info("Reading Ember CSI Config File")
	embercsi.ReadConfig(configFile)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal("Error while getting config.", err)
	}

	// Become the leader before proceeding
	leader.Become(context.TODO(), "ember-csi-operator-lock")

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		glog.Fatal("Error becoming leader", err)
	}
	defer r.Unset()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		glog.Fatal("Error creating manager", err)
	}

	glog.Info("Registering Components.")
	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal("Error at AddToScheme", err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		glog.Fatal("Error at AddToManager", err)
	}

	glog.Info("Starting the Cmd.")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatal("manager exited non-zero", err)
	}
}
