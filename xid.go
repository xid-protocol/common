package common

import (
	"github.com/google/uuid"
	uxid "github.com/rs/xid"
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
