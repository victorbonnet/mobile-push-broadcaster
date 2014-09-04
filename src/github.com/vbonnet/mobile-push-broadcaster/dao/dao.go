package dao

import (
	"encoding/json"
	"io/ioutil"
	"log"
	// "os"
	// "path/filepath"
	"sync"
)

var GCMFileLock sync.RWMutex
var GCMMemLock sync.RWMutex

var gcm_tokens map[string][]string = make(map[string][]string)
var apns_tokens map[string][]string = make(map[string][]string)

func AddGCMToken(app string, token string) {
	GCMMemLock.Lock()
	defer GCMMemLock.Unlock()
	for _, element := range gcm_tokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	gcm_tokens[app] = append(gcm_tokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	go persistGCM(gcm_tokens)
}

func RemoveGCMToken(app string, token string) {
	GCMMemLock.Lock()
	defer GCMMemLock.Unlock()
	for i, element := range gcm_tokens[app] {
		if token == element {
			gcm_tokens[app] = append(gcm_tokens[app][:i], gcm_tokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			return
		}
	}
	log.Println("No Token to remove: " + token)

	go persistGCM(gcm_tokens)
}

func GetGCMTokens(app string) []string {
	GCMMemLock.RLock()
	defer GCMMemLock.RUnlock()
	return gcm_tokens[app]
}

func GetNbGCMTokens(app string) int {
	GCMMemLock.RLock()
	defer GCMMemLock.RUnlock()
	return len(gcm_tokens[app])
}

func persistGCM(tokens map[string][]string) {
	GCMFileLock.Lock()
	defer GCMFileLock.Unlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"
	res, _ := json.Marshal(tokens)
	data := string(res)
	err := ioutil.WriteFile(dir+"/gcm.db", []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}

func LoadGCMFromStorage() {
	GCMFileLock.RLock()
	defer GCMFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/gcm.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &gcm_tokens)
	if err != nil {
		panic(er)
	}
}

//TODO APNS
func GetNbAPNSTokens(app string) int {
	return len(apns_tokens[app])
}
