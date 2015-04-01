package stats

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zvelo/zvelo-services/port"
)

var (
	// Statsd is the global client
	Statsd    *Client
	statsOnce sync.Once
	// Initialized denotes if the Init() successfully completed (including conneciton opened to statsd)
	Initialized                              bool
	countCh, gaugeCh, gaugeUpdateCh, timerCh chan Metric
	// ReqDataCh is a channel for requesting the stats data, sending anything
	// indicates a request and must be followed by a read of DataCh
	ReqDataCh chan interface{}
	// DataCh is a channel where the current stats data is sent following a
	// request on ReqDataCh
	DataCh chan Data
)

// Data is a data structure for tracking (simplified) stat data in memory
type Data map[string]int64

// Base metric data
type data struct {
	name  string
	value int64
	rate  float32 // sample rate, default 1.0
}

func (m *data) Name() string {
	return m.name
}
func (m *data) Value() int64 {
	return m.value
}
func (m *data) Rate() float32 {
	return m.rate
}

// Metric as base data
type Metric interface {
	Name() string
	Value() int64
	Rate() float32 // deafault is a 100% sampling rate, 1.0
	format() string
}

// A Counter keeps a running total of counts
type Counter struct {
	data
}

func (c *Counter) format() string {
	return "%d|c|@%g"
}

// Timer is a metric with a start time
type Timer struct {
	data
	start time.Time
}

func (t *Timer) format() string {
	return "%d|ms|@%g"
}

// Gauge is a point-in-time reading that
// Sets the gauge to the value provided.
// Setting sum to true will have the metrics value added to the gauge's current
// if the gauge metric already exists
type Gauge struct {
	data
	sum bool // set to true to add the metric's value to the extant gauge's value (if it exists)
}

func (g *Gauge) format() string {
	if g.sum == true {
		return "+%d|g|@%g" // explicitly print sign for both pos and neg when summing gauge
	}
	return "%d|g|@%g"
}

// Set is a unique occurences of events between flushes.
type Set struct {
	data
}

func (s *Set) format() string {
	return "%d|s|@%g"
}

// Client is a simple object for sending data to statsd
type Client struct {
	conn  net.Conn
	addr  string
	image string
	ip    string
}

// Send metrics
func (c *Client) send(metrics ...Metric) error {
	// TODO metrics can be sent as one write, newline separated, so long as the
	// total length dos not exceed a single frame (jumbo frame is 8932B)
	for _, m := range metrics {

		// Send host-deliniated stat
		// Note that summations can be created downstream to
		// measure cluster-wide stats
		data := c.serialize(m, false)
		log.Printf("[INFO] [statsd] " + data)
		if c.conn != nil {
			_, err := fmt.Fprintf(c.conn, data)
			if err != nil {
				return err
			}
		} else {
			log.Println("[INFO] [statsd] No connection")
		}

	}
	return nil
}

// Seralize a formatable metric for transmission to statsd
// @param clusterwisde serializeds without jhost IP being specified (thus being a cluster-wide stat)
func (c *Client) serialize(metric Metric, clusterwide bool) string {
	imageSerialized := strings.Replace(strings.ToLower(c.image), "/", ".", -1)
	ipSerialized := strings.Replace(c.ip, ".", "_", -1)
	var prefix string
	if clusterwide {
		prefix = fmt.Sprintf("%s.", imageSerialized) // omits ip
	} else {
		prefix = fmt.Sprintf("%s.%s.", imageSerialized, ipSerialized)
	}
	format := fmt.Sprintf("%s%s:%s", prefix, metric.Name(), metric.format())
	return fmt.Sprintf(format, metric.Value(), metric.Rate())
}

// init provides standard initialization
func init() {
	// flag indicating whether explicit initialization has been performed
	Initialized = false
}

// doInit is called exactly once by Init().
// This is broken out into a separate function to enable re-calling
func doInit() error {
	// Set up channels
	countCh = make(chan Metric)
	gaugeCh = make(chan Metric)
	gaugeUpdateCh = make(chan Metric) // for additive gauge changes
	timerCh = make(chan Metric)
	ReqDataCh = make(chan interface{})
	DataCh = make(chan Data)

	go dataBroker()

	image := os.Getenv("IMAGE_NAME")
	if len(image) == 0 {
		log.Println("[WARN] [statsd] could not set up client, IMAGE_NAME is empty")
		Statsd = &Client{}
		return errors.New("IMAGE_NAME is empty")
	}

	ip := os.Getenv("COREOS_PRIVATE_IPV4")
	if len(ip) == 0 {
		log.Println("[WARN] [statsd] could not set up client, COREOS_PRIVATE_IPV4 is empty")
		Statsd = &Client{}
		return errors.New("COREOS_PRIVATE_IPV4 is empty")
	}

	Statsd = &Client{
		addr:  ip + ":" + strconv.Itoa(int(port.Statsd)),
		image: image,
		ip:    ip,
	}

	err := Statsd.createSocket()
	Initialized = true
	return err
}

// Init initializes the default Statsd client and internal stats tracking.
func Init() {
	statsOnce.Do(func() { doInit() }) // Do don't do errors.
}

// Handle data receives
func dataBroker() {
	data := Data{}

	for {
		select {
		case val := <-countCh:
			data[val.Name()] += val.Value()
		case val := <-gaugeCh:
			data[val.Name()] = val.Value()
		case val := <-gaugeUpdateCh:
			data[val.Name()] += val.Value()
		case val := <-timerCh:
			data[val.Name()] = val.Value()
		case <-ReqDataCh:
			log.Printf("Received ReqDataChannel")
			// make a copy of the data since maps are passed by reference
			clone := Data{}
			for k, v := range data {
				clone[k] = v
			}
			DataCh <- clone
		}
	}
}

