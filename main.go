package main

import (
	"flag"
	"fmt"
	"os"
	"sync"


	log "k8s.io/klog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nithu0115/event-scrapper/signals"
)

var (
	kubeconfigPath string
	apiServerAddr  string
)

func newKubernetesClient(kubeconfigPath, apiServerAddr string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(kubeconfigPath, apiServerAddr)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize Kubernetes Client: %v", err)
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

	sharedInformers := informers.NewSharedInformerFactory(client, 0)
	eventInformer := sharedInformers.Core().V1().Events()

	eventExporter := newEventExporter(client, eventsInformer)
	stopCh := signals.SigHandler()

	if err != nil {
		log.Errorf("Error: %v", err)
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		eventExporter.Start(stopCh)
	}()

	// Startup the Informer(s)
	log.Infof("Starting shared Informer(s)")
	sharedInformers.Start(stopCh)
	wg.Wait()
	log.Warnf("Exiting main()")
	os.Exit(1)
}
