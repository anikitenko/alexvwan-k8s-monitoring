package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Config struct {
	Database DatabaseConfig `toml:"database" json:"database"`
	Log      LogConfig      `toml:"log" json:"log"`
}

type DatabaseConfig struct {
	Compressors        []string `toml:"compressors" json:"compressors"`
	Hosts              []string `toml:"hosts" json:"hosts"`
	DatabaseName       string   `toml:"database_name" json:"database_name"`
	ReplicaSet         string   `toml:"replica_set" json:"replica_set"`
	RetryWrites        bool     `toml:"retry_writes" json:"retry_writes"`
	Direct             bool     `toml:"direct_connection" json:"direct"`
	ZlibLevel          int      `toml:"zlib_level" json:"zlib_level"`
	AuthMechanism      string   `toml:"auth_authentication_mechanism" json:"auth_authentication_mechanism"`
	AuthSource         string   `toml:"auth_authentication_source" json:"auth_authentication_source"`
	Username           string   `toml:"auth_username" json:"auth_username"`
	Password           string   `toml:"auth_password" json:"auth_password"`
	PasswordSet        bool     `toml:"auth_is_password_set" json:"auth_is_password_set"`
	RootCAs            []string `toml:"tls_root_CAs" json:"tls_root_CAs"`
	CertificateFile    string   `toml:"tls_certificate_file" json:"tls_certificate_file"`
	CertificateKeyFile string   `toml:"tls_certificate_key_file" json:"tls_certificate_key_file"`
}

type LogConfig struct {
	MaxSize    int  `toml:"max_size" json:"max_size"`
	MaxBackups int  `toml:"max_backups" json:"max_backups"`
	MaxAge     int  `toml:"max_age" json:"max_age"`
	LocalTime  bool `toml:"local_time" json:"local_time"`
	Compress   bool `toml:"compress" json:"compress"`
}

type DataSecureSessionKey struct {
	SecureSessionKey []byte `bson:"secure_session_key"`
}

type Kubeconfig struct {
	ID      primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Name    string             `bson:"name" json:"name"`
	Content string             `bson:"content" json:"content"`
}

type KubeConfigParsed struct {
	ID       string                     `json:"id"`
	Name     string                     `json:"name"`
	Clusters []KubeConfigClustersParsed `json:"clusters"`
}

type KubeConfigClustersParsed struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Server string `json:"server"`
}

type ApiSimpleResponse struct {
	ID   string `json:"id"`
	Item string `json:"item"`
}

type LogActivity struct {
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
}