// The client address
func (c *Client) String() string {
	return c.addr
}

// CreateSocket opens the connection to the statsd server
func (c *Client) createSocket() error {
	conn, err := net.DialTimeout("udp", c.addr, time.Second)
	if err != nil {
		log.Println("[WARN] [statsd] could not set up client, error creating socket", err)
		return err
	}
	c.conn = conn

	log.Println("[DEBUG] [statsd] client created", c.addr)
	return nil
}

// Close the connection to the statsd server
func (c *Client) Close() error {
	log.Println("[DEBUG] [statsd] client closed", c.addr)
	err := c.conn.Close()
	c.conn = nil
	return err
}

/////////////////////////////////////////
// Instance Accessors to stats methods //
/////////////////////////////////////////

// Incr increments a counter of a statsd metric
func (c *Client) Incr(stat string, count int64, rate float32) error {
	return c.send(&Counter{data{stat, count, rate}})
}

// Decr decrements a counter of a statsd metric
func (c *Client) Decr(stat string, count int64, rate float32) error {
	return c.send(&Counter{data{stat, -count, rate}})
}

// SetGauge sets the value of a statsd gauge metric
func (c *Client) SetGauge(stat string, value int64, rate float32) error {
	return c.send(&Gauge{data{stat, value, rate}, false}) // not additive
}

// UpdateGauge updates the value of a statsd gauge by adding (possibly negative) value
func (c *Client) UpdateGauge(stat string, value int64, rate float32) error {
	return c.send(&Gauge{data{stat, value, rate}, true}) // additive
}

// Timing submits the value of a statsd timing metric with explicit value
func (c *Client) Timing(stat string, delta time.Duration, rate float32) error {
	ms := int64(delta / time.Millisecond)
	t0 := time.Now().Add(-delta) // approx. start time
	return c.send(&Timer{data{stat, ms, rate}, t0})
}

// StartTiming begins a timer (returned)
func (c *Client) StartTiming(stat string, rate float32) *Timer {
	return &Timer{data{stat, -1, rate}, time.Now()}
}

// StopTiming ends a timer (timer must have had 'start' populated in order to calculate value.
func (c *Client) StopTiming(t *Timer) error {
	t.value = int64(time.Since(t.start) / time.Millisecond)
	return c.send(t)
}

///////////////////////////////////////
// Static Accessors to stats methods //
///////////////////////////////////////

// Incr increments the counter using the global statsd client and updates the
// in-memory counter
func Incr(stat string, count int64) error {
	return IncrThrottled(stat, count, 1.0)
}

// IncrThrottled increments a counter, but throttles the amount of requests
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func IncrThrottled(stat string, count int64, rate float32) error {
	Init()
	countCh <- &Counter{data{stat, count, rate}}
	return Statsd.Incr(stat, count, rate)
}

// Decr decrements the counter using the global statsd client and updates the
// in-memory counter
func Decr(stat string, count int64) error {
	return DecrThrottled(stat, count, 1.0)
}

// DecrThrottled decrements a counter, but throttles the amount of requests
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func DecrThrottled(stat string, count int64, rate float32) error {
	Init()
	countCh <- &Counter{data{stat, -count, rate}}
	return Statsd.Decr(stat, count, rate)
}

// SetGauge sets the gauge to the given value
// in-memory gauge
func SetGauge(stat string, value int64) error {
	return SetGaugeThrottled(stat, value, 1.0)
}

// SetGaugeThrottled sets the gauge to the given value, but throttles the amount of requests
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func SetGaugeThrottled(stat string, value int64, rate float32) error {
	Init()
	gaugeCh <- &Gauge{data{stat, value, rate}, false} // not additive
	return Statsd.SetGauge(stat, value, rate)
}

// UpdateGauge adds the value (possibly negative) to the existing value of the gauge
func UpdateGauge(stat string, value int64) error {
	return UpdateGaugeThrottled(stat, value, 1.0)
}

// UpdateGaugeThrottled adds the value (possibly negative), but throttles the amount of requests
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func UpdateGaugeThrottled(stat string, value int64, rate float32) error {
	Init()
	gaugeUpdateCh <- &Gauge{data{stat, value, rate}, true} // additive
	return Statsd.UpdateGauge(stat, value, rate)
}

// Timing sets the results of a timing operation to an explicit duration.
func Timing(stat string, delta time.Duration) error {
	return TimingThrottled(stat, delta, 1.0)
}

// TimingThrottled sets the results of a timing operation, but throttles the amount of requests
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func TimingThrottled(stat string, delta time.Duration, rate float32) error {
	Init()
	ms := int64(delta / time.Millisecond)
	t0 := time.Now().Add(-delta) // approx. start time
	timerCh <- &Timer{data{stat, ms, rate}, t0}
	return Statsd.Timing(stat, delta, rate)
}

// StartTiming begins timing
func StartTiming(stat string) *Timer {
	return StartTimingThrottled(stat, 1.0)
}

// StartTimingThrottled begins timing, but the returned timer will be throttled when sent,
// sent on from statsd to grafana.  Set this 'rate' to less than 1.0
// for high frequency statistics.
func StartTimingThrottled(stat string, rate float32) *Timer {
	Init()
	// (no publish on channels until completed using StopTiming())
	return Statsd.StartTiming(stat, rate)
}

// StopTiming completes a timer, sending the result to statsd
func StopTiming(t *Timer) error {
	Init()
	t.value = int64(time.Since(t.start) / time.Millisecond)
	timerCh <- t
	return Statsd.StopTiming(t)
}
