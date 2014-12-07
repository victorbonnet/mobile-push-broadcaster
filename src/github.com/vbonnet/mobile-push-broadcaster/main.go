package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexjlockwood/gcm"
	"github.com/anachronistic/apns"
	"github.com/codegangsta/martini"
	"github.com/martini-contrib/render"
	"github.com/vbonnet/mobile-push-broadcaster/dao"
	"github.com/vbonnet/mobile-push-broadcaster/web_logs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type WebPageInfo struct {
	Server   string
	Port     string
	AppInfos []AppInfo
}

type AppInfo struct {
	Name              string
	AndroidDevices    int
	IOSDevices        int
	IOSSandboxDevices int
	Fields            []Field
}

type Field struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Tips  string `json:"tips"`
}

type AppSettings struct {
	Name            string  `json:"name"`
	GcmApiKey       string  `json:"gcm_api_key"`
	ApnsCert        string  `json:"apns_cert"`
	ApnsKey         string  `json:"apns_key"`
	ApnsCertSandbox string  `json:"apns_cert_sandbox"`
	ApnsKeySandbox  string  `json:"apns_key_sandbox"`
	Fields          []Field `json:"fields"`
}

var settings struct {
	SERVER string        `json:"server"`
	PORT   string        `json:"port"`
	Apps   []AppSettings `json:"apps"`
}

const MAX_GCM_TOKENS = 1000

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	// dir := "./"

	LoadConfig(dir)

	// Load tokens from Storage
	log.Println("Load the Tokens from Storage")
	dao.LoadGCMFromStorage()
	dao.LoadAPNSFromStorage()
	dao.LoadAPNSSandboxFromStorage()
	log.Println("Tokens loaded")

	m := martini.Classic()

	m.Use(render.Renderer(render.Options{
		Directory:  dir + "/web",
		Extensions: []string{".tmpl", ".html"},
		Charset:    "UTF-8",
		Delims:     render.Delims{"{[{", "}]}"},
		IndentJSON: false,
	}))
	m.Use(martini.Static(dir + "/web"))

	m.Get("/", Index)
	m.Get("/broadcast", Broadcast)

	// GCM
	m.Post("/gcm/register", RegisterGcm)
	m.Post("/gcm/unregister", UnregisterGcm)

	// APNS
	m.Post("/apns/register", RegisterApns)
	m.Post("/apns/unregister", UnregisterApns)
	m.Post("/apns/register_sandbox", RegisterApnsSandbox)
	m.Post("/apns/unregister_sandbox", UnregisterApnsSandbox)

	// websockets to display logs in the web page
	m.Get("/sock_gcm", web_logs.SockGCM)
	m.Get("/sock_apns", web_logs.SockAPNS)

	log.Fatal(http.ListenAndServe(":"+settings.PORT, m))
	m.Run()
}

func LoadConfig(dir string) {
	configFile, err := os.Open(dir + "/config.json")
	if err != nil {
		fmt.Errorf("opening config file", err.Error())
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&settings); err != nil {
		fmt.Errorf("parsing config file", err.Error())
	}
}

func GetAppConfig(app string) (AppSettings, error) {
	for _, element := range settings.Apps {
		if app == element.Name {
			return element, nil
		}
	}
	return AppSettings{}, errors.New("No app with the name: " + app)
}
func GetPageInfo() WebPageInfo {
	var webPageInfo WebPageInfo
	var appInfos []AppInfo
	for _, element := range settings.Apps {
		appInfo := AppInfo{element.Name, dao.GetNbGCMTokens(element.Name), dao.GetNbAPNSTokens(element.Name), dao.GetNbAPNSSandboxTokens(element.Name), element.Fields}
		appInfos = append(appInfos, appInfo)
	}
	webPageInfo.Server = settings.SERVER
	webPageInfo.Port = settings.PORT
	webPageInfo.AppInfos = appInfos
	return webPageInfo
}

func Index(render render.Render) {
	render.HTML(200, "broadcaster", GetPageInfo())
}

func Broadcast(render render.Render, w http.ResponseWriter, r *http.Request) {
	var params = make(map[string]interface{})
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	if params["app"] == nil {
		log.Println("app is not defined")
		return
	}

	if params["GCM"] == "true" {
		go SendGCM(params)
	}
	if params["APNS"] == "true" {
		go SendApns(params)
	}
	if params["APNSSandbox"] == "true" {
		go SendApnsSandbox(params)
	}
}

