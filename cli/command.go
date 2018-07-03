package cli

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/jawher/mow.cli"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/nmaupu/freenas-provisioner/freenas"
	freenasProvisioner "github.com/nmaupu/freenas-provisioner/provisioner"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	provisionerName           = "maupu.org/freenas"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

var (
	// cli parameters
	kubeconfig                      *string
	identifier                      *string
	protocol                        *string
	host                            *string
	port                            *int
	username, password              *string
	insecure                        *bool
	pool, mountpoint, parentDataset *string
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
		Desc:   "Provisioner identifier (e.g. if unsure set it to current node name)",
		EnvVar: "IDENTIFIER",
	})
	protocol = app.String(cli.StringOpt{
		Name:   "P protocol",
		Desc:   "Freenas protocol (http|https)",
		EnvVar: "FREENAS_PROTOCOL",
	})
	host = app.String(cli.StringOpt{
		Name:   "h host",
		Desc:   "Freenas host address",
		EnvVar: "FREENAS_HOST",
	})
	port = app.Int(cli.IntOpt{
		Name:   "p port",
		Value:  443,
		Desc:   "Freenas port",
		EnvVar: "FREENAS_PORT",
	})
	username = app.String(cli.StringOpt{
		Name:   "u username",
		Value:  "root",
		Desc:   "Freenas username for the API connection",
		EnvVar: "FREENAS_USERNAME",
	})
	password = app.String(cli.StringOpt{
		Name:   "w password",
		Desc:   "Freenas password for the API connection",
		EnvVar: "FREENAS_PASSWORD",
	})
	insecure = app.Bool(cli.BoolOpt{
		Name:   "insecure",
		Value:  false,
		Desc:   "Skip SSL check for Freenas API communications (self-signed certificate)",
		EnvVar: "FREENAS_INSECURE",
	})
	pool = app.String(cli.StringOpt{
		Name:   "pool",
		Value:  "tank",
		Desc:   "Pool to use for storage",
		EnvVar: "FREENAS_POOL",
	})
	mountpoint = app.String(cli.StringOpt{
		Name:   "mountpoint",
		Value:  "/mnt",
		Desc:   "Pool mountpoint",
		EnvVar: "FREENAS_MOUNTPOINT",
	})
	parentDataset = app.String(cli.StringOpt{
		Name:   "parentDataset",
		Desc:   "Parent dataset to use e.g. /<mountpoint>/<pool>/<parentDataset>, parent dataset must already exist !",
		EnvVar: "FREENAS_PARENT_DATASET",
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
	if *host == "" {
		msgs = append(msgs, "Host parameter must be specified")
	}
	if *username == "" || *password == "" {
		msgs = append(msgs, "Username and password parameters must be specified")
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

	/* Everything's good so far, ready to start */
	glog.Infoln("Starting freenas-provisioner with the following parameters:")
	glog.Infof("  Freenas address: %s://%s:%d\n", *protocol, *host, *port)
	glog.Infof("  Insecure: %t\n", *insecure)
	glog.Infof("  pool: %s\n", filepath.Join(*mountpoint, *pool))
	glog.Infof("  parentDataset: %s\n", *parentDataset)

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
		*pool,
		*mountpoint,
		*parentDataset,
		*identifier,
		freenas.NewFreenasServer(
			*protocol,
			*host, *port,
			*username, *password,
			*insecure,
		),
	)

	// Start the provision controller which will dynamically provision datasets and nfs shares
	pc := controller.NewProvisionController(
		clientset,
		15*time.Second,
		provisionerName,
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
