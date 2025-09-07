package common

import (
	"context"

	"github.com/google/uuid"
	uxid "github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
)

// 传入明文生成XID
func GenerateXid(name string) string {
	xidNS := uuid.NewSHA1(uuid.NameSpaceURL, []byte("xid-protocol"))
	// normalized := strings.ToLower(strings.TrimSpace(name))
	xid := uuid.NewSHA1(xidNS, []byte(name))
	return xid.String()
}

// generates unique ID
func GenerateID() string {
	uxid := uxid.New()
	return uxid.String()
}

// generates unique UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// generates SHA1 of the text
func GenerateSHA1(text string) string {
	xidNS := uuid.NewSHA1(uuid.NameSpaceURL, []byte(text))
	xid := uuid.NewSHA1(xidNS, []byte(text))
	return xid.String()
}

// check xid is exists
func CheckXidExists(collection string, xid string) bool {
	return GetCollection(collection).FindOne(context.Background(), bson.M{"xid": xid}).Err() == nil
}
