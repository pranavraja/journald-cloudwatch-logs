package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type Writer struct {
	conn              *cloudwatchlogs.CloudWatchLogs
	logGroupName      string
	logStreamName     string
	rawMessage        bool
	nextSequenceToken string
}

func NewWriter(sess *awsSession.Session, logGroupName, logStreamName string, rawMessage bool, firstSeqToken string) (*Writer, error) {
	conn := cloudwatchlogs.New(sess)

	return &Writer{
		conn:              conn,
		logGroupName:      logGroupName,
		logStreamName:     logStreamName,
		rawMessage:        rawMessage,
		nextSequenceToken: firstSeqToken,
	}, nil
}

func (w *Writer) WriteBatch(records []Record) (string, error) {

	events := make([]*cloudwatchlogs.InputLogEvent, 0, len(records))
	for _, record := range records {
		// Skip records older than 24 hours or CloudWatch Logs will spew
		if record.TimeUsec < time.Now().Add(-24*time.Hour).UnixNano()/1000 {
			continue
		}

		var payload string
		if w.rawMessage {
			payload = record.Message
		} else {
			jsonDataBytes, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				return "", err
			}
			payload = string(jsonDataBytes)
		}

		events = append(events, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(payload),
			Timestamp: aws.Int64(record.TimeUsec / 1e3),
		})
	}

	putEvents := func() error {
		request := &cloudwatchlogs.PutLogEventsInput{
			LogEvents:     events,
			LogGroupName:  &w.logGroupName,
			LogStreamName: &w.logStreamName,
		}
		if w.nextSequenceToken != "" {
			request.SequenceToken = aws.String(w.nextSequenceToken)
		}
		result, err := w.conn.PutLogEvents(request)
		if err != nil {
			return err
		}
		w.nextSequenceToken = *result.NextSequenceToken
		return nil
	}

	createStream := func() error {
		request := &cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  &w.logGroupName,
			LogStreamName: &w.logStreamName,
		}
		_, err := w.conn.CreateLogStream(request)
		return err
	}

	err := putEvents()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				// Maybe our log stream doesn't exist yet. We'll try
				// to create it and then, if we're successful, try
				// writing the events again.
				err := createStream()
				if err != nil {
					return "", fmt.Errorf("failed to create stream: %s", err)
				}

				err = putEvents()
				if err != nil {
					return "", fmt.Errorf("failed to put events: %s", err)
				}
				return w.nextSequenceToken, nil
			}
			if awsErr.Code() == "DataAlreadyAcceptedException" {
				// This batch was already sent
				return "", nil
			}
			if awsErr.Code() == "InvalidSequenceTokenException" {
				request := &cloudwatchlogs.DescribeLogStreamsInput{
					LogGroupName:        &w.logGroupName,
					LogStreamNamePrefix: &w.logStreamName,
					Descending:          aws.Bool(true),
					Limit:               aws.Int64(1),
				}
				result, err := w.conn.DescribeLogStreams(request)
				if err != nil {
					return "", fmt.Errorf("failed to get next sequence token: %s", err)
				}

				w.nextSequenceToken = *(result.LogStreams[0].UploadSequenceToken)

				err = putEvents()
				if err != nil {
					return "", fmt.Errorf("failed to put events: %s", err)
				}
				return w.nextSequenceToken, nil
			}
		}
		return "", fmt.Errorf("failed to put events: %s", err)
	}

	return w.nextSequenceToken, nil
}
