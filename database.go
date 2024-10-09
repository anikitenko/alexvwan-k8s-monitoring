package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/BurntSushi/toml"
	logger "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"reflect"
)

type DatabaseHelper struct {
	db *mongo.Database
}

func NewDatabaseHelper(db *mongo.Database) *DatabaseHelper {
	return &DatabaseHelper{
		db: db,
	}
}

func InitDatabaseConnection() (*mongo.Client, string) {
	var config Config
	ctx := context.Background()

	if _, err := toml.DecodeFile(ConfigFile, &config); err != nil {
		logger.Fatalf("Failed to parse config file: %v", err)
	}

	var mongoCredentials options.Credential
	mongoCredentials.AuthMechanism = config.Database.AuthMechanism
	mongoCredentials.AuthSource = config.Database.AuthSource
	mongoCredentials.Username = config.Database.Username
	mongoCredentials.Password = config.Database.Password
	mongoCredentials.PasswordSet = config.Database.PasswordSet
	mongoOptions := options.Client().SetAppName("k8s-monitoring").SetAuth(mongoCredentials)
	mongoOptions = mongoOptions.SetCompressors(config.Database.Compressors).SetHosts(config.Database.Hosts).SetReplicaSet(config.Database.ReplicaSet)
	mongoOptions = mongoOptions.SetRetryWrites(config.Database.RetryWrites).SetDirect(config.Database.Direct).SetZlibLevel(config.Database.ZlibLevel)

	rootCerts := x509.NewCertPool()
	for _, file := range config.Database.RootCAs {
		if ca, err := os.ReadFile(file); err == nil {
			rootCerts.AppendCertsFromPEM(ca)
		} else {
			logger.Warnf("Unable to add Root CA file during establishing connection to database: %v", err)
		}
	}

	if config.Database.CertificateFile != "" && config.Database.CertificateKeyFile != "" {
		var clientCerts []tls.Certificate
		if cert, err := tls.LoadX509KeyPair(config.Database.CertificateFile, config.Database.CertificateKeyFile); err == nil {
			clientCerts = append(clientCerts, cert)
		}
		mongoOptions = mongoOptions.SetTLSConfig(&tls.Config{
			RootCAs:      rootCerts,
			Certificates: clientCerts,
		})
	}

	if err := mongoOptions.Validate(); err != nil {
		logger.Fatalf("Failed to validate database settings: %v", err)
	}

	client, err := mongo.Connect(ctx, mongoOptions)
	if err != nil {
		logger.Fatalf("Failed to create new database client from provided config: %v", err)
	}

	return client, config.Database.DatabaseName
}

func (dh *DatabaseHelper) FindOne(collectionName string, filter bson.M, result interface{}) error {
	res := dh.db.Collection(collectionName).FindOne(context.TODO(), filter)
	if err := res.Decode(result); err != nil {
		return err
	}

	return nil
}

func (dh *DatabaseHelper) FindAll(collectionName string, filter bson.M, results interface{}) error {
	cur, err := dh.db.Collection(collectionName).Find(context.TODO(), filter)
	if err != nil {
		return err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			logger.Warnf("Failed to close mongo cursor: %v", err)
		}
	}(cur, context.Background())

	resultsVal := reflect.ValueOf(results)
	if resultsVal.Kind() != reflect.Ptr || resultsVal.Elem().Kind() != reflect.Slice {
		logger.Warnf("results argument must be a slice pointer")
	}
	resultsVal = resultsVal.Elem()

	elemType := resultsVal.Type().Elem()
	for cur.Next(context.Background()) {
		elem := reflect.New(elemType).Interface() // create new type instance
		err := cur.Decode(elem)
		if err != nil {
			return err
		}
		resultsVal.Set(reflect.Append(resultsVal, reflect.ValueOf(elem).Elem())) // append to slice
	}

	if err := cur.Err(); err != nil {
		return err
	}

	return nil
}

func (dh *DatabaseHelper) FindGtLtLimit(collectionName string, field string, value interface{}, greater bool, equals bool, limit int, results interface{}) error {
	var filter bson.M
	if greater {
		if equals {
			filter = bson.M{field: bson.M{"$gte": value}}
		} else {
			filter = bson.M{field: bson.M{"$gt": value}}
		}
	} else {
		if equals {
			filter = bson.M{field: bson.M{"$lte": value}}
		} else {
			filter = bson.M{field: bson.M{"$lt": value}}
		}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{field, -1}}).SetLimit(int64(limit))

	cur, err := dh.db.Collection(collectionName).Find(context.TODO(), filter, findOptions)
	if err != nil {
		return err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			logger.Warnf("Failed to close mongo cursor: %v", err)
		}
	}(cur, context.Background())

	resultsVal := reflect.ValueOf(results)
	if resultsVal.Kind() != reflect.Ptr || resultsVal.Elem().Kind() != reflect.Slice {
		logger.Warnf("results argument must be a slice pointer")
	}
	resultsVal = resultsVal.Elem()

	elemType := resultsVal.Type().Elem()
	for cur.Next(context.Background()) {
		elem := reflect.New(elemType).Interface() // create new type instance
		err := cur.Decode(elem)
		if err != nil {
			return err
		}
		resultsVal.Set(reflect.Append(resultsVal, reflect.ValueOf(elem).Elem())) // append to slice
	}

	if err := cur.Err(); err != nil {
		return err
	}

	return nil
}

func (dh *DatabaseHelper) InsertOne(collectionName string, data interface{}) error {
	_, err := dh.db.Collection(collectionName).InsertOne(context.TODO(), data)
	return err
}

func (dh *DatabaseHelper) DeleteOne(collectionName string, filter bson.M) error {
	_, err := dh.db.Collection(collectionName).DeleteOne(context.Background(), filter)
	return err
}
