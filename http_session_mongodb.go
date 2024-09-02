package main

import (
	"context"
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"
)

type HttpSessionMongoDBSession struct {
	Id          primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Data        string             `bson:"data"`
	ExpireAfter time.Time          `bson:"expire_after"`
}

type HttpSessionMongoDB struct {
	Codecs     []securecookie.Codec
	Options    *sessions.Options
	Collection *mongo.Collection
}

func (s *HttpSessionMongoDB) MaxLength(l int) {
	for _, c := range s.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

func (s *HttpSessionMongoDB) MaxAge(age int) {
	s.Options.MaxAge = age

	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

func NewHttpSessionMongoDB(c *mongo.Collection, age int, keyPairs ...[]byte) *HttpSessionMongoDB {
	cs := &HttpSessionMongoDB{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: age,
		},
		Collection: c,
	}

	cs.MaxAge(cs.Options.MaxAge)
	return cs
}

func (s *HttpSessionMongoDB) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

func (s *HttpSessionMongoDB) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	var err error
	if c, err := r.Cookie(name); err == nil {
		if err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...); err == nil {
			err = s.load(session)
		}
	}
	return session, err
}

func (s *HttpSessionMongoDB) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = primitive.NewObjectID().Hex()
	}

	if err := s.save(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, s.Options))
	return nil
}

func (s *HttpSessionMongoDB) Delete(session *sessions.Session) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session ID provided")
	}

	deleteResult, err := s.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if deleteResult.DeletedCount == 0 {
		return errors.New("no session found with provided ID")
	}

	return nil
}

func (s *HttpSessionMongoDB) load(session *sessions.Session) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session ID provided")
	}

	var mongoSession HttpSessionMongoDBSession
	if err := s.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&mongoSession); err != nil {
		return err
	}

	if err := securecookie.DecodeMulti(session.Name(), mongoSession.Data, &session.Values, s.Codecs...); err != nil {
		return err
	}

	return nil
}

func (s *HttpSessionMongoDB) save(session *sessions.Session) error {
	ctx := context.Background()
	objectID, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return errors.New("invalid session ID provided")
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}

	mongoSession := HttpSessionMongoDBSession{
		Id:          objectID,
		Data:        encoded,
		ExpireAfter: time.Now().Add(time.Second * time.Duration(s.Options.MaxAge)),
	}

	if _, err := s.Collection.UpdateOne(ctx, bson.M{"_id": mongoSession.Id}, bson.M{"$set": mongoSession}, options.Update().SetUpsert(true)); err != nil {
		return err
	}

	return nil
}
