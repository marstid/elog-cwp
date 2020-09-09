package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"testing"
)

/*
The AWS environment variables has to be set for the test cases to work.

AWS_REGION
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
 */


func TestList(t *testing.T) {

	mySession := session.Must(session.NewSession())

	// Create a CloudWatch client from just a session.
	svc := cloudwatch.New(mySession)

	prefix := "red/"
	max := int64(100)

	input := &cloudwatch.DescribeAlarmsInput{
		AlarmNamePrefix: &prefix,
		MaxRecords:      &max,
	}

	out, err := svc.DescribeAlarms(input)
	if err != nil {
		fmt.Println(err)
	}

	for _, alarm := range out.MetricAlarms {
		log.Println(*alarm.AlarmName)
		ts := *alarm.StateUpdatedTimestamp
		log.Println(ts.Unix())
	}
}

func TestGetActiveAlerts(t *testing.T) {
	mySession := session.Must(session.NewSession())

	prefix := "red/"
	//prefix := ""
	max := int64(100)

	_, err := getActiveAlerts(mySession, &prefix, &max)
	if err != nil {
		log.Println(err)
	}

	am, _ := getAlarmMap(mySession, &prefix, &max)

	for _, alert := range am {
		fmt.Println(alert)
	}

}

func TestPostToElog(t *testing.T) {

	url := "http://localhost:8080/api/webhook/elog"
	key := "Bearer 123456789"

	tags := make(map[string]string)
	tags["priority"] = "P1"
	tags["product"] = "OGS"
	tags["site"] = "AWS1C0"
	tags["env"] = "Prod"

	al := Alarm{
		Name:        "TEST1",
		Id:          "123456",
		Description: "foobar",
		State:       "resolved",
		Tags:        tags,
	}

	err := postToElog(url, key, al)
	if err != nil {
		t.Fail()
	}

}
