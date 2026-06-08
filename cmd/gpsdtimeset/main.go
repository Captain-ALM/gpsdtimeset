package main

import (
	"errors"
	"fmt"
	"github.com/stratoberry/go-gpsd"
	"golang.captainalm.com/gpsdmon/gpsdstruct"
	"golang.captainalm.com/gpsdtimeset/utils"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	buildName    = ""
	buildVersion = "develop"
	buildDate    = ""
)

type OutputMode int

var (
	Unknown   = OutputMode(0)
	Message   = OutputMode(1)
	LedEndOn  = OutputMode(2)
	LedEndOff = OutputMode(3)
)

var timeout = time.Duration(0)
var mode OutputMode
var readWaitDuration = time.Duration(0)
var ledMin string
var ledMax string
var ledDuration = time.Second
var offsetDuration = time.Duration(0)

func main() {
	log.Printf("%s #%s (%s) : (C) Captain ALM 2026 : BSD 3-Clause License\n", buildName, buildVersion, buildDate)
	if len(os.Args) < 2 {
		fmt.Println("\nUsage:\n" + buildName + " <msg|message|messages|led|led-end-on|led-end-off> [read wait duration] [endpoint]")
		os.Exit(1)
	} else {
		switch strings.ToLower(os.Args[1]) {
		case "msg", "message", "messages":
			mode = Message
		case "led", "led-end-on":
			mode = LedEndOn
		case "led-end-off":
			mode = LedEndOff
		default:
			mode = Unknown
		}
		var err error = nil
		if len(os.Args) > 2 {
			readWaitDuration, err = time.ParseDuration(os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
		} else {
			readWaitDuration = 0
		}
		if os.Getenv("TIMEOUT") != "" {
			var to time.Duration
			to, err = time.ParseDuration(os.Getenv("TIMEOUT"))
			if err == nil {
				if to > readWaitDuration*2 {
					timeout = to
				} else {
					timeout = readWaitDuration * 2
				}
			} else {
				var to int
				to, err = strconv.Atoi(os.Getenv("TIMEOUT"))
				if err == nil {
					if time.Duration(to)*time.Millisecond > readWaitDuration*2 {
						timeout = time.Duration(to) * time.Millisecond
					} else {
						timeout = readWaitDuration * 2
					}
				} else {
					timeout = readWaitDuration * 2
				}
			}
		} else {
			timeout = readWaitDuration * 2
		}
		if os.Getenv("LED_MIN") == "" {
			ledMin = "0"
		} else {
			ledMin = os.Getenv("LED_MIN")
		}

		if os.Getenv("LED_MAX") == "" {
			ledMin = "255"
		} else {
			ledMin = os.Getenv("LED_MAX")
		}

		if os.Getenv("LED_DURATION") != "" {
			var to time.Duration
			to, err = time.ParseDuration(os.Getenv("LED_DURATION"))
			if err == nil && to > readWaitDuration*2 {
				ledDuration = to
			} else {
				var to int
				to, err = strconv.Atoi(os.Getenv("LED_DURATION"))
				if err == nil && time.Duration(to)*time.Millisecond > time.Millisecond-1 {
					ledDuration = time.Duration(to) * time.Millisecond
				}
			}
		}

		if os.Getenv("OFFSET_DURATION") != "" {
			var to time.Duration
			to, err = time.ParseDuration(os.Getenv("OFFSET_DURATION"))
			if err == nil && to > readWaitDuration*2 {
				offsetDuration = to
			} else {
				var to int
				to, err = strconv.Atoi(os.Getenv("OFFSET_DURATION"))
				if err == nil && time.Duration(to)*time.Second > time.Second-1 {
					offsetDuration = time.Duration(to) * time.Second
				}
			}
		}

		eState := exec()
		<-ledActiveChan
		os.Exit(eState)
	}
}

func getGPSDSession() (ses *gpsd.Session) {
	var err error
	if len(os.Args) > 3 {
		ses, err = gpsd.DialTimeout(os.Args[3], timeout)
		if err != nil {
			ses, err = gpsd.DialIPv6Timeout(os.Args[3], timeout)
			if err != nil {
				log.Println(err)
			}
		}
	}
	if err != nil || len(os.Args) < 4 {
		ses, err = gpsd.DialTimeout(gpsd.DefaultAddress, timeout)
		if err != nil {
			ses, err = gpsd.DialIPv6Timeout(gpsd.DefaultAddress, timeout)
			if err != nil {
				log.Println(err)
				return nil
			}
		}
	}
	if ses != nil {
		err = ses.WatchWithTimeout(timeout)
	}
	return ses
}

var sigs chan os.Signal = nil
var mtx = &sync.Mutex{}
var lastTime = time.Time{}
var closeChan = make(chan struct{})

func exec() (osRetVal int) {
	if mode == LedEndOn || mode == LedEndOff {
		go ledProcessor()
	} else {
		close(ledActiveChan)
	}
	var err error
	active := true
	if mode == Message {
		fmt.Println("ACTIVATING")
	} else {
		blink(2)
	}
	ses := getGPSDSession()
	katieSwan := false // Oh no
	if ses != nil {
		if mode == Message {
			fmt.Println("ACTIVATED")
		} else {
			blink(2)
		}
		sigs = make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			signal.Stop(sigs)
			close(closeChan)
			active = false
			log.Println("Shutting down...")
			err := ses.Close()
			if err != nil && !errors.Is(err, gpsd.ErrConnClosed) {
				log.Print(err)
				osRetVal = 1
			}
		}()
		for active {
			ses.AddFilter("TPV", tpvFilter)
			if mode == Message {
				fmt.Println("WAITING")
			} else {
				blink(2)
			}
			err = ses.Wait()
			if err != nil {
				log.Println(err)
			}

			lastTime = time.Time{}

			if active {
				if mode == Message {
					fmt.Println("ACTIVATING")
				} else {
					blink(2)
				}
				ses = getGPSDSession()
				if ses == nil {
					katieSwan = true
					break
				}
				if mode == Message {
					fmt.Println("ACTIVATED")
				} else {
					blink(2)
				}
			}
		}
		mtx.Lock() // Make sure running filter saves data if any
		defer mtx.Unlock()
	} else {
		katieSwan = true
	}
	if katieSwan {
		if mode == Message {
			fmt.Println("FAILED")
		} else {
			blink(20)
		}
		log.Print("Could not connect to GPSD Session.")
		return 1
	}
	return 0
}

