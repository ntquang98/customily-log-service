package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Log struct {
	ID              primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Type            string             `json:"type,omitempty" bson:"type,omitempty"`
	Method          string             `json:"method,omitempty" bson:"method,omitempty"`
	URL             string             `json:"url,omitempty" bson:"url,omitempty"`
	StoreDomain     string             `json:"storeDomain,omitempty" bson:"store_domain,omitempty"`
	ScriptLoaded    bool               `json:"scriptLoaded,omitempty" bson:"script_loaded,omitempty"`
	CustomilyLoaded bool               `json:"customilyLoaded,omitempty" bson:"customily_loaded,omitempty"`
	AppLoaded       bool               `json:"appLoaded,omitempty" bson:"app_loaded,omitempty"`
	Body            string             `json:"body,omitempty" bson:"body,omitempty"`
	Timestamp       time.Time          `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
}
