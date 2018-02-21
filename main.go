package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"

	"mobile-push-broadcaster/dao"
	"mobile-push-broadcaster/web_logs"

	"github.com/alexjlockwood/gcm"
	"github.com/timehop/apns"
)

type webPageInfo struct {
	Server   string
	Port     string
	AppInfos []appInfo
}

type appInfo struct {
	Name              string
	AndroidDevices    int
	IOSDevices        int
	IOSSandboxDevices int
	Fields            []field
}

type field struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Tips  string `json:"tips"`
}

type appSettings struct {
	Name            string  `json:"name"`
	GcmAPIKey       string  `json:"gcm_api_key"`
	ApnsCert        string  `json:"apns_cert"`
	ApnsKey         string  `json:"apns_key"`
	ApnsCertSandbox string  `json:"apns_cert_sandbox"`
	ApnsKeySandbox  string  `json:"apns_key_sandbox"`
	Fields          []field `json:"fields"`
}

var settings struct {
	Login    string        `json:"login"`
	Password string        `json:"password"`
	Server   string        `json:"server"`
	PORT     string        `json:"port"`
	Apps     []appSettings `json:"apps"`
}

const maxGcmTokens = 1000

var renderer = render.New()

func main() {
	staticFilesDir := "."
	if len(os.Args) > 1 {
		staticFilesDir = os.Args[1]
	}

	loadConfig(staticFilesDir)

	// Load tokens from Storage
	log.Println("Load the Tokens from Storage")
	dao.InitCache()
	log.Println("Tokens loaded")

	renderer = render.New(render.Options{
		Directory: staticFilesDir + "/web",
		Delims:    render.Delims{"{[{", "}]}"},
	})

	r := mux.NewRouter()

	r.HandleFunc("/", basicAuth(index)).Methods("GET")
	r.HandleFunc("/new", basicAuth(index2)).Methods("GET")
	r.HandleFunc("/broadcast", basicAuth(broadcast)).Methods("GET")

	r.HandleFunc("/gcm/register", registerGcm).Methods("POST")
	r.HandleFunc("/gcm/unregister", unregisterGcm).Methods("POST")
	r.HandleFunc("/apns/register", registerApns).Methods("POST")
	r.HandleFunc("/apns/unregister", unregisterApns).Methods("POST")
	r.HandleFunc("/apns/register_sandbox", registerApnsSandbox).Methods("POST")
	r.HandleFunc("/apns/unregister_sandbox", unregisterApnsSandbox).Methods("POST")
	r.HandleFunc("/sock_gcm", web_logs.SockGCM).Methods("GET")
	r.HandleFunc("/sock_apns", web_logs.SockAPNS).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticFilesDir + "/web"))).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(":"+settings.PORT, r)
}

