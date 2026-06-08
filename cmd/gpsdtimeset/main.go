package main

import (
	"fmt"
	"github.com/stratoberry/go-gpsd"
	"log"
	"os"
	"strconv"
	"strings"
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

func main() {
	log.Printf("%s #%s (%s) : (C) Captain ALM 2026 : BSD 3-Clause License\n", buildName, buildVersion, buildDate)
	if len(os.Args) < 2 {
		fmt.Println("\nUsage:\n" + buildName + " <msg|messages|led|led-end-on|led-end-off> [read wait duration] [endpoint]")
		os.Exit(1)
	} else {
		var mode OutputMode
		switch strings.ToLower(os.Args[1]) {
		case "msg", "messages":
			mode = Message
		case "led", "led-end-on":
			mode = LedEndOn
		case "led-end-off":
			mode = LedEndOff
		default:
			mode = Unknown
		}
		var err error = nil
		var readWaitDuration = time.Duration(0)
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
		var ledMin string
		if os.Getenv("LED_MIN") == "" {
			ledMin = "0"
		} else {
			ledMin = os.Getenv("LED_MIN")
		}
		var ledMax string
		if os.Getenv("LED_MAX") == "" {
			ledMin = "255"
		} else {
			ledMin = os.Getenv("LED_MAX")
		}
		var ledDuration = time.Second
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

		os.Exit(exec(mode, readWaitDuration, ledMin, ledMax, ledDuration))
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

func exec(mode OutputMode, readWaitDuration time.Duration, ledMin string, ledMax string, ledDuration time.Duration) (osRetVal int) {
	
	return 0
}
