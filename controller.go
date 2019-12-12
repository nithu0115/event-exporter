package main

import (
	"context"
	"fmt"

	sinks "github.com/event-exporter/sinks"
	log "k8s.io/klog"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// EventRouter is responsible for maintaining a stream of kubernetes
// system Events and pushing them to another channel for storage
type EventRouter struct {
	// client is the main kubernetes interface
	client kubernetes.Interface

	// store of events populated by the shared informer
	lister corelisters.EventLister

	// returns true if the event store has been synced
	listerSynched cache.InformerSynced

	// event sink
	sink sinks.EventSinkInterface
}

// NewEventRouter will create a new event router using the input params
func newEventRouter(kubeClient kubernetes.Interface, eventsInformer coreinformers.EventInformer) *EventRouter {
	var ctx context.Context

	er := &EventRouter{
		client: kubeClient,
		sink:   sinks.ManufactureSink(ctx),
	}

	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    er.addEvent,
		UpdateFunc: er.updateEvent,
		DeleteFunc: er.deleteEvent,
	})
	er.lister = eventsInformer.Lister()
	er.listerSynched = eventsInformer.Informer().HasSynced
	return er
}

// Run starts the EventRouter/Controller.
func (er *EventRouter) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer log.Infof("Shutting down EventRouter")

	log.Infof("Starting EventRouter")

	// here is where we kick the caches into gear
	if !cache.WaitForCacheSync(stopCh, er.listerSynched) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	<-stopCh
}

// addEvent is called when an event is created, or during the initial list
func (er *EventRouter) addEvent(obj interface{}) {
	event := obj.(*v1.Event)
	//fmt.Println(event)
	er.sink.UpdateEvents(event, nil)
}

// updateEvent is called any time there is an update to an existing event
func (er *EventRouter) updateEvent(objOld interface{}, objNew interface{}) {
	oldEvent := objOld.(*v1.Event)
	newEvent := objNew.(*v1.Event)
	er.sink.UpdateEvents(newEvent, oldEvent)
}

// deleteEvent should only occur when the system garbage collects events via TTL expiration
func (er *EventRouter) deleteEvent(obj interface{}) {
	event, ok := obj.(*v1.Event)

	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.V(2).Info("Object is neither event nor tombstone: %+v", obj)
			return
		}
		event, ok = tombstone.Obj.(*v1.Event)
		if !ok {
			log.V(2).Info("Tombstone contains object that is not a pod: %+v", obj)
			return
		}
	}
	// NOTE: This should *only* happen on TTL expiration there
	// is no reason to push this to a sink
	log.V(5).Info("Event Deleted from the system:\n%v", event)
}
