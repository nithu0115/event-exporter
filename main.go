package main

import (
	"flag"
	"os"
	"sync"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	log "k8s.io/klog"

	"github.com/event-exporter/signals"
)

var (
	kubeconfigPath string
	apiServerAddr  string
)

func newKubernetesClient(kubeconfigPath, apiServerAddr string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiServerAddr, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	config.ContentType = "application/vnd.kubernetes.protobuf"
	return kubernetes.NewForConfig(config)
}

func init() {
	flag.StringVar(&apiServerAddr, "apiServerAddr", "", "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	flag.StringVar(&kubeconfigPath, "kubeconfigPath", "", "Path to kubeconfig file with authorization and master location information.")
}

func main() {
	flag.Set("logtostderr", "true")
	defer log.Flush()
	flag.Parse()
	client, err := newKubernetesClient(kubeconfigPath, apiServerAddr)

	if err != nil {
		log.Fatal("Failed to initialize Kubernetes client: ", err)
	}

	sharedInformers := informers.NewSharedInformerFactory(client, 0)
	eventsInformer := sharedInformers.Core().V1().Events()

	eventExporter := newEventRouter(client, eventsInformer)
	stopCh := signals.SigHandler()

	if err != nil {
		log.Errorf("Error: %v", err)
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		eventExporter.Run(stopCh)
	}()

	// Startup the Informer(s)
	log.Infof("Starting shared Informer(s)")
	sharedInformers.Start(stopCh)
	wg.Wait()
	log.Warningf("Exiting main()")
	os.Exit(1)
}
