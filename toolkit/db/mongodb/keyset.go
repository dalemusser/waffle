// toolkit/db/mongodb/keyset.go
package mongodb

import "go.mongodb.org/mongo-driver/bson"

// KeysetWindow composes a two-clause $or for stable keyset pagination on (field, _id).
// dir must be "lt" or "gt".
func KeysetWindow(field, dir, key string, id any) bson.M {
	switch dir {
	case "lt":
		return bson.M{"$or": []bson.M{
			{field: bson.M{"$lt": key}},
			{field: key, "_id": bson.M{"$lt": id}},
		}}
	default: // "gt"
		return bson.M{"$or": []bson.M{
			{field: bson.M{"$gt": key}},
			{field: key, "_id": bson.M{"$gt": id}},
		}}
	}
}
