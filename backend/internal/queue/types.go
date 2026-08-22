package queue

import (
	"encoding/json"

	"goscrapy/internal/model"
)

const (
	ReadyKey   = "queue:ready"
	LeaseKey   = "queue:lease"
	PayloadKey = "queue:payload"
	MetaPrefix = "queue:task:"
	ReclaimKey = "queue:reclaim_total"
)

func payloadField(jobID string) string { return jobID }

func taskSetKey(taskID int64) string {
	return MetaPrefix + itoa(taskID)
}

func encodeJob(j *model.CrawlJob) (string, error) {
	b, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeJob(raw string) (*model.CrawlJob, error) {
	var j model.CrawlJob
	if err := json.Unmarshal([]byte(raw), &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
