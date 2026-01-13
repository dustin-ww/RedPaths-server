package core

import "time"

type Assertion struct {
	Predicate  string    `json:"assertion.predicate"`
	Method     string    `json:"assertion.method"`
	Source     string    `json:"assertion.source"`
	Confidence float64   `json:"assertion.confidence"`
	Status     string    `json:"assertion.status"`
	Timestamp  time.Time `json:"assertion.timestamp"`
	SubjectUID string    `json:"assertion.subject_uid"`
	ObjectUID  string    `json:"assertion.object_uid"`
}
