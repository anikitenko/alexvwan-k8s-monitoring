package main

import "go.mongodb.org/mongo-driver/bson"

func BsonExists(field string) bson.M {
	return bson.M{field: bson.M{"$exists": true}}
}

// BsonGreaterThan Helper function for creating a '$gt' filter condition
func BsonGreaterThan(fieldName string, value interface{}) bson.M {
	return bson.M{fieldName: bson.M{"$gt": value}}
}

// BsonLessThan Helper function for creating a '$lt' filter condition
func BsonLessThan(fieldName string, value interface{}) bson.M {
	return bson.M{fieldName: bson.M{"$lt": value}}
}

// BsonCombineFilters Helper function for combining filter conditions
func BsonCombineFilters(filters ...bson.M) bson.M {
	result := bson.M{}
	for _, filter := range filters {
		for key, value := range filter {
			result[key] = value
		}
	}
	return result
}

// BsonEquals Helper function for creating a '$eq' filter condition
func BsonEquals(fieldName string, value interface{}) bson.M {
	return bson.M{fieldName: value}
}

// BsonFieldsEqual Helper function for creating 'equals' conditions for multiple fields
// valuesMap is a map where the key is the field name and the value is the field value
func BsonFieldsEqual(valuesMap map[string]interface{}) bson.M {
	filter := bson.M{}
	for key, value := range valuesMap {
		filter[key] = value
	}
	return filter
}