func basicAuth(pass http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if ok && user == settings.Login && password == settings.Password {
			pass(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
		http.Error(w, "authorization failed", http.StatusUnauthorized)
	}
}

func loadConfig(staticFilesDir string) {
	configFile, err := os.Open(staticFilesDir + "/config.json")
	if err != nil {
		fmt.Errorf("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&settings); err != nil {
		fmt.Errorf("parsing config file", err.Error())
	}
}

func getAppConfig(app string) (appSettings, error) {
	for _, element := range settings.Apps {
		if app == element.Name {
			return element, nil
		}
	}
	return appSettings{}, errors.New("No app with the name: " + app)
}
func getPageInfo() webPageInfo {
	var webPageInfo webPageInfo
	var appInfos []appInfo
	for _, element := range settings.Apps {
		appInfo := appInfo{element.Name, dao.GetNbGCMTokens(element.Name), dao.GetNbAPNSTokens(element.Name), dao.GetNbAPNSSandboxTokens(element.Name), element.Fields}
		appInfos = append(appInfos, appInfo)
	}
	webPageInfo.Server = settings.Server
	webPageInfo.Port = settings.PORT
	webPageInfo.AppInfos = appInfos
	return webPageInfo
}

func index(w http.ResponseWriter, r *http.Request) {
	renderer.HTML(w, http.StatusOK, "broadcaster", getPageInfo())
}

func index2(w http.ResponseWriter, r *http.Request) {
	renderer.HTML(w, http.StatusOK, "broadcaster2", getPageInfo())
}

func broadcast(w http.ResponseWriter, r *http.Request) {
	var params = make(map[string]interface{})
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	if params["app"] == nil {
		log.Println("app is not defined")
		return
	}

	if params["GCM"] == "true" {
		go sendGcm(params)
	}
	if params["APNS"] == "true" {
		go sendApns(params)
	}
	if params["APNSSandbox"] == "true" {
		go sendApnsSandbox(params)
	}
}

func registerGcm(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("RegisterGcm: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Register GCM token: " + token)
	dao.AddGCMToken(app, token)

	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token saved"})
}

func unregisterGcm(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterGcm: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Unregister GCM token: " + token)
	dao.RemoveGCMToken(app, token)
	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token deleted"})
}

func registerApns(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("RegisterApns: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Register APNS token: " + token)
	dao.AddAPNSToken(app, token)
	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token saved"})
}

func unregisterApns(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterApns: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Unregister APNS token: " + token)
	dao.RemoveAPNSToken(app, token)
	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token deleted"})
}

func registerApnsSandbox(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	log.Println("app: " + app)
	if token == "" || app == "" {
		log.Println("RegisterApnsSandbox: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Register APNSSandbox token: " + token)
	dao.AddAPNSSandboxToken(app, token)
	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token saved"})
}

func unregisterApnsSandbox(w http.ResponseWriter, r *http.Request) {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterApnsSandbox: app or token empty")
		renderer.JSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "app and token params are required"})
		return
	}
	log.Println("Unregister APNSSandbox token: " + token)
	dao.RemoveAPNSSandboxToken(app, token)
	renderer.JSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Token deleted"})
}

func sendGcm(params map[string]interface{}) {
	var wg sync.WaitGroup
	t1 := time.Now()
	tokens := dao.GetGCMTokens(params["app"].(string))

	var reqNumber int
	for i := 0; i < len(tokens); i = i + maxGcmTokens {
		max := i + maxGcmTokens
		if max >= len(tokens) {
			max = len(tokens)
		}
		reqNumber = reqNumber + 1
		log.Println("Send request " + strconv.Itoa(reqNumber) + " to the GCM server")
		wg.Add(1)
		go sendRequestToGCM(params, tokens[i:max], reqNumber, &wg)
	}

	wg.Wait()
	t2 := time.Now()
	duration := t2.Sub(t1)
	web_logs.GCMLogs("Notifications sent to " + strconv.Itoa(len(tokens)) + " Android devices in " + duration.String())
	log.Println("Notifications sent to " + strconv.Itoa(len(tokens)) + " Android devices in " + duration.String())
}
func sendRequestToGCM(data map[string]interface{}, toks []string, reqNumber int, wg *sync.WaitGroup) {
	tokens := make([]string, len(toks))
	copy(tokens, toks)

	t1 := time.Now()
	msg := gcm.NewMessage(data, tokens...)

	appSettings, appError := getAppConfig(data["app"].(string))
	if appError != nil {
		return
	}
	sender := &gcm.Sender{ApiKey: appSettings.GcmAPIKey}

	// Send the message and receive the response after at most two retries.
	resp, err := sender.Send(msg, 2)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		web_logs.GCMLogs("ERROR: " + err.Error())
	}
	if resp != nil {
		res, _ := json.Marshal(resp)
		log.Println(string(res))

		if resp.Failure > 0 || resp.CanonicalIDs > 0 {
			var app = data["app"].(string)
			for index, el := range resp.Results {
				if el.Error != "" && el.RegistrationID == "" {
					go dao.RemoveGCMToken(app, tokens[index])
				} else if el.RegistrationID != "" {
					go dao.RemoveGCMToken(app, tokens[index])
					go dao.AddGCMToken(app, el.RegistrationID)
				}
			}
		}
	}

	t2 := time.Now()
	duration := t2.Sub(t1)
	web_logs.GCMLogs("Request " + strconv.Itoa(reqNumber) + " sent to " + strconv.Itoa(len(toks)) + " devices in " + duration.String())
	log.Println("Request " + strconv.Itoa(reqNumber) + " sent to " + strconv.Itoa(len(toks)) + " devices in " + duration.String())
	wg.Done()
}

