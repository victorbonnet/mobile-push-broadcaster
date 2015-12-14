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

	"mobile-push-broadcaster/apns"
	"mobile-push-broadcaster/dao"
	"mobile-push-broadcaster/web_logs"

	"github.com/alexjlockwood/gcm"
	"github.com/codegangsta/martini"
	"github.com/martini-contrib/auth"
	"github.com/martini-contrib/render"
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

func main() {
	staticFilesDir := "."
	if len(os.Args) > 1 {
		staticFilesDir = os.Args[1]
	}

	loadConfig(staticFilesDir)

	// Load tokens from Storage
	log.Println("Load the Tokens from Storage")
	dao.LoadGCMFromStorage()
	dao.LoadAPNSFromStorage()
	dao.LoadAPNSSandboxFromStorage()
	log.Println("Tokens loaded")

	m := martini.Classic()

	authenticator := auth.BasicFunc(func(username, password string) bool {
		return auth.SecureCompare(username, settings.Login) && auth.SecureCompare(password, settings.Password)
	})

	m.Use(render.Renderer(render.Options{
		Directory:  staticFilesDir + "/web",
		Extensions: []string{".tmpl", ".html"},
		Charset:    "UTF-8",
		Delims:     render.Delims{"{[{", "}]}"},
		IndentJSON: false,
	}))
	m.Use(martini.Static(staticFilesDir + "/web"))

	m.Get("/", authenticator, index)
	m.Get("/broadcast", authenticator, broadcast)

	// GCM
	m.Post("/gcm/register", registerGcm)
	m.Post("/gcm/unregister", unregisterGcm)

	// APNS
	m.Post("/apns/register", registerApns)
	m.Post("/apns/unregister", unregisterApns)
	m.Post("/apns/register_sandbox", registerApnsSandbox)
	m.Post("/apns/unregister_sandbox", unregisterApnsSandbox)

	// websockets to display logs in the web page
	m.Get("/sock_gcm", web_logs.SockGCM)
	m.Get("/sock_apns", web_logs.SockAPNS)

	log.Fatal(http.ListenAndServe(":"+settings.PORT, m))
	m.Run()
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

func index(render render.Render) {
	render.HTML(200, "broadcaster", getPageInfo())
}

func broadcast(render render.Render, w http.ResponseWriter, r *http.Request) {
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

func registerGcm(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("RegisterGcm: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Register GCM token: " + token)
	dao.AddGCMToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token saved\"}"
}

func unregisterGcm(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterGcm: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Unregister GCM token: " + token)
	dao.RemoveGCMToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token deleted\"}"
}

func registerApns(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("RegisterApns: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Register APNS token: " + token)
	dao.AddAPNSToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token saved\"}"
}

func unregisterApns(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterApns: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Unregister APNS token: " + token)
	dao.RemoveAPNSToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token deleted\"}"
}

func registerApnsSandbox(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	log.Println("app: " + app)
	if token == "" || app == "" {
		log.Println("RegisterApnsSandbox: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Register APNSSandbox token: " + token)
	dao.AddAPNSSandboxToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token saved\"}"
}

func unregisterApnsSandbox(r *http.Request) string {
	app := r.PostFormValue("app")
	token := r.PostFormValue("token")
	if token == "" || app == "" {
		log.Println("UnregisterApnsSandbox: app or token empty")
		return "{\"status\":\"error\",\"message\":\"app and token params are required\"}"
	}
	log.Println("Unregister APNSSandbox token: " + token)
	dao.RemoveAPNSSandboxToken(app, token)
	return "{\"status\":\"success\",\"message\":\"Token deleted\"}"
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

	payload := apns.NewPayload()
	payload.Alert = params["message"]
	payload.Badge = 42
	payload.Sound = "bingbong.aiff"

	client := apns.NewClient("gateway.push.apple.com:2195", appSettings.ApnsCert, appSettings.ApnsKey)

	tokens := dao.GetAPNSTokens(params["app"].(string))

	go apnsFeedback(params)

	web_logs.APNSLogs("Prepare notifications")
	var pushNotifications []*apns.PushNotification
	for i := 0; i < len(tokens); i = i + 1 {
		pn := apns.NewPushNotification()
		pn.DeviceToken = tokens[i]
		pn.AddPayload(payload)

		for key, value := range params {
			pn.Set(key, value)
		}

		pushNotifications = append(pushNotifications, pn)
	}

	web_logs.APNSLogs("Broadcasting to " + strconv.Itoa(len(tokens)) + " devices")
	err := client.Broadcast(pushNotifications)
	if err != nil {
		log.Println("Unable to broadcast apns: " + err.Error())
		fmt.Errorf("Error while broadcasting", err)
		web_logs.APNSLogs("Unable to push messages: " + err.Error())
	} else {
		web_logs.APNSLogs("Sent to " + strconv.Itoa(len(tokens)) + " devices")
	}
}

func apnsFeedback(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}
	fmt.Println("- connecting to check for deactivated tokens (maximum read timeout =", apns.FeedbackTimeoutSeconds, "seconds)")

	client := apns.NewClient("feedback.push.apple.com:2196", appSettings.ApnsCert, appSettings.ApnsKey)
	go client.ListenForFeedback()

	for {
		select {
		case resp := <-apns.FeedbackChannel:
			fmt.Println("- recv'd:", resp.DeviceToken)
			go dao.RemoveAPNSToken(app, resp.DeviceToken)
		case <-apns.ShutdownChannel:
			fmt.Println("- nothing returned from the feedback service")
		}
	}
}

func sendApnsSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}

	payload := apns.NewPayload()
	payload.Alert = params["message"]
	payload.Badge = 42
	payload.Sound = "bingbong.aiff"

	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", appSettings.ApnsCertSandbox, appSettings.ApnsKeySandbox)

	tokens := dao.GetAPNSSandboxTokens(params["app"].(string))

	go apnsFeedbackSandbox(params)

	web_logs.APNSLogs("Prepare notifications")
	var pushNotifications []*apns.PushNotification
	for i := 0; i < len(tokens); i = i + 1 {
		pn := apns.NewPushNotification()
		pn.DeviceToken = tokens[i]
		pn.AddPayload(payload)

		for key, value := range params {
			pn.Set(key, value)
		}

		pushNotifications = append(pushNotifications, pn)
	}

	web_logs.APNSLogs("Broadcasting to " + strconv.Itoa(len(tokens)) + " devices")
	err := client.Broadcast(pushNotifications)
	if err != nil {
		fmt.Println("Error while broadcasting", err)
	}
	web_logs.APNSLogs("Sent to " + strconv.Itoa(len(tokens)) + " devices")
}

func apnsFeedbackSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, appError := getAppConfig(app)
	if appError != nil {
		return
	}
	fmt.Println("- connecting to check for deactivated tokens (maximum read timeout =", apns.FeedbackTimeoutSeconds, "seconds)")

	client := apns.NewClient("feedback.sandbox.push.apple.com:2196", appSettings.ApnsCertSandbox, appSettings.ApnsKeySandbox)
	go client.ListenForFeedback()

	for {
		select {
		case resp := <-apns.FeedbackChannel:
			fmt.Println("- recv'd:", resp.DeviceToken)
			go dao.RemoveAPNSSandboxToken(app, resp.DeviceToken)
		case <-apns.ShutdownChannel:
			fmt.Println("- nothing returned from the feedback service")
		}
	}

	go apnsFeedbackSandbox(params)
}
