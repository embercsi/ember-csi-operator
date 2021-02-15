package main

import (
	"context"
	"flag"
	"os"
	"runtime"

	"github.com/embercsi/ember-csi-operator/pkg/apis"
	"github.com/embercsi/ember-csi-operator/pkg/controller"
	"github.com/embercsi/ember-csi-operator/pkg/controller/embercsi"
	"github.com/embercsi/ember-csi-operator/version"
	"github.com/golang/glog"
	"github.com/operator-framework/operator-lib/leader"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrl "sigs.k8s.io/controller-runtime"
)

func printVersion() {
	glog.Infof("ember-csi-operator Version: %v", version.Version)
	glog.Infof("Go Version: %s", runtime.Version())
	glog.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
}

func main() {
	flag.Set("logtostderr", "true")
	configFile := flag.String("config", "", "Config file. (Optional)")
	flag.Parse()
	printVersion()

	namespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		glog.Fatal("Failed to get watch namespace: ", found)
	}

	// Config File
	glog.Info("Reading Ember CSI Config File")
	embercsi.ReadConfig(configFile)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal("Error while getting config: ", err)
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), "ember-csi-operator-lock")
	if err != nil {
		glog.Fatal("Error becoming leader", err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		glog.Fatal("Error creating manager: ", err)
	}

	glog.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal("Error at AddToScheme: ", err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		glog.Fatal("Error at AddToManager: ", err)
	}

	glog.Info("Starting the Cmd.")

	f, err := os.Create("/tmp/operator-sdk-ready")
	f.Close()
	defer os.Remove("/tmp/operator-sdk-ready")

	// Start the Cmd
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		glog.Fatal("Error starting Cmd: ", err)
	}


}
