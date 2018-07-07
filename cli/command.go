package cli

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/jawher/mow.cli"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	freenasProvisioner "github.com/nmaupu/freenas-provisioner/provisioner"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"syscall"
	"time"
)

const (
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

var (
	// cli parameters
	kubeconfig      *string
	identifier      *string
	provisionerName *string
)

// Process all command line parameters
func Process(appName, appDesc, appVersion string) {
	syscall.Umask(0)
	flag.Set("logtostderr", "true")

	app := cli.App(appName, appDesc)
	app.Version("v version", fmt.Sprintf("%s version %s", appName, appVersion))

	kubeconfig = app.String(cli.StringOpt{
		Name:   "kubeconfig",
		Desc:   "Path to kubernetes configuration file (for out of cluster execution)",
		EnvVar: "KUBECONFIG",
	})
	identifier = app.String(cli.StringOpt{
		Name:   "i identifier",
		Value:  "freenas-nfs-provisioner",
		Desc:   "Provisioner identifier (e.g. if unsure set it to current node name)",
		EnvVar: "IDENTIFIER",
	})
	provisionerName = app.String(cli.StringOpt{
		Name:   "provisioner-name",
		Value:  "freenas.org/nfs",
		Desc:   "Provisioner Name (e.g. 'provisioner' attribute of storage-class)",
		EnvVar: "PROVISIONER_NAME",
	})

	app.Action = execute
	app.Run(os.Args)
}

func execute() {
	var err error
	var config *rest.Config

	/* Params checking */
	var msgs []string
	if *identifier == "" {
		msgs = append(msgs, "Identifier parameter must be specified")
	}

	// Print all parameters' error and exist if need be
	if len(msgs) > 0 {
		fmt.Fprintf(os.Stderr, "The following error(s) occured:\n")
		for _, m := range msgs {
			fmt.Fprintf(os.Stderr, "  - %s\n", m)
		}
		os.Exit(1)
	}
	/* End params checking */

	if *kubeconfig != "" {
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		// Create an InClusterConfig and use it to create a client for the controller
		// to use to communicate with Kubernetes
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	clientFreenasProvisioner := freenasProvisioner.New(
		clientset,
		*identifier,
	)

	// Start the provision controller which will dynamically provision datasets and nfs shares
	pc := controller.NewProvisionController(
		clientset,
		15*time.Second,
		*provisionerName,
		clientFreenasProvisioner,
		serverVersion.GitVersion,
		exponentialBackOffOnError,
		failedRetryThreshold,
		leasePeriod,
		renewDeadline,
		retryPeriod,
		termLimit,
	)
	pc.Run(wait.NeverStop)
}
