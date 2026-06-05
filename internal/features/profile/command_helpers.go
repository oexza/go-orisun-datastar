package profile

import "github.com/oexza/go-orisun-datastar/internal/eventstore"

func extension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func profileEventQuery(eventType, idField, id string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: eventType},
		{Key: idField, Value: id},
	}}}}
}