func RegisterGcm(r *http.Request) string {
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

func UnregisterGcm(r *http.Request) string {
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

func RegisterApns(r *http.Request) string {
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

func UnregisterApns(r *http.Request) string {
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

func RegisterApnsSandbox(r *http.Request) string {
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

func UnregisterApnsSandbox(r *http.Request) string {
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

func SendGCM(params map[string]interface{}) {
	var wg sync.WaitGroup
	t1 := time.Now()
	tokens := dao.GetGCMTokens(params["app"].(string))

	var reqNumber int = 0
	for i := 0; i < len(tokens); i = i + MAX_GCM_TOKENS {
		max := i + MAX_GCM_TOKENS
		if max >= len(tokens) {
			max = len(tokens)
		}
		reqNumber = reqNumber + 1
		log.Println("Send request " + strconv.Itoa(reqNumber) + " to the GCM server")
		wg.Add(1)
		go SendRequestToGCM(params, tokens[i:max], reqNumber, &wg)
	}

	wg.Wait()
	t2 := time.Now()
	var duration time.Duration = t2.Sub(t1)
	web_logs.GCMLogs("Notifications sent to " + strconv.Itoa(len(tokens)) + " Android devices in " + duration.String())
	log.Println("Notifications sent to " + strconv.Itoa(len(tokens)) + " Android devices in " + duration.String())
}
func SendRequestToGCM(data map[string]interface{}, toks []string, reqNumber int, wg *sync.WaitGroup) {
	tokens := make([]string, len(toks))
	copy(tokens, toks)

	t1 := time.Now()
	msg := gcm.NewMessage(data, tokens...)

	appSettings, app_error := GetAppConfig(data["app"].(string))
	if app_error != nil {
		return
	}
	sender := &gcm.Sender{ApiKey: appSettings.GcmApiKey}

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
	var duration time.Duration = t2.Sub(t1)
	web_logs.GCMLogs("Request " + strconv.Itoa(reqNumber) + " sent to " + strconv.Itoa(len(toks)) + " devices in " + duration.String())
	log.Println("Request " + strconv.Itoa(reqNumber) + " sent to " + strconv.Itoa(len(toks)) + " devices in " + duration.String())
	wg.Done()
}

func SendApns(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, app_error := GetAppConfig(app)
	if app_error != nil {
		return
	}

	payload := apns.NewPayload()
	payload.Alert = params["message"]
	payload.Badge = 42
	payload.Sound = "bingbong.aiff"

	client := apns.NewClient("gateway.push.apple.com:2195", appSettings.ApnsCert, appSettings.ApnsKey)

	tokens := dao.GetAPNSTokens(params["app"].(string))
	for i := 0; i < len(tokens); i = i + 1 {
		pn := apns.NewPushNotification()
		pn.DeviceToken = tokens[i]
		pn.AddPayload(payload)

		for key, value := range params {
			pn.Set(key, value)
		}

		resp := client.Send(pn)

		pn.PayloadString()
		alert, _ := pn.PayloadString()
		fmt.Println("  Alert:", alert)
		fmt.Println("Success:", resp.Success)
		fmt.Println("  Error:", resp.Error)

		web_logs.GCMLogs("Sent to " + strconv.Itoa(i) + " devices")

		if resp.Error != nil {
			go dao.RemoveAPNSToken(app, tokens[i])
		}
	}

	go ApnsFeedback(params)
}

func ApnsFeedback(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, app_error := GetAppConfig(app)
	if app_error != nil {
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

func SendApnsSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, app_error := GetAppConfig(app)
	if app_error != nil {
		return
	}

	payload := apns.NewPayload()
	payload.Alert = params["message"]
	payload.Badge = 42
	payload.Sound = "bingbong.aiff"

	client := apns.NewClient("gateway.sandbox.push.apple.com:2195", appSettings.ApnsCertSandbox, appSettings.ApnsKeySandbox)

	tokens := dao.GetAPNSSandboxTokens(params["app"].(string))
	for i := 0; i < len(tokens); i = i + 1 {
		pn := apns.NewPushNotification()
		pn.DeviceToken = tokens[i]
		pn.AddPayload(payload)

		for key, value := range params {
			pn.Set(key, value)
		}

		resp := client.Send(pn)

		pn.PayloadString()
		alert, _ := pn.PayloadString()
		fmt.Println("  Alert:", alert)
		fmt.Println("Success:", resp.Success)
		fmt.Println("  Error:", resp.Error)

		web_logs.GCMLogs("Sent to " + strconv.Itoa(i) + " devices")

		if resp.Error != nil {
			go dao.RemoveAPNSSandboxToken(app, tokens[i])
		}
	}

	go ApnsFeedbackSandbox(params)
}

func ApnsFeedbackSandbox(params map[string]interface{}) {
	app := params["app"].(string)
	appSettings, app_error := GetAppConfig(app)
	if app_error != nil {
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

	go ApnsFeedbackSandbox(params)
}
