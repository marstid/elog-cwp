package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Alarm struct {
	Name        string
	Description string
	Tags        map[string]string
	Id          string
	State       string
}

// eLog Event Type
type Event struct {
	ID          string            `json:"uuid,omitempty" bson:"uuid"`
	Triggered   time.Time         `json:"triggered,omitempty" bson:"triggered"`
	Cleared     time.Time         `json:"cleared,omitempty" bson:"cleared,omitempty"`
	Fingerprint string            `json:"fingerprint" bson:"fingerprint"`
	Priority    string            `json:"priority,omitempty" bson:"priority,omitempty"` // P1, P2, P3, P4, P5
	Severity    string            `json:"severity,omitempty" bson:"severity,omitempty"` // Critical, Warning, Info
	Status      string            `json:"status,omitempty" bson:"status"`
	Msg         string            `json:"msg" bson:"msg"`
	Resource    string            `json:"resource,omitempty" bson:"resource,omitempty"`
	Source      string            `json:"source" bson:"source"`
	Site        string            `json:"site,omitempty" bson:"site,omitempty"`
	Env         string            `json:"env,omitempty" bson:"env,omitempty"` // Environment - Production, Stage, Test etc
	KB          string            `json:"kb,omitempty" bson:"kb,omitempty"`
	Ticket      string            `json:"ticket,omitempty" bson:"ticket,omitempty"`
	Comment     string            `json:"comment,omitempty" bson:"comment,omitempty"` // Operator comment added when acknowledge
	Ack         bool              `json:"ack,omitempty" bson:"ack,omitempty"`         // Used to mark event as acknowledge by operator
	AckBy       string            `json:"ackby,omitempty" bson:"ackby,omitempty"`
	GraphUrl    string            `json:"graphurl,omitempty" bson:"graphurl,omitempty"` // Optional: A Link to a graph associated with the event
	Tags        map[string]string `json:"tags,omitempty" bson:"tags,omitempty"`
}

type AlarmMap map[string]Alarm

func getActiveAlerts(session *session.Session, prefix *string, limit *int64) (AlarmList []Alarm, err error) {

	svc := cloudwatch.New(session)

	state := "ALARM"
	input := &cloudwatch.DescribeAlarmsInput{
		AlarmNamePrefix: prefix,
		MaxRecords:      limit,
		StateValue:      &state,
	}

	out, err := svc.DescribeAlarms(input)
	if err != nil {
		return nil, err
	}

	for _, alarm := range out.MetricAlarms {

		ltf := &cloudwatch.ListTagsForResourceInput{
			ResourceARN: alarm.AlarmArn,
		}
		tags, err := svc.ListTagsForResource(ltf)
		if err != nil {
			log.Println(err)
		}

		m := make(map[string]string)
		for _, t := range tags.Tags {
			m[*t.Key] = *t.Value
		}

		name := strings.ReplaceAll(*alarm.AlarmName, "red/", "")
		ts := *alarm.StateUpdatedTimestamp
		//timeStamp :=
		newAlert := Alarm{
			Name:        name,
			Description: *alarm.AlarmDescription,
			Tags:        m,
			Id:          hash(*alarm.AlarmArn + strconv.FormatInt(ts.Unix(), 10)),
			State:       *alarm.StateValue,
		}
		AlarmList = append(AlarmList, newAlert)
	}
	return
}

func getAlarmMap(session *session.Session, prefix *string, limit *int64) (AlarmMap, error) {
	alarm := make(AlarmMap)

	al, err := getActiveAlerts(session, prefix, limit)
	if err != nil {
		return alarm, err
	}

	for _, a := range al {
		alarm[a.Id] = a
	}
	return alarm, nil
}

// Used to create a hash from the arn for a uniq identifier of the Alarm
func hash(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return strconv.Itoa(int(h.Sum32()))
}

func postToElog(url string, key string, alarm Alarm) error {

	var state string
	switch alarm.State {
	case "ALARM":
		state = "active"
	case "OK":
		state = "resolved"
	default:
		state = "unknown"
	}

	prio := alarm.Tags["priority"]
	if cfg.downgrade {
		prio = "P4"
	}

	event := Event{
		Fingerprint: alarm.Id,
		Priority:    prio,
		Site:        alarm.Tags["site"],
		Tags:        alarm.Tags,
		Msg:         cfg.prepend + alarm.Description,
		Status:      state,
		Resource:    alarm.Name,
		Source:      "CloudWatchPoller",
		Env:         alarm.Tags["env"],
	}

	postdata, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(postdata))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.Status != "200 OK" {
		return fmt.Errorf("Post error:  %v: %v", resp.Status, string(body))
	}

	return nil
}
