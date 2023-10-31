package main

import (
	"flag"

	"github.com/kelseyhightower/envconfig"
	syncerConfig "github.com/lmxia/syncer/pkg/config"
	"github.com/lmxia/syncer/pkg/known"
	"github.com/multi-cluster-network/octopus/pkg/controllers"
	octopusClientset "github.com/multi-cluster-network/octopus/pkg/generated/clientset/versioned"
	"github.com/multi-cluster-network/octopus/pkg/generated/informers/externalversions"
	kubeinformers "github.com/multi-cluster-network/octopus/pkg/generated/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	localMasterURL  string
	localKubeconfig string
)

func init() {
	flag.StringVar(&localKubeconfig, "kubeconfig", "", "Path to kubeconfig of local cluster. Only required if out-of-cluster.")
	flag.StringVar(&localMasterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {
	flag.Parse()

	var oClient *octopusClientset.Clientset
	var hubKubeConfig *rest.Config

	agentSpec := controllers.Specification{}
	restConfig, err := clientcmd.BuildConfigFromFlags(localMasterURL, localKubeconfig)
	if err != nil {
		//
		return
	}
	// we will merge this repo into syncer, so user syncer prefix for now.
	if err := envconfig.Process("syncer", &agentSpec); err != nil {
		klog.Fatal(err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if !agentSpec.IsHub {
		hubKubeConfig, err = syncerConfig.GetHubConfig(k8sClient, agentSpec.HubURL, agentSpec.LocalNamespace)
	} else {
		hubKubeConfig = restConfig
	}

	if oClient, err = octopusClientset.NewForConfig(hubKubeConfig); err != nil {
		//
		return
	}
	ctx := signals.SetupSignalHandler()
	w, err := controllers.NewTunnel(oClient, &agentSpec, ctx.Done())
	if err != nil {
		//
		return
	}
	// up the interface.
	if w.Init() != nil {
		//
		return
	}
	hubInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(oClient, known.DefaultResync, kubeinformers.WithNamespace(agentSpec.ShareNamespace))
	peerController, err := controllers.NewPeerController(agentSpec, w, hubInformerFactory)
	peerController.Start(ctx)
	<-ctx.Done()

	// remove your self from hub.
	if err := w.Cleanup(); err != nil {
		klog.Error(nil, "Error cleaning up resources before removing peer")
	}

	klog.Info("All controllers stopped or exited. Stopping main loop")
}