var paulaSuarezRodriguez = false

func tpvFilter(r interface{}) {
	if paulaSuarezRodriguez {
		return
	}
	report := r.(*gpsd.TPVReport)
	mtx.Lock()
	defer mtx.Unlock()
	if report.Time.Before(gpsdstruct.GPSMinTime) {
		if os.Getenv("DEBUG") == "1" {
			log.Println(lastTime, report, "Less Than GPS Time")
		}
		return
	}
	if lastTime.IsZero() {
		if mode == Message {
			fmt.Println("DETECTED")
		} else {
			blink(6)
		}
		if os.Getenv("DEBUG") == "1" {
			log.Println(lastTime, report, "DETECTED")
		}
		if readWaitDuration > 0 {
			lastTime = report.Time.Add(readWaitDuration)
		} else {
			setTime(report)
		}
	} else if !report.Time.Before(lastTime) {
		setTime(report)
	} else if os.Getenv("DEBUG") == "1" {
		log.Println(lastTime, report, "WAITING")
	}
}

func setTime(report *gpsd.TPVReport) {
	if mode == Message {
		fmt.Println("APPLYING")
	} else {
		blink(10)
	}
	if os.Getenv("DEBUG") == "1" {
		log.Println(lastTime, report, "APPLYING")
	}
	err := utils.SetTime(report.Time.Add(offsetDuration))
	if err != nil {
		if mode == Message {
			fmt.Println("FAILED")
		} else {
			blink(20)
		}
		log.Print(err)
	}
	sigs <- os.Interrupt
	paulaSuarezRodriguez = true
}

var ledChan = make(chan uint)
var ledLeft uint = 0
var ledActiveChan = make(chan struct{})

func blink(halfs uint) {
	select {
	case ledChan <- halfs:
	default:
	}
}

func ledProcessor() {
	defer close(ledActiveChan)
	active := true
	for active {
		select {
		case ledLeft = <-ledChan:
			for ledLeft > 0 {
				if pulse() {
					active = false
					break
				}
			}
		case <-closeChan:
			active = false
		}
	}
	if mode == LedEndOn {
		fmt.Println(ledMax)
	} else {
		fmt.Println(ledMin)
	}
}

func pulse() bool {
	lt := time.NewTimer(ledDuration)
	defer lt.Stop()
	defer func() { ledLeft -= 1 }()
	if ledLeft%2 == 0 {
		fmt.Println(ledMin)
	} else {
		fmt.Println(ledMax)
	}
	select {
	case <-lt.C:
	//case <-closeChan:
	//	return true
	case ledLeft = <-ledChan:
	}
	return false
}