func sendApns(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}

	c, err := apns.NewClientWithFiles(apns.ProductionGateway, appSettings.ApnsCert, appSettings.ApnsKey)
	if err != nil {
		fmt.Errorf("could not create new client", err.Error())
		web_logs.APNSLogs("Could not create new client")
		return
	}

	go func() {
		for f := range c.FailedNotifs {
			fmt.Println("Notif", f.Notif.ID, "failed with", f.Err.Error())
			web_logs.APNSLogs("Error with token " + f.Notif.DeviceToken + ", removed from database")
			dao.RemoveAPNSToken(app, f.Notif.DeviceToken)
		}
	}()

	tokens := dao.GetAPNSTokens(params["app"].(string))

	web_logs.APNSLogs("Broadcasting to " + strconv.Itoa(len(tokens)) + " devices")
	for i := 0; i < len(tokens); i = i + 1 {
		p := apns.NewPayload()
		p.APS.Alert.Body = params["message"].(string)
		p.APS.Sound = "bingbong.aiff"
		p.APS.ContentAvailable = 1
		for key, value := range params {
			p.SetCustomValue(key, value)
		}

		m := apns.NewNotification()
		m.Payload = p
		m.DeviceToken = tokens[i]
		m.Priority = apns.PriorityImmediate
		m.Identifier = 25167       // Integer for APNS
		m.ID = "user_id:timestamp" // ID not sent to Apple – to identify error notifications

		c.Send(m)
	}
	web_logs.APNSLogs("Sent to " + strconv.Itoa(len(tokens)) + " devices")

}

func apnsFeedback(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}

	f, err := apns.NewFeedback(apns.ProductionFeedbackGateway, appSettings.ApnsCert, appSettings.ApnsKey)
	if err != nil {
		log.Fatal("Could not create feedback", err.Error())
	}

	for ft := range f.Receive() {
		fmt.Println("Feedback for token:", ft.DeviceToken)
	}
}

func sendApnsSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}

	c, err := apns.NewClientWithFiles(apns.SandboxGateway, appSettings.ApnsCertSandbox, appSettings.ApnsKeySandbox)
	if err != nil {
		fmt.Errorf("could not create new client", err.Error())
		web_logs.APNSLogs("Could not create new client")
		return
	}

	//todo reuse code from sendApns()

	go func() {
		for f := range c.FailedNotifs {
			fmt.Println("Notif", f.Notif.ID, "failed with", f.Err.Error())
			web_logs.APNSLogs("Error with token " + f.Notif.DeviceToken + ", removed from database")
			dao.RemoveAPNSToken(app, f.Notif.DeviceToken)
		}
	}()

	tokens := dao.GetAPNSTokens(params["app"].(string))

	web_logs.APNSLogs("Broadcasting to " + strconv.Itoa(len(tokens)) + " devices")
	for i := 0; i < len(tokens); i = i + 1 {
		p := apns.NewPayload()
		p.APS.Alert.Body = params["message"].(string)
		p.APS.Sound = "bingbong.aiff"
		p.APS.ContentAvailable = 1
		for key, value := range params {
			p.SetCustomValue(key, value)
		}

		m := apns.NewNotification()
		m.Payload = p
		m.DeviceToken = tokens[i]
		m.Priority = apns.PriorityImmediate
		m.Identifier = 25167       // Integer for APNS
		m.ID = "user_id:timestamp" // ID not sent to Apple – to identify error notifications

		c.Send(m)
	}
	web_logs.APNSLogs("Sent to " + strconv.Itoa(len(tokens)) + " devices")
}

func apnsFeedbackSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}

	f, err := apns.NewFeedback(apns.SandboxFeedbackGateway, appSettings.ApnsCert, appSettings.ApnsKey)
	if err != nil {
		log.Fatal("Could not create feedback", err.Error())
	}

	for ft := range f.Receive() {
		fmt.Println("Feedback for token:", ft.DeviceToken)
	}
}
