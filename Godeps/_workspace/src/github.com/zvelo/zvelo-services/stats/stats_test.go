package stats

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPreconditions(t *testing.T) {

	Convey("Calling witout setting environment variables", t, func() {

		Convey("Internal init should fail when COREOS_PRIVATE_IPV4 is undefined.", func() {

			envBak := os.Getenv("COREOS_PRIVATE_IPV4")
			os.Unsetenv("COREOS_PRIVATE_IPV4")
			err := doInit()
			So(err, ShouldNotEqual, nil)
			So(Initialized, ShouldEqual, false)
			os.Setenv("COREOS_PRIVATE_IPV4", envBak)

		})

		Convey("init should fail when IMAGE_NAME is undefined.", func() {
			envBak := os.Getenv("IMAGE_NAME")
			os.Unsetenv("IMAGE_NAME")
			err := doInit()
			So(err, ShouldNotEqual, nil)
			So(Initialized, ShouldEqual, false)
			os.Setenv("IMAGE_NAME", envBak)
		})

		Convey("Should succeed when env vars set.", func() {
			ip := os.Getenv("COREOS_PRIVATE_IPV4")
			in := os.Getenv("IMAGE_NAME")
			if ip == "" {
				fmt.Println("Warning:  setting COREOS_PRIVATE_IPV4 to test value.")
				os.Setenv("COREOS_PRIVATE_IPV4", "127.0.0.1")
			}
			if in == "" {
				fmt.Println("Warning:  setting IMAGE_NAME to test value.")
				os.Setenv("IMAGE_NAME", "zvelo/stats_test")
			}
			doInit()
			So(Initialized, ShouldEqual, true)
		})

	})

}

// TestStats runs a battery of tests against the basic stats API.
// Note that Statsd itself is out-of-band and this tests receipts
// of data from the in-band channels upon which stats data is
// simulcast.
func TestStats(t *testing.T) {

	Init()

	// Test Counter increment
	Convey("Given a new Counter.", t, func() {

		name := "test_counter"
		value := int64(10)

		Convey("When incremented.", func() {

			IncrThrottled(name, value, 0.1) // one could also Decr(name, -value)

			Convey("The value should reflect that increase.", func() {
				ReqDataCh <- 1
				data := <-DataCh
				So(data[name], ShouldEqual, value)
			})
		})

		Convey("When decremented.", func() {

			Decr(name, value) // one could also Incr(name, -value)

			Convey("The counter should reflect that decrease.", func() {
				ReqDataCh <- 1
				data := <-DataCh
				So(data[name], ShouldEqual, (value - value))
			})
		})

	})

	// Test Gauges
	Convey("Given a new Gauge.", t, func() {

		name := "test_gauge"
		value := int64(10)

		// Set
		Convey("When the value is set.", func() {

			SetGauge(name, value)
			// gauge now == value

			Convey("The gauge should reflect the new value.", func() {
				ReqDataCh <- 1   // request data
				data := <-DataCh // wait for data
				So(data[name], ShouldEqual, value)
			})

		})

		// Update positive
		Convey("When the value is 'updated' with a positive numer.", func() {

			UpdateGauge(name, value)
			// gauge now == value*2

			Convey("The gauge should be increased by that amount.", func() {
				ReqDataCh <- 1   // request data
				data := <-DataCh // wait for data
				So(data[name], ShouldEqual, value*2)
			})
		})

		// Update negative
		Convey("When the value is 'updated' with a negative number.", func() {

			UpdateGauge(name, (-value))
			// gauge now == value

			Convey("The gauge should be decreased by that amount.", func() {
				ReqDataCh <- 1   // request data
				data := <-DataCh // wait for data
				So(data[name], ShouldEqual, value)
			})
		})
	})

	// Test Timers
	Convey("Give a time delta.", t, func() {

		name := "test_timer"

		// Set
		Convey("When the timer is set using a single operation.", func() {

			Timing(name, 10*time.Second)

			Convey("The timer should reflect the new value.", func() {
				ReqDataCh <- 1
				data := <-DataCh
				So(data[name], ShouldEqual, int64(10*time.Second/time.Millisecond))
			})

		})

		// Time then Set
		Convey("When the timer is set using start and stop operations.", func() {

			timer := StartTiming(name)
			time.Sleep(1 * time.Second)
			StopTiming(timer)

			Convey("The timer should show more than the time the thread slept.", func() {
				ReqDataCh <- 1
				data := <-DataCh
				So(data[name], ShouldBeGreaterThanOrEqualTo, int64(1*time.Second/time.Millisecond))
			})

		})

	})

	// Test Sets
	// TODO

}
