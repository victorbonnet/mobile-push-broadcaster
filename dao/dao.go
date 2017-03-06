package dao

import (
	"log"
	"fmt"
	"sync"
	"strings"
	"github.com/boltdb/bolt"
)

var gcmMemLock sync.RWMutex
var apnsMemLock sync.RWMutex
var apnsSandboxMemLock sync.RWMutex

var gcmTokens = make(map[string][]string)
var apnsTokens = make(map[string][]string)
var apnsSandboxTokens = make(map[string][]string)

var db *bolt.DB

func init() {
	db, _ = bolt.Open("broadcaster.db", 0600, nil)
    // defer db.Close()
}

func GetGCMTokens(app string) []string {
	gcmMemLock.RLock()
	defer gcmMemLock.RUnlock()
	return gcmTokens[app]
}

func GetNbGCMTokens(app string) int {
	gcmMemLock.RLock()
	defer gcmMemLock.RUnlock()
	return len(gcmTokens[app])
}

func AddGCMToken(app string, token string) {
	gcmMemLock.Lock()
	defer gcmMemLock.Unlock()
	for _, element := range gcmTokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	gcmTokens[app] = append(gcmTokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	saveTokenInDB("gcm", app, token)
}

func RemoveGCMToken(app string, token string) {
	gcmMemLock.Lock()
	defer gcmMemLock.Unlock()
	for i, element := range gcmTokens[app] {
		if token == element {
			gcmTokens[app] = append(gcmTokens[app][:i], gcmTokens[app][i+1:]...)
			deleteTokenInDB("gcm", app, token)
			log.Println("Token removed: " + token)
			return
		}
	}
	log.Println("No Token to remove: " + token)
}

func GetAPNSTokens(app string) []string {
	apnsMemLock.RLock()
	defer apnsMemLock.RUnlock()
	return apnsTokens[app]
}

func GetNbAPNSTokens(app string) int {
	apnsMemLock.RLock()
	defer apnsMemLock.RUnlock()
	return len(apnsTokens[app])
}

func AddAPNSToken(app string, token string) {
	apnsMemLock.Lock()
	defer apnsMemLock.Unlock()
	for _, element := range apnsTokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	apnsTokens[app] = append(apnsTokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	saveTokenInDB("apns", app, token)
}

func RemoveAPNSToken(app string, token string) {
	apnsMemLock.Lock()
	defer apnsMemLock.Unlock()
	for i, element := range apnsTokens[app] {
		if token == element {
			apnsTokens[app] = append(apnsTokens[app][:i], apnsTokens[app][i+1:]...)
			deleteTokenInDB("apns", app, token)
			log.Println("Token removed: " + token)
			return
		}
	}
	log.Println("No Token to remove: " + token)
}

func GetAPNSSandboxTokens(app string) []string {
	apnsSandboxMemLock.RLock()
	defer apnsSandboxMemLock.RUnlock()
	return apnsSandboxTokens[app]
}

func GetNbAPNSSandboxTokens(app string) int {
	apnsSandboxMemLock.RLock()
	defer apnsSandboxMemLock.RUnlock()
	return len(apnsSandboxTokens[app])
}

func AddAPNSSandboxToken(app string, token string) {
	for _, element := range apnsSandboxTokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	apnsSandboxTokens[app] = append(apnsSandboxTokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	saveTokenInDB("apnssandbox", app, token)
}

func RemoveAPNSSandboxToken(app string, token string) {
	for i, element := range apnsSandboxTokens[app] {
		if token == element {
			apnsSandboxTokens[app] = append(apnsSandboxTokens[app][:i], apnsSandboxTokens[app][i+1:]...)
			deleteTokenInDB("apnssandbox", app, token)
			log.Println("Token removed: " + token)
			return
		}
	}
	log.Println("No Token to remove: " + token)
}

func CreateBucket() *bolt.Bucket {
    var bucket bolt.Bucket
    err := db.Update(func(tx *bolt.Tx) error {
        b, err := tx.CreateBucketIfNotExists([]byte("tokens"))
        if err != nil {
            return fmt.Errorf("create bucket: %s", err)
        }
        bucket = *b
        return nil
    })
    if err != nil {
        return nil
    }
    return &bucket
}

func InitCache() {
	err := db.View(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte("tokens"))
        if bucket == nil {
            bucket = CreateBucket();
            if bucket == nil {
                fmt.Errorf("Bucket not found!")
            }
        }

        bucket.ForEach(func(k, v []byte) error {
            res := strings.Split(string(k), "#")
            if res[0] == "gcm" {
                    gcmTokens[res[1]] = append(gcmTokens[res[1]], res[2])
            } else if res[0] == "apns" {
                    apnsTokens[res[1]] = append(apnsTokens[res[1]], res[2])
            } else {
                    apnsSandboxTokens[res[1]] = append(apnsSandboxTokens[res[1]], res[2])
            }
            return nil
        })

        return nil
    })
    if err != nil {
    	panic(err)
    }
}

func saveTokenInDB(plateform string, app string, token string) {
    db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("tokens"))
        err := b.Put([]byte(plateform + "#" + app + "#" + token), []byte(token))
        return err
    })
}

func deleteTokenInDB(plateform string, app string, token string) {
    db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("tokens"))
        err := b.Delete([]byte(plateform + "#" + app + "#" + token))
        return err
    })
}
