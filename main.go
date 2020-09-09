package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// Build vars
	sha1ver   string // sha1 revision used to build the program
	buildTime string // when the executable was built
	version   string
)

type Config struct {
	debug     bool
	url       string
	key       string
	prepend   string
	downgrade bool
}

var cfg Config

func main() {

	history := make(map[string]Alarm)

	checkEnv()

	// Set up a channel to listen to for interrupt signals
	var runChan = make(chan os.Signal, 1)

	// Handle ctrl+c/ctrl+x interrupt
	signal.Notify(runChan, os.Interrupt, syscall.SIGTSTP)

	scrapeInterval, err := time.ParseDuration(os.Getenv("CWPOLL_INT"))
	if err != nil {
		if cfg.debug {
			log.Print(os.Getenv("CWPOLL_INT"))
			log.Println(err)
		}

		// Set default interval
		scrapeInterval, _ = time.ParseDuration("60s")
	}

	ticker := time.NewTicker(scrapeInterval)
	done := make(chan bool)

	// Create AWS session
	mySession := session.Must(session.NewSession())
	prefix := "red/"
	if os.Getenv("CWPOLL_PREFIX") != "" {
		prefix = os.Getenv("CWPOLL_PREFIX")
	}

	limit := int64(100)
	if os.Getenv("CWPOLL_LIMIT") != "" {
		nl, err := strconv.ParseInt(os.Getenv("CWPOLL_LIMIT"), 10, 64)
		if err != nil {
			log.Println(err)
			limit = 100
		}
		limit = nl
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				currentAlarms, err := getAlarmMap(mySession, &prefix, &limit)
				if err != nil {
					fmt.Println(err.Error())
					// Try re-establish connection
					if strings.Contains(err.Error(), "no such host") {
						mySession = session.Must(session.NewSession())
					}
					return
				}

				// Iterate history to clear
				for _, alarm := range history {

					_, ok := currentAlarms[alarm.Id]
					if ok {
						// Alarm Still active
					} else {
						// Clear
						// send clear msg
						log.Printf("Clear: %s: %s - %s \n", alarm.Id, alarm.Name, alarm.Description)
						alarm.State = "OK"
						err := postToElog(cfg.url, cfg.key, alarm)
						if err != nil {
							log.Println(err)
						}
					}
				}

				// Iterate current to send new
				for _, alarm := range currentAlarms {
					_, ok := history[alarm.Id]
					if ok {
						// Alarm already active
					} else {
						// Raise alert
						// send new alert  msg
						log.Printf("Trigger: %s: %s - %s \n", alarm.Id, alarm.Name, alarm.Description)
						err := postToElog(cfg.url, cfg.key, alarm)
						if err != nil {
							log.Println(err)
						}

					}
				}

				history = currentAlarms

				if cfg.debug {
					for i, data := range history {
						log.Printf("%s: %s - %s \n", i, data.Name, data.Description)
					}

					log.Println("Tick at", t)
				}

			}
		}
	}()

	// Block on this channel listeninf for those previously defined syscalls assign
	// to variable so we can let the user know why the server is shutting down
	interrupt := <-runChan
	log.Printf("Server is shutting down due to %+v\n", interrupt)

	ticker.Stop()
	done <- true

}

func checkEnv() {
	if os.Getenv("AWS_REGION") == "" {
		fmt.Println("AWS_REGION not set")
		os.Exit(1)
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		fmt.Println("AWS_ACCESS_KEY_ID not set")
		os.Exit(1)
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		fmt.Println("AWS_SECRET_ACCESS_KEY")
		os.Exit(1)
	}

	// String to prepend to description
	if os.Getenv("CWPOLL_PRE") != "" {
		cfg.prepend = os.Getenv("CWPOLL_PRE")
	}

	// Downgrade Priority
	if os.Getenv("CWPOLL_DOWN") == "1" {
		cfg.downgrade = true
	} else {
		cfg.downgrade = false
	}

	if os.Getenv("ELOG_URL") == "" {
		fmt.Println("ELOG_URL")
		os.Exit(1)
	}
	cfg.url = os.Getenv("ELOG_URL")

	if os.Getenv("ELOG_KEY") == "" {
		fmt.Println("ELOG_KEY")
		os.Exit(1)
	}
	cfg.key = os.Getenv("ELOG_KEY")

	if os.Getenv("CWPOLL_DEBUG") == "1" {
		cfg.debug = true
	} else {
		cfg.debug = false
	}

}
