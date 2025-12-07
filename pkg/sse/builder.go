package sse

import "RedPaths-server/pkg/model/events"

type EventBuilder struct {
	eventType  events.EventType
	projectUID string
	payload    map[string]interface{}
}

func NewEvent(eventType events.EventType) *EventBuilder {
	return &EventBuilder{
		eventType: eventType,
		payload:   make(map[string]interface{}),
	}
}

func (eb *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	eb.payload[key] = value
	return eb
}

func (eb *EventBuilder) WithPayload(payload map[string]interface{}) *EventBuilder {
	eb.payload = payload
	return eb
}

func (eb *EventBuilder) Log(logger *SSELogger) {
	logger.Event(eb.eventType, eb.payload)
}
