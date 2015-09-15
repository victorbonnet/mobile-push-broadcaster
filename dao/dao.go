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

var APNSFileLock sync.RWMutex
var APNSMemLock sync.RWMutex

var APNSSandboxFileLock sync.RWMutex
var APNSSandboxMemLock sync.RWMutex

var gcm_tokens map[string][]string = make(map[string][]string)
var apns_tokens map[string][]string = make(map[string][]string)
var apns_sandbox_tokens map[string][]string = make(map[string][]string)

//GCM
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
			go persistGCM(gcm_tokens)
			return
		}
	}
	log.Println("No Token to remove: " + token)
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

//APNS
func AddAPNSToken(app string, token string) {
	APNSMemLock.Lock()
	defer APNSMemLock.Unlock()
	for _, element := range apns_tokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	apns_tokens[app] = append(apns_tokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	go persistAPNS(apns_tokens)
}

func RemoveAPNSToken(app string, token string) {
	APNSMemLock.Lock()
	defer APNSMemLock.Unlock()
	for i, element := range apns_tokens[app] {
		if token == element {
			apns_tokens[app] = append(apns_tokens[app][:i], apns_tokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			go persistAPNS(apns_tokens)
			return
		}
	}
	log.Println("No Token to remove: " + token)
}

func GetAPNSTokens(app string) []string {
	APNSMemLock.RLock()
	defer APNSMemLock.RUnlock()
	return apns_tokens[app]
}

func GetNbAPNSTokens(app string) int {
	APNSMemLock.RLock()
	defer APNSMemLock.RUnlock()
	return len(apns_tokens[app])
}

func persistAPNS(tokens map[string][]string) {
	APNSFileLock.Lock()
	defer APNSFileLock.Unlock()
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
	APNSFileLock.RLock()
	defer APNSFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/apns.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &apns_tokens)
	if err != nil {
		panic(er)
	}
}

//APNS Sandbox
func AddAPNSSandboxToken(app string, token string) {
	APNSSandboxMemLock.Lock()
	defer APNSSandboxMemLock.Unlock()
	for _, element := range apns_sandbox_tokens[app] {
		if token == element {
			log.Println("Token already registered: " + token + " for the app: " + app)
			return
		}
	}
	apns_sandbox_tokens[app] = append(apns_sandbox_tokens[app], token)
	log.Println("Token added: " + token + " for the app: " + app)

	go persistAPNSSandbox(apns_sandbox_tokens)
}

func RemoveAPNSSandboxToken(app string, token string) {
	APNSSandboxMemLock.Lock()
	defer APNSSandboxMemLock.Unlock()
	for i, element := range apns_sandbox_tokens[app] {
		if token == element {
			apns_sandbox_tokens[app] = append(apns_sandbox_tokens[app][:i], apns_sandbox_tokens[app][i+1:]...)
			log.Println("Token removed: " + token)
			go persistAPNSSandbox(apns_sandbox_tokens)
			return
		}
	}
	log.Println("No Token to remove: " + token)
}

func GetAPNSSandboxTokens(app string) []string {
	APNSSandboxMemLock.RLock()
	defer APNSSandboxMemLock.RUnlock()
	return apns_sandbox_tokens[app]
}

func GetNbAPNSSandboxTokens(app string) int {
	APNSSandboxMemLock.RLock()
	defer APNSSandboxMemLock.RUnlock()
	return len(apns_sandbox_tokens[app])
}

func persistAPNSSandbox(tokens map[string][]string) {
	APNSSandboxFileLock.Lock()
	defer APNSSandboxFileLock.Unlock()
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
	APNSSandboxFileLock.RLock()
	defer APNSSandboxFileLock.RUnlock()
	// dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir := "./"

	b, err := ioutil.ReadFile(dir + "/apns_sandbox.db")
	if err != nil {
		return
	}
	er := json.Unmarshal(b, &apns_sandbox_tokens)
	if err != nil {
		panic(er)
	}
}
