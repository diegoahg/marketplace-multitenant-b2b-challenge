package domain

import "time"

// Retries counts failed executions; one initial attempt plus five retries.
type DeliveryState struct {
	Failures    int       `bson:"failures" json:"failures"`
	NextAttempt time.Time `bson:"nextAttempt" json:"nextAttempt"`
	Done        bool      `bson:"done" json:"done"`
	Dead        bool      `bson:"dead" json:"dead"`
}

type DeadLetter struct {
	ID          string              `bson:"_id" json:"id"`
	Event       OrderConfirmedEvent `bson:"event" json:"event"`
	Destination string              `bson:"destination" json:"destination"`
	Error       string              `bson:"error" json:"error"`
	Raw         string              `bson:"raw" json:"raw"`
	Attempts    int                 `bson:"attempts" json:"attempts"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
}
