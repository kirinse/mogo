package bongo

import (
	"errors"
	// "github.com/maxwellhealth/dotaccess"
	"github.com/oleiade/reflections"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	// "log"
	// "strings"
)

// Relation types (one-to-many or one-to-one)
const (
	REL_MANY = iota
	REL_ONE  = iota
)

// Configuration to tell Bongo how to cascade data to related documents on save or delete
type CascadeConfig struct {
	// The collection to cascade to
	Collection *mgo.Collection

	// The relation type (does the target doc have an array of these docs [REL_MANY] or just reference a single doc [REL_ONE])
	RelType int

	// The property on the related doc to populate
	ThroughProp string

	// The query to find related docs
	Query bson.M

	// The data that constructs the query may have changed - this is to remove self from previous relations
	OldQuery bson.M

	// Data to cascade. Can be in dot notation
	Properties []string
}

// Cascades a document's properties to related documents, after it has been prepared
// for db insertion (encrypted, etc)
func CascadeSave(doc interface{}, preparedForSave map[string]interface{}) {
	// Find out which properties to cascade
	if conv, ok := doc.(interface {
		GetCascade() []*CascadeConfig
	}); ok {
		toCascade := conv.GetCascade()

		for _, conf := range toCascade {
			cascadeSaveWithConfig(conf, preparedForSave)
		}
	}
}

// Deletes references to a document from its related documents
func CascadeDelete(doc interface{}) {
	// Find out which properties to cascade
	if conv, ok := doc.(interface {
		GetCascade() []*CascadeConfig
	}); ok {
		toCascade := conv.GetCascade()

		// Get the ID
		id, err := reflections.GetField(doc, "Id")

		if err != nil {
			panic(err)
		}

		// Cast as bson.ObjectId
		if bsonId, ok := id.(bson.ObjectId); ok {
			for _, conf := range toCascade {
				cascadeDeleteWithConfig(conf, bsonId)
			}
		}

	}
}

// Runs a cascaded delete operation with one configuration
func cascadeDeleteWithConfig(conf *CascadeConfig, id bson.ObjectId) (*mgo.ChangeInfo, error) {
	switch conf.RelType {
	case REL_ONE:
		update := map[string]map[string]interface{}{
			"$set": map[string]interface{}{},
		}

		update["$set"][conf.ThroughProp] = nil

		return conf.Collection.UpdateAll(conf.Query, update)
	case REL_MANY:
		update := map[string]map[string]interface{}{
			"$pull": map[string]interface{}{},
		}

		update["$pull"][conf.ThroughProp] = bson.M{
			"_id": id,
		}
		return conf.Collection.UpdateAll(conf.Query, update)
	}

	return &mgo.ChangeInfo{}, errors.New("Invalid relation type")
}

// Runs a cascaded save operation with one configuration
func cascadeSaveWithConfig(conf *CascadeConfig, preparedForSave map[string]interface{}) (*mgo.ChangeInfo, error) {
	// Create a new map with just the props to cascade

	id := preparedForSave["_id"]

	data := make(map[string]interface{})
	// Set the id field automatically
	data["_id"] = id

	for _, prop := range conf.Properties {
		data[prop] = preparedForSave[prop]
	}

	switch conf.RelType {
	case REL_ONE:

		if len(conf.OldQuery) > 0 {

			update1 := map[string]map[string]interface{}{
				"$set": map[string]interface{}{},
			}

			update1["$set"][conf.ThroughProp] = nil
			conf.Collection.UpdateAll(conf.OldQuery, update1)
		}

		update := map[string]map[string]interface{}{
			"$set": map[string]interface{}{},
		}

		update["$set"][conf.ThroughProp] = data

		// Just update
		return conf.Collection.UpdateAll(conf.Query, update)
	case REL_MANY:
		update1 := map[string]map[string]interface{}{
			"$pull": map[string]interface{}{},
		}

		update1["$pull"][conf.ThroughProp] = bson.M{
			"_id": id,
		}

		if len(conf.OldQuery) > 0 {
			conf.Collection.UpdateAll(conf.OldQuery, update1)
		}

		// Remove self from current relations, so we can replace it
		conf.Collection.UpdateAll(conf.Query, update1)

		update2 := map[string]map[string]interface{}{
			"$push": map[string]interface{}{},
		}
		update2["$push"][conf.ThroughProp] = data

		return conf.Collection.UpdateAll(conf.Query, update2)

	}

	return &mgo.ChangeInfo{}, errors.New("Invalid relation type")

}