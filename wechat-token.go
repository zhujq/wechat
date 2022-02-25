package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devfeel/dotweb"
	"github.com/tidwall/buntdb"
)

const AccessTokenAPI = "https://api.weixin.qq.com/cgi-bin/token"

type Token struct {
	AccessToken string `json:"access_token"`
	Expire      int    `json:"expires_in"`
}

type App struct {
	Accounts map[string]string
	DB       *buntdb.DB
	Web      *dotweb.DotWeb
	WxToken  *Token
}

type Account struct {
	AppID  string `json:"appid"`
	Secret string `json:"secret"`
}

type ResBody struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
}

var message = ResBody{
	Status:      "failed",
	AccessToken: "",
}

func NewApp() *App {
	var a = &App{}
	a.Accounts = make(map[string]string)
	a.Web = dotweb.New()
	a.WxToken = new(Token)

	return a
}

// 读取配置文件中的appid和secret值到一个map中
func (a *App) SetAccounts(config *string) {
	accounts := make([]Account, 1)

	if _, err := os.Stat(*config); err != nil {
		fmt.Println("配置文件无法打开！")
		os.Exit(1)
	}

	raw, err := ioutil.ReadFile(*config)
	if err != nil {
		fmt.Println("配置文件读取失败！")
		os.Exit(1)
	}

	if err := json.Unmarshal(raw, &accounts); err != nil {
		fmt.Println("配置文件内容错误！")
		os.Exit(1)
	}

	for _, acc := range accounts {
		a.Accounts[acc.AppID] = acc.Secret
	}
}

func (a *App) Query(appid string, key string) string {
	var value string

	err := a.DB.View(func(tx *buntdb.Tx) error {
		v, err := tx.Get(appid + "_" + key)
		if err != nil {
			return err
		}
		value = v
		return nil
	})
	if err != nil {
		value = ""
	}

	return value
}

// 更新AppID上下文环境中的Access Token和到期时间
func (a *App) UpdateToken(appid string) {
	timestamp := time.Now().Unix()

	a.DB.Update(func(tx *buntdb.Tx) error {
		tx.Delete(appid + "_timestamp")
		tx.Delete(appid + "_access_token")
		tx.Delete(appid + "_expires_in")

		tx.Set(appid+"_timestamp", strconv.FormatInt(timestamp, 10), nil)
		tx.Set(appid+"_access_token", a.WxToken.AccessToken, nil)
		tx.Set(appid+"_expires_in", strconv.Itoa(a.WxToken.Expire), nil)
		return nil
	})
}

func tokenHandler(ctx dotweb.Context) error {
	appid := ctx.QueryString("appid")
	if appid == "" {
		log.Println("ERROR: 没有提供AppID参数")
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}

	if secret, isExist := app.Accounts[appid]; isExist {
		var access_token string
		var record_time string
		var expires_in string

		// 查询数据库中是否已经存在这个AppID的access_token
		record_time = app.Query(appid, "timestamp")
		access_token = app.Query(appid, "access_token")
		expires_in = app.Query(appid, "expires_in")
		expire_time, _ := strconv.ParseInt(record_time, 10, 64)
		timeout, _ := strconv.ParseInt(expires_in, 10, 64)

		if access_token != "" {
			// 如果数据库中已经存在了Token，就检查过期时间，如果过期了就去GetToken获取
			curTime := time.Now().Unix()
			if curTime >= expire_time+timeout {
				token := app.WxToken.Get(appid, secret)
				// 没获得access_token就返回Failed消息
				if token == "" {
					log.Println("ERROR: 没有获得access_token.")
					return ctx.WriteJsonC(http.StatusNotFound, message)
				}

				//获取Token之后更新运行时环境，然后返回access_token
				app.UpdateToken(appid)
				message.AccessToken = app.WxToken.AccessToken
			} else {
				message.AccessToken = access_token
			}
		} else {
			token := app.WxToken.Get(appid, secret)
			if token == "" {
				log.Println("ERROR: 没有获得access_token.")
				return ctx.WriteJsonC(http.StatusNotFound, message)
			}
			app.UpdateToken(appid)
			message.AccessToken = app.WxToken.AccessToken
		}

		message.Status = "success"
		return ctx.WriteJson(message)
	}

	log.Println("ERROR: AppID不存在")
	// 如果提交的appid不在配置文件中，就返回Failed消息
	return ctx.WriteJsonC(http.StatusNotFound, message)
}

func InitRoute(server *dotweb.HttpServer) {
	// 定义Basic Auth的用户名和密码用来防止接口被恶意访问

	server.GET("/token", tokenHandler)
}

// 获取AppID的access_token
func (t *Token) Get(appid string, secret string) string {

	requestLine := strings.Join([]string{AccessTokenAPI, "?grant_type=client_credential&appid=", appid, "&secret=", secret}, "")

	resp, err := http.Get(requestLine)
	//	res, _ := grequests.Get(AccessTokenAPI, ro)

	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if err := json.Unmarshal(body, t); err != nil {
		return ""
	}

	return string(body)
}

var app = NewApp()

func main() {
	var err error

	var (
		version = flag.Bool("version", false, "version v1.0")
		config  = flag.String("config", "account.json", "config file.")
		port    = flag.Int("port", 8880, "listen port.")
	)

	flag.Parse()

	if *version {
		fmt.Println("v0.1")
		os.Exit(0)
	}

	app.SetAccounts(config)
	app.DB, err = buntdb.Open("wechat.db")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer app.DB.Close()

	InitRoute(app.Web.HttpServer)
	log.Println("Start AccessToken Server on ", *port)
	app.Web.StartServer(*port)
}
