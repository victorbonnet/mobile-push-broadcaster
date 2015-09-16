package dao

import (
	"encoding/json"
	"io/ioutil"
	"log"
	// "os"
	// "path/filepath"
	"sync"
)

var gcmFileLock sync.RWMutex
var gcmMemLock sync.RWMutex

var apnsFileLock sync.RWMutex
var apnsMemLock sync.RWMutex

var apnsSandboxFileLock sync.RWMutex
var apnsSandboxMemLock sync.RWMutex

var gcmTokens = make(map[string][]string)
var apnsTokens = make(map[string][]string)
var apnsSandboxTokens = make(map[string][]string)

//GCM
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

	go persistGCM(gcmTokens)
}

func RemoveGCMToken(app string, token string) {
	gcmMemLock.Lock()
	defer gcmMemLock.Unlock()
	for i, element := range gcmTokens[app] {
		if token == element {
			gcmTokens[app] = append(gcmTokens[app][:i], gcmTokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			go persistGCM(gcmTokens)
			return
		}
	}
	log.Println("No Token to remove: " + token)
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

func persistGCM(tokens map[string][]string) {
	gcmFileLock.Lock()
	defer gcmFileLock.Unlock()
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
	gcmFileLock.RLock()
	defer gcmFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/gcm.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &gcmTokens)
	if err != nil {
		panic(er)
	}
}

//APNS
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

	go persistAPNS(apnsTokens)
}

func RemoveAPNSToken(app string, token string) {
	apnsMemLock.Lock()
	defer apnsMemLock.Unlock()
	for i, element := range apnsTokens[app] {
		if token == element {
			apnsTokens[app] = append(apnsTokens[app][:i], apnsTokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			go persistAPNS(apnsTokens)
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

func persistAPNS(tokens map[string][]string) {
	apnsFileLock.Lock()
	defer apnsFileLock.Unlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"
	res, _ := json.Marshal(tokens)
	data := string(res)
	err := ioutil.WriteFile(dir+"/apns.db", []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}

func LoadAPNSFromStorage() {
	apnsFileLock.RLock()
	defer apnsFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/apns.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &apnsTokens)
	if err != nil {
		panic(er)
	}
}

//APNS Sandbox
func AddAPNSSandboxToken(app string, token string) {
	apnsSandboxMemLock.Lock()
	defer apnsSandboxMemLock.Unlock()
	for _, element := range apnsSandboxTokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	apnsSandboxTokens[app] = append(apnsSandboxTokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	go persistAPNSSandbox(apnsSandboxTokens)
}

func RemoveAPNSSandboxToken(app string, token string) {
	apnsSandboxMemLock.Lock()
	defer apnsSandboxMemLock.Unlock()
	for i, element := range apnsSandboxTokens[app] {
		if token == element {
			apnsSandboxTokens[app] = append(apnsSandboxTokens[app][:i], apnsSandboxTokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			go persistAPNSSandbox(apnsSandboxTokens)
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

func persistAPNSSandbox(tokens map[string][]string) {
	apnsSandboxFileLock.Lock()
	defer apnsSandboxFileLock.Unlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"
	res, _ := json.Marshal(tokens)
	data := string(res)
	err := ioutil.WriteFile(dir+"/apns_sandbox.db", []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}

func LoadAPNSSandboxFromStorage() {
	apnsSandboxFileLock.RLock()
	defer apnsSandboxFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/apns_sandbox.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &apnsSandboxTokens)
	if err != nil {
		panic(er)
	}
}
