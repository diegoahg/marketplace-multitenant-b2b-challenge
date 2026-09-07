package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	d "marketplace/internal/domain"
	"time"
)

func (s *Store) Pending(ctx context.Context, limit int) ([]d.OrderConfirmedEvent, error) {
	// A broker acknowledgement is not proof that the destinations completed.
	// Reconcile older publications after one minute, preserving the original eventId.
	cutoff := time.Now().UTC().Add(-time.Minute)
	cur, e := s.DB.Collection("outbox_events").Aggregate(ctx, driver.Pipeline{
		{{Key: "$match", Value: bson.M{"$or": bson.A{bson.M{"publishedAt": bson.M{"$exists": false}}, bson.M{"publishedAt": bson.M{"$lte": cutoff}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "publishedAt", Value: 1}, {Key: "occurredAt", Value: 1}}}},
		{{Key: "$lookup", Value: bson.M{"from": "processed_events", "localField": "eventId", "foreignField": "eventId", "as": "progress"}}},
		{{Key: "$match", Value: bson.M{"$or": bson.A{bson.M{"progress.effects.erp.done": bson.M{"$ne": true}}, bson.M{"progress.effects.push.done": bson.M{"$ne": true}}}}}},
		{{Key: "$limit", Value: int64(limit)}},
		{{Key: "$unset", Value: "progress"}},
	})
	if e != nil {
		return nil, e
	}
	defer cur.Close(ctx)
	out := []d.OrderConfirmedEvent{}
	e = cur.All(ctx, &out)
	return out, e
}
func (s *Store) MarkPublished(ctx context.Context, id string) error {
	_, e := s.DB.Collection("outbox_events").UpdateOne(ctx, bson.M{"eventId": id}, bson.M{"$set": bson.M{"publishedAt": time.Now().UTC()}})
	return e
}
func (s *Store) ClaimEffect(ctx context.Context, event, kind, owner string, now time.Time) (bool, error) {
	col := s.DB.Collection("processed_events")
	_, e := col.UpdateOne(ctx, bson.M{"eventId": event}, bson.M{"$setOnInsert": bson.M{"eventId": event}}, options.UpdateOne().SetUpsert(true))
	if e != nil && !driver.IsDuplicateKeyError(e) {
		return false, e
	}
	prefix := "effects." + kind
	f := bson.M{"eventId": event, prefix + ".done": bson.M{"$ne": true}, "$or": bson.A{bson.M{prefix + ".until": bson.M{"$exists": false}}, bson.M{prefix + ".until": bson.M{"$lte": now}}}}
	r, e := col.UpdateOne(ctx, f, bson.M{"$set": bson.M{prefix + ".owner": owner, prefix + ".until": now.Add(time.Minute)}})
	if e != nil {
		return false, e
	}
	if r.MatchedCount == 1 {
		return true, nil
	}
	// A completed effect is a successful no-op; an active lease must be retried.
	e = col.FindOne(ctx, bson.M{"eventId": event, prefix + ".done": true}).Err()
	if e == nil {
		return false, nil
	}
	if e == driver.ErrNoDocuments {
		return false, d.Fail("EFFECT_BUSY", "effect is being processed")
	}
	return false, e
}
func (s *Store) CompleteEffect(ctx context.Context, event, kind, owner string) error {
	p := "effects." + kind
	r, e := s.DB.Collection("processed_events").UpdateOne(ctx, bson.M{"eventId": event, p + ".owner": owner}, bson.M{"$set": bson.M{p + ".done": true}, "$unset": bson.M{p + ".until": "", p + ".owner": ""}})
	if e == nil && r.MatchedCount != 1 {
		return d.Fail("EFFECT_BUSY", "effect lease lost")
	}
	return e
}
func (s *Store) ReleaseEffect(ctx context.Context, event, kind, owner string) error {
	p := "effects." + kind
	_, e := s.DB.Collection("processed_events").UpdateOne(ctx, bson.M{"eventId": event, p + ".owner": owner}, bson.M{"$unset": bson.M{p + ".until": "", p + ".owner": ""}})
	return e
}
