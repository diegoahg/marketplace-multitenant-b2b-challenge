package mongo

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	d "marketplace/internal/domain"
	"time"
)

func (s *Store) DeliveryState(ctx context.Context, id, kind string) (d.DeliveryState, error) {
	var doc struct {
		Effects map[string]d.DeliveryState `bson:"effects"`
	}
	err := s.DB.Collection("processed_events").FindOne(ctx, bson.M{"eventId": id}).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		return d.DeliveryState{}, nil
	}
	return doc.Effects[kind], err
}

// Terminal status and durable DLQ insertion commit together, before the source ACK.
func (s *Store) FailDelivery(ctx context.Context, letter d.DeadLetter, owner string, next time.Time, terminal bool) error {
	session, err := s.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		p := "effects." + letter.Destination
		set := bson.M{p + ".failures": letter.Attempts, p + ".nextAttempt": next, p + ".lastError": letter.Error}
		if terminal {
			set[p+".dead"] = true
		}
		r, e := s.DB.Collection("processed_events").UpdateOne(tx, bson.M{"eventId": letter.Event.EventID, p + ".owner": owner}, bson.M{"$set": set, "$unset": bson.M{p + ".owner": "", p + ".until": ""}})
		if e != nil {
			return nil, e
		}
		if r.MatchedCount != 1 {
			return nil, d.Fail("EFFECT_BUSY", "effect lease lost")
		}
		if terminal {
			_, e = s.DB.Collection("dead_letters").UpdateOne(tx, bson.M{"_id": letter.ID}, bson.M{"$setOnInsert": letter}, options.UpdateOne().SetUpsert(true))
		}
		return nil, e
	})
	return err
}

func (s *Store) ClaimAlert(ctx context.Context, owner string, now time.Time) (d.DeadLetter, bool, error) {
	var letter d.DeadLetter
	f := bson.M{"emailedAt": bson.M{"$exists": false}, "$and": bson.A{
		bson.M{"$or": bson.A{bson.M{"until": bson.M{"$exists": false}}, bson.M{"until": bson.M{"$lte": now}}}},
		bson.M{"$or": bson.A{bson.M{"nextMail": bson.M{"$exists": false}}, bson.M{"nextMail": bson.M{"$lte": now}}}},
	}}
	err := s.DB.Collection("dead_letters").FindOneAndUpdate(ctx, f, bson.M{"$set": bson.M{"owner": owner, "until": now.Add(time.Minute)}}, options.FindOneAndUpdate().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetReturnDocument(options.After)).Decode(&letter)
	if errors.Is(err, driver.ErrNoDocuments) {
		return letter, false, nil
	}
	return letter, err == nil, err
}

func (s *Store) FinishAlert(ctx context.Context, id, owner string, sent bool, message string) error {
	set := bson.M{"nextMail": time.Now().UTC().Add(30 * time.Second), "mailError": message}
	if sent {
		set["emailedAt"] = time.Now().UTC()
	}
	r, err := s.DB.Collection("dead_letters").UpdateOne(ctx, bson.M{"_id": id, "owner": owner}, bson.M{"$set": set, "$unset": bson.M{"owner": "", "until": ""}})
	if err == nil && r.MatchedCount != 1 {
		return d.Fail("EFFECT_BUSY", "mail lease lost")
	}
	return err
}
