package util

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

const (
	// The amount of time to do a blocking query for
	defaultWaitTime = 60 * time.Second

	// pollErrorSleep the amount of time to sleep when an error occurs
	// TODO: make this an exponential backoff.
	pollErrorSleep = 5 * time.Second
)

type Watcher struct {
	// DataCh is the chan where new WatchData will be published
	DataCh chan *WatchData

	// ErrCh is the chan where any errors will be published
	ErrCh chan error

	// FinishCh is the chan where the watcher reports it is "done"
	FinishCh chan struct{}

	// client is the mechanism for communicating with the Consul API
	client *etcd.Client

	// dependencies is the slice of Dependencies this Watcher will poll
	dependencies []Dependency

	// waitGroup is the WaitGroup to ensure all Go routines return when we stop
	waitGroup sync.WaitGroup
}

func NewWatcher(client *etcd.Client, dependencies []Dependency) (*Watcher, error) {
	watcher := &Watcher{
		client:       client,
		dependencies: dependencies,
	}
	if err := watcher.init(); err != nil {
		return nil, err
	}

	return watcher, nil
}

func (w *Watcher) Watch() {
	log.Printf("[DEBUG] (watcher) starting watch")

	views := make([]*WatchData, 0, len(w.dependencies))
	for _, dependency := range w.dependencies {
		view, err := NewWatchData(dependency)
		if err != nil {
			w.ErrCh <- err
			return
		}

		views = append(views, view)
	}

	for _, view := range views {
		w.waitGroup.Add(1)
		go func(view *WatchData) {
			defer w.waitGroup.Done()
			view.poll(w)
		}(view)
	}

	log.Printf("[DEBUG] (watcher) all pollers have started, waiting for finish")
	w.waitGroup.Wait()

	log.Printf("[DEBUG] (watcher) closing finish channel")
	close(w.FinishCh)
}

func (w *Watcher) init() error {
	if w.client == nil {
		return fmt.Errorf("watcher: missing Consul API client")
	}

	if len(w.dependencies) == 0 {
		log.Printf("[WARN] (watcher) no dependencies in template(s)")
	}

	// Setup the chans
	w.DataCh = make(chan *WatchData)
	w.ErrCh = make(chan error)
	w.FinishCh = make(chan struct{})

	return nil
}

/// ------------------------- ///

type WatchData struct {
	Dependency Dependency
	Data       interface{}

	receivedData bool
}

func NewWatchData(dependency Dependency) (*WatchData, error) {
	if dependency == nil {
		return nil, fmt.Errorf("watchdata: missing Dependency")
	}

	return &WatchData{Dependency: dependency}, nil
}

func (wd *WatchData) poll(w *Watcher) {
	log.Printf("[DEBUG] (%s) starting poll", wd.Display())

	data, _, err := wd.Dependency.Fetch(w.client)
	if err != nil {
		log.Printf("[ERR] (%s) %s", wd.Display(), err.Error())
		w.ErrCh <- err
		time.Sleep(pollErrorSleep)
		wd.poll(w)
		return
	}

	log.Printf("[DEBUG] (%s) writing data to channel", wd.Display())

	// If we got this far, there is new data!
	wd.Data = data
	wd.receivedData = true
	w.DataCh <- wd
}

func (wd *WatchData) Display() string {
	return wd.Dependency.Display()
}
