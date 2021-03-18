package main

import (
	"crypto/sha1"
	"encoding/xml"
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"os"
	"bytes"
	"flag"
	"strconv"
	"net"
	"unicode"
	"github.com/garyburd/redigo/redis"
	"unicode/utf8"
	"github.com/enescakir/emoji"
	"net/url"
)

const (
	token = "wechat4go"
)

const GetTokenUrl = "http://token.zhujq.ga:1080/token?appid=wxf183d5e1fe4d5204"
const GetMaterialSum = "https://api.weixin.qq.com/cgi-bin/material/get_materialcount?access_token="
const GetMaterial = "https://api.weixin.qq.com/cgi-bin/material/batchget_material?access_token="
const GetMediainfo = "https://api.weixin.qq.com/cgi-bin/material/get_material?access_token="
const GetIndexUrl = "http://127.0.0.1:8080/?keyword="
//const GetIndexUrl = "https://wechat-index-wechat-zhujq.cloud.okteto.net/?keyword=" 把index放在同一个docker中部署
const WelcomeMsg =  "谢谢您的关注！[微笑]\n      “一只猪一世界”个人公众号主要用来记录本人体验这大千世界的所见、所听、所想、所思，内容完善中，您可以输入 help 或 帮助 获得使用帮助，输入about me 或 关于我 获得本公众号的详细说明，也可以任意输入看看有没好玩的。\n       由于本公众号是个人性质的订阅号，腾讯公司只赋予非常有限的权限，只能进行你问我答式的消息回复，回复的内容是有且只有一条文本（或图片或视频或图文）。\n       特别说明：本公众号后端搭建涉及的所有硬件、软件以及公众号呈现的内容均与本人所供职的公司（Z公司）无关，也无任何涉及Z公司知识产权或商业机密的内容呈现!\n       Best Wishes!\n                                                Zhujq [猪头]"
const GetIpinfoUrl = "http://ip-api.com/json/"
const GetInntelnuminfoUrl ="http://mobsec-dianhua.baidu.com/dianhua_api/open/location?tel="
const GetOuttelnuminfoUrl ="https://api.veriphone.io/v2/verify?key=0F0466BD7808436AB6F68930B8324802&phone="
const GetHeadnewsUrl = "https://api.isoyu.com/api/News/banner"
const CommMsg = "找不到什么东东回你了......"
const GetEntocnUrl = "http://fanyi.youdao.com/translate?&doctype=json&type=AUTO&i="
//const RedisDB = "wechat-redis:6379"
//const RedisDB = "redis-12069.c1.us-east1-2.gce.cloud.redislabs.com:12069"
//const RedisPWD ="Juju1234"
var  RedisDB,RedisPWD string
//const RedisPWD ="bZbvrprPKsz7ttNxanwYGSDhMgNXQdfy"

type TextRequestBody struct {                    //请求结构，需要解析xml后才能赋值给它
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string
	FromUserName string
	CreateTime   time.Duration
	MsgType      string
	Content      string
	MsgId        int
	Event		 string       //处理订阅事件时增加
	PicUrl       string       //处理用户发送图片时增加
	Recognition  string       //处理用户发送语音时增加
}

type TextResponseBody struct {                   //文本响应结构，需要用xml编码后才能http发送
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	Content      CDATAText
}

type ImageResponseBody struct {                   //图片响应结构，需要用xml编码后才能http发送
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	ImageMediaid CDATAText   `xml:"Image>MediaId"`
}

type VoiceResponseBody struct {                   //音频响应结构，需要用xml编码后才能http发送
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	VoiceMediaid CDATAText   `xml:"Voice>MediaId"`
}


type VideoResponseBody struct {                   //视频响应结构，需要用xml编码后才能http发送
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	VideoMediaid CDATAText   `xml:"Video>MediaId"`
	VideoTitle CDATAText   `xml:"Video>Title"`
	VideoDesc CDATAText   `xml:"Video>Description"`
}

type NewsResponseBody struct {                   //图文响应结构，需要用xml编码后才能http发送
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	ArticleCount  int
	NewsTitle  CDATAText `xml:"Articles>item>Title"`
	NewsDesc  CDATAText `xml:"Articles>item>Description"`
	NewsPicurl  CDATAText `xml:"Articles>item>PicUrl"`
	NewsUrl  CDATAText `xml:"Articles>item>Url"`
}


type Token struct {
	AccessToken string `json:"access_token"`
}

type ResBody struct {
	Status      string `json:"status"`
	Mediatype   string `json:"mediatype"`
	Mediaid     string `json:"mediaid"`
	Mediatitle  string `json:"mediatitle"`
	Mediaurl    string `json:"mediaurl"`
	Mediadigest string `json:"mediadigest"`
	Mediathumb  string `json:"mediathumb"`
}

type ResIpinfoBody struct {
	Status      string `json:"status"`
	Country   	string `json:"country"`
	RegionName  string `json:"regionName"`
	City  string `json:"city"`
	Isp    string `json:"isp"`
	As string `json:"as"`
}

type ResInnertelephoneinfoBody struct {
	Status int
	Location string 
}

type ResOuttelephoneinfoBody struct {
	Status      string `json:"status"`
	Phonetype  string `json:"phone_type"`
	Phoneregion  string `json:"phone_region"`
	Country    string `json:"country"`
	Carrier string `json:"carrier"`
}

type HeadnewsinfoBody struct {
	Source      string `json:"source"`
	Title  string `json:"title"`
}

type TransRsp struct {
	Type            string `json:"type"`
	ErrorCode       int    `json:"errorCode"`
	ElapsedTime     int    `json:"elapsedTime"`
	TranslateResult [][]struct {
		Src string `json:"src"`
		Tgt string `json:"tgt"`
	} `json:"translateResult"`
}

type CDATAText struct {
	Text string `xml:",innerxml"`
}

type MediaVideoinfo struct {
	Mediaid string
	Title string 
	Desc  string 
	Url   string 
}

type MediaNewsinfo struct {
	Mediaid string
	Title string 
	Desc  string 
	Picurl string
	Url   string 
}

/*
func init() {                                         //初始，日志文件生成
	file := "./" +"log"+ ".txt"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
			panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile) // 将文件设置为log输出的文件
	log.SetPrefix("[wechat]")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	return
}
日志输出到控制台*/

func makeSignature(timestamp, nonce string) string {
	sl := []string{token, timestamp, nonce}
	sort.Strings(sl)
	s := sha1.New()
	io.WriteString(s, strings.Join(sl, ""))
	return fmt.Sprintf("%x", s.Sum(nil))
}

func validateUrl(w http.ResponseWriter, r *http.Request) bool {
	timestamp := strings.Join(r.Form["timestamp"], "")
	nonce := strings.Join(r.Form["nonce"], "")
	signatureGen := makeSignature(timestamp, nonce)

	signatureIn := strings.Join(r.Form["signature"], "")
	if signatureGen != signatureIn {
		return false
	}
	log.Println("signature check pass!")                        //日志记录签名通过
	echostr := strings.Join(r.Form["echostr"], "")
	fmt.Fprintf(w, echostr)                                    //echostr作为body返回给微信公众服务器，只在接入鉴权时带echostr
	return true
}

func parseTextRequestBody(r *http.Request) *TextRequestBody {   //读取http请求中的body部分赋值给TextRequestBody
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
		return nil
	}
//	fmt.Println(string(body))
	log.Println(string(body))                                  //收到的body写入日志文件
	requestBody := &TextRequestBody{}
	xml.Unmarshal(body, requestBody)
	return requestBody
}

func value2CDATA(v string) CDATAText {
	//return CDATAText{[]byte("<![CDATA[" + v + "]]>")}
	return CDATAText{"<![CDATA[" + v + "]]>"}
}

func makeTextResponseBody(fromUserName, toUserName, content string) ([]byte, error) {    //赋值TextResponseBody后用xml编码
	textResponseBody := &TextResponseBody{}
	textResponseBody.FromUserName = value2CDATA(fromUserName)
	textResponseBody.ToUserName = value2CDATA(toUserName)
	textResponseBody.MsgType = value2CDATA("text")
	textResponseBody.Content = value2CDATA(content)
	textResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(textResponseBody, " ", "  ")
}

func makeImageResponseBody(fromUserName, toUserName, imageid string) ([]byte, error) {    //赋值ImageResponseBody后用xml编码
	imageResponseBody := &ImageResponseBody{}
	imageResponseBody.FromUserName = value2CDATA(fromUserName)
	imageResponseBody.ToUserName = value2CDATA(toUserName)
	imageResponseBody.MsgType = value2CDATA("image")
	imageResponseBody.ImageMediaid = value2CDATA(imageid)
	imageResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(imageResponseBody, " ", "  ")
}

func makeVoiceResponseBody(fromUserName, toUserName, voiceid string) ([]byte, error) {    //赋值VoiceResponseBody后用xml编码
	voiceResponseBody := &VoiceResponseBody{}
	voiceResponseBody.FromUserName = value2CDATA(fromUserName)
	voiceResponseBody.ToUserName = value2CDATA(toUserName)
	voiceResponseBody.MsgType = value2CDATA("voice")
	voiceResponseBody.VoiceMediaid = value2CDATA(voiceid)
	voiceResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(voiceResponseBody, " ", "  ")
}

func makeVideoResponseBody(fromUserName string, toUserName string, videoinfo MediaVideoinfo) ([]byte, error) {    //赋值VideoResponseBody后用xml编码
	videoResponseBody := &VideoResponseBody{}
	videoResponseBody.FromUserName = value2CDATA(fromUserName)
	videoResponseBody.ToUserName = value2CDATA(toUserName)
	videoResponseBody.MsgType = value2CDATA("video")
	videoResponseBody.VideoMediaid = value2CDATA(videoinfo.Mediaid)
	videoResponseBody.VideoTitle = value2CDATA(videoinfo.Title)
	videoResponseBody.VideoDesc = value2CDATA(videoinfo.Desc)
	videoResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(videoResponseBody, " ", "  ")
}

func makeNewsResponseBody(fromUserName string, toUserName string, newsinfo MediaNewsinfo) ([]byte, error) {    //赋值NewsResponseBody后用xml编码
	newsResponseBody := &NewsResponseBody{}
	newsResponseBody.FromUserName = value2CDATA(fromUserName)
	newsResponseBody.ToUserName = value2CDATA(toUserName)
	newsResponseBody.MsgType = value2CDATA("news")
	newsResponseBody.ArticleCount = 1
	newsResponseBody.NewsTitle = value2CDATA(newsinfo.Title)
	newsResponseBody.NewsDesc = value2CDATA(newsinfo.Desc)
	newsResponseBody.NewsPicurl = value2CDATA(newsinfo.Picurl)
	newsResponseBody.NewsUrl = value2CDATA(newsinfo.Url)
	newsResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(newsResponseBody, " ", "  ")

}

func IsNumber(str string) (bool) {    //判断字符串是否全是数字	
	for _, r := range str {
		if unicode.IsNumber(r) == false{
			return false
		}
	}
	return true
}

func isEven(num int) bool {
	if num%2 == 0 {
		return true
	}
	return false
}

func FilterEmoji(content string) string {        //过滤字符串中的emoj
    new_content := ""
    for _, value := range content {
        _, size := utf8.DecodeRuneInString(string(value))
        if size <= 3 {
            new_content += string(value)
        }
    }
    return new_content
}  

//HTTPGet get 请求
func HTTPGet(uri string) ([]byte, error) {
	response, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http get error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return ioutil.ReadAll(response.Body)
}

func httpClient() *http.Client {
	return &http.Client{ }
}

//HTTPPost post 请求
func HTTPPost(uri string, data string) ([]byte, error) {
	body := bytes.NewBuffer([]byte(data))
	response, err := http.Post(uri, "", body)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http get error : uri=%v , statusCode=%v", uri, response.StatusCode)
	}
	return ioutil.ReadAll(response.Body)
}

//PostJSON post json 数据请求
func PostJson(uri string, obj interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient().Post(uri, "application/json;charset=utf-8", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http post error : uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func procRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if !validateUrl(w, r) {
		log.Println("Wechat Service: this http request is not from Wechat platform!")
		return
	}

	if r.Method == "GET" {  //get方法只有接入鉴权时用，所以第一步vlidateUrl后无需再处理
		return
	}

	if r.Method !=  "POST" {  //按照规范，应该收到POST信息，如果不是，直接返回SUCCESS
		fmt.Fprintf(w, string("success"))
		return
	}
	
	textRequestBody := parseTextRequestBody(r)
	var rsp ResBody
	responseBody := make([]byte, 0)

	if textRequestBody.Event == "subscribe" {  //订阅事件处理

		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,WelcomeMsg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}

	if textRequestBody.Event == "unsubscribe" {  //取消订阅事件处理

		redisconn.Do("DEL", textRequestBody.FromUserName)
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,"")
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}
	

	if textRequestBody.MsgType == "image"{

		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,"您好，目前只能识别文本消息，请重新输入文本。")
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
		
	} 

	if textRequestBody.MsgType == "voice"{
		                                      
		if textRequestBody.Recognition != ""{                        //如果有语音识别结果，按照文本输入方式处理
			textRequestBody.Content = textRequestBody.Recognition
			textRequestBody.MsgType = "text"
		}else{
			responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,"听不清你的语音，麻烦再说一次。")
			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprintf(w, string(responseBody))
		return
		}

	} 

	if textRequestBody.MsgType != "text" {   //收到非文本消息

		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,"您好，目前只能识别文本和语音消息，请重新输入。")
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}

	if textRequestBody == nil || textRequestBody.Content =="" || textRequestBody.Content == " " {  //空内容时直接返回
		fmt.Fprintf(w, string("success"))
		return
	}

	if strings.Contains(textRequestBody.Content,"【收到不支持的消息类型，暂无法显示】"){ //接收到微信公众平台发的微信平台不识别信息（如自定义表情）
		fmt.Fprintf(w, string("success"))
		return
	}

//	fmt.Printf("Wechat Service: Recv text msg [%s] from user [%s]!",textRequestBody.Content,textRequestBody.FromUserName)

	_, err := redisconn.Do("lpush", textRequestBody.FromUserName,textRequestBody.Content)  //记录用户输入的信息

	if err != nil {                                   //有可能redis连接断了，重新连接
		log.Println("Error to connnect redis,re-connect....")

		redisconn, err = redis.Dial("tcp", RedisDB,redis.DialKeepAlive(time.Hour*48),redis.DialPassword(RedisPWD))  //连接redis数据库，记录用户文本记录和预处理
		if err != nil {                                   //如果无法连接redis数据库，不返回继续处理
        log.Println("Connect to redis error", err)
        
		}
		redisconn.Do("lpush", textRequestBody.FromUserName,textRequestBody.Content)
//		defer redisconn.Close()

	}
	
	redisconn.Do("INCR", "keywordtimes:"+ textRequestBody.Content)		        //统计用户输入信息的次数并放到有序集合中。
	var keytimes int64 = 0
	keytimes,_ = redis.Int64(redisconn.Do("GET", "keywordtimes:"+ textRequestBody.Content))
	if keytimes > 0 {
		redisconn.Do("ZADD", "keywordalltimes",keytimes, textRequestBody.Content)	
	}

	textRequestBody.Content = strings.TrimSpace(textRequestBody.Content)   //去掉首尾空格

	if strings.Contains(textRequestBody.Content,`/:`) && len(textRequestBody.Content) >= 4 {  //接收到含表情符号时回填表情符号,表情符号可能为/: 或[]

		index := strings.Index(textRequestBody.Content,`/:`)  
		msg := []byte((textRequestBody.Content)[index:len(textRequestBody.Content)])
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,string(msg[:]))
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
	}

	if strings.Contains(textRequestBody.Content,`[`) && strings.Contains(textRequestBody.Content,`]`) && len(textRequestBody.Content) > 4 {  

		index := strings.Index(textRequestBody.Content,`[`)  
		msg := []byte((textRequestBody.Content)[index:len(textRequestBody.Content)])
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,string(msg[:]))
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
	}

	textRequestBody.Content = FilterEmoji(textRequestBody.Content)   //接收到unicode表情符号时去掉表情符号

	if  textRequestBody.Content == "" {       //如果去除unicode表情符号后为空，就返回随机表情符号，
		textRequestBody.Content = "give me a "		
	}else{
		log.Println("New request content:"+textRequestBody.Content)
	}

	

	if strings.HasPrefix(textRequestBody.Content ,"给我一首诗") || strings.HasPrefix(textRequestBody.Content ,"give me a poem"){
		
		tempstr := strings.Replace(textRequestBody.Content, "给我一首诗", "", -1 ) 
		tempstr = strings.Replace(tempstr, "give me a poem", "", -1 ) 
		if tempstr == ""{
			msg, err := redis.String(redisconn.Do("SRANDMEMBER", "poemsall"))
			log.Println(msg)
			if err != nil{
				log.Println("SRANDMEMBER err",err.Error())
				msg = "发生错误了，默认发送：\n[唐]李白\n《早发白帝城》\n朝辞白帝彩云间，千里江陵一日还。\n两岸猿声啼不住，轻舟已过万重山。\n"
			}
			responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
			w.Header().Set("Content-Type", "text/xml")
			fmt.Fprintf(w, string(responseBody))
			return
		}else{
			if strings.HasPrefix(tempstr ,"+"){

				buff, err := HTTPGet(GetIndexUrl+url.QueryEscape("poem:"+tempstr))  //需要把用户输入的关键字用html格式编码，否则空格等不能传递
				if err != nil{
					log.Println("error:", err)
					//看看本地redis有没有
					temp, _ := redis.String(redisconn.Do("SRANDMEMBER", ("keyword:"+textRequestBody.Content)))
					if temp != ""{
						log.Println("Get Index Error,but Get rsp from redis!")
			
					}
					buff = []byte(temp)
					json.Unmarshal(buff,&rsp)

				}else{
					//这里增加keyword:用户输入内容为key值，value为返回的json数据的 集合sadd操作，可以作为一种容灾方式
					redisconn.Do("SADD", ("keyword:"+textRequestBody.Content),string(buff))
					err := json.Unmarshal(buff,&rsp)
		
					if err != nil {			
						log.Println("error:", err)
					}
				}
				if rsp.Status == "success" &&  rsp.Mediatype == "poem" {
					responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,rsp.Mediadigest)
					
				}else{
					commreply, _ := redis.String(redisconn.Do("SRANDMEMBER", "commtoreply"))
					if commreply == ""{
						commreply = CommMsg
					}	
					responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":disappointed_face:")+commreply))
				}

				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprintf(w, string(responseBody))
				return
			}
		}
	} 

	if textRequestBody.Content == "给我一条新闻" || textRequestBody.Content == "给我新闻" || textRequestBody.Content == "give me a news"{

		var newsinfo HeadnewsinfoBody
		buff, _ := HTTPGet(GetHeadnewsUrl)
		json.Unmarshal(buff,&newsinfo)
		msg := emoji.Parse(":newspaper:") + newsinfo.Source + "：" + newsinfo.Title 
		log.Println(msg)
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	} 
													//用户发give me a时自动补尾空格
	if textRequestBody.Content == "give me a" {
		textRequestBody.Content = "give me a "
	}
	if textRequestBody.Content == "give me an" {
		textRequestBody.Content = "give me an "
	}

	if strings.HasPrefix(textRequestBody.Content,"give me a ") || strings.HasPrefix(textRequestBody.Content,"give me an "){   //返回emoj

		msg := strings.Replace(textRequestBody.Content, "give me an ", "", -1 ) 
		msg = strings.Replace(textRequestBody.Content, "give me a ", "", -1 )
		if msg =="" {
			msg, _ = redis.String(redisconn.Do("SRANDMEMBER", "emojisall"))
			log.Println("ramdom emoji:"+msg)
		}
		
		msg = emoji.Parse(":"+msg+":")
		log.Println(msg)
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}


	if strings.HasPrefix(textRequestBody.Content,"给我"){

		msg := strings.Replace(textRequestBody.Content, "给我", "", -1 ) 
		switch msg{
			case "一面国旗","国旗","一国旗":
				msg = "flag_for_china"
			case "一架飞机","飞机","一飞机":
				msg = "airplane"
			case "一只笔","笔":
				msg = "pencil2"
			case "一颗爱心","爱心","一颗心","心":
				msg = "beating_heart"
			case "一个火箭","火箭","一火箭":
				msg = "rocket"
			case "一辆公共汽车","公共汽车","公汽":
				msg = "bus"
			case "一个西红柿","西红柿","一西红柿":
				msg = "tomato"
			case "一个手机","手机":
				msg = "mobile_phone"
			case "一个100分","100","100分","一个100":
				msg = "100"
			case "一个天使","天使","一天使":
				msg = "angel"
			case "一本书","书":
				msg = "book"
			case "一颗炸弹","一个炸弹","一炸弹","炸弹":
				msg = "bomb"
			case "money","钱","一点钱","一点美元","一叠美元","美元","一张美元","some money":
				msg = "dollar"
			case "钻石","蓝钻","一颗钻石","一颗蓝钻":
				msg = "large_blue_diamond"
			case "一个吻","吻","亲吻","一个亲吻","kiss","a kiss":
				msg = "lips"
			case "一朵云","云","一云":
				msg = "cloud"
			case "一块饼干","饼干","一饼干","甜点","点心","一块甜点","一块点心":
				msg = "cookie"
			case "一副眼镜","眼镜","一眼镜":
				msg = "eyeglasses"
			case "","一个","一个随便":
				msg, _ = redis.String(redisconn.Do("SRANDMEMBER", "emojisall"))
			default:
				buff, err := HTTPGet(GetEntocnUrl+url.QueryEscape(textRequestBody.Content)) 
				if err != nil{
					log.Println(err.Error())
					msg, _ = redis.String(redisconn.Do("SRANDMEMBER", "emojisall"))
				}else{
					var trans TransRsp 
					json.Unmarshal(buff,&trans)
			        msg = trans.TranslateResult[0][0].Tgt
					if msg == ""{
						msg, _ = redis.String(redisconn.Do("SRANDMEMBER", "emojisall"))
					}else{
						sindex := strings.LastIndex(msg," ")
						if sindex == (len(msg) - 1){   //翻译返回结果最后为空格
							msg, _ = redis.String(redisconn.Do("SRANDMEMBER", "emojisall"))
						}else{
							lw := []byte(msg)[(sindex+1):len(msg)]  
							msg = string(lw[:])
						}
					}
				}
		}
		msg = emoji.Parse(":"+msg+":")
		log.Println(msg)
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}


	if (strings.Count(textRequestBody.Content,".") >= 1 || strings.Count(textRequestBody.Content,":") >= 2 ) && len(textRequestBody.Content) > 2 {  //可能是IPV4/6地址或域名

		ipaddress := net.ParseIP(textRequestBody.Content)  
				if ipaddress != nil {       //合法的IP地址，查询IP信息

					var ipinfo ResIpinfoBody
					ipinfo.Status = "fail"
					buff, _ := HTTPGet(GetIpinfoUrl+textRequestBody.Content)
					json.Unmarshal(buff,&ipinfo)
					
					if ipinfo.Status == "success" {
						msg := "IP地址"+textRequestBody.Content +"的信息： \n"+ "所在国家：" + ipinfo.Country + "\n所在地区：" + ipinfo.RegionName + "\n所在城市："+ ipinfo.City +"\nISP："+ ipinfo.Isp +"\nAS："+ ipinfo.As
						responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":surfing_man:")+msg))
						w.Header().Set("Content-Type", "text/xml")
						fmt.Fprintf(w, string(responseBody))
						return

					}else{

						responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":disappointed_face:")+"暂时获取不到IP地址信息，请稍后再试。"))
						w.Header().Set("Content-Type", "text/xml")
						fmt.Fprintf(w, string(responseBody))
						return
				
					}

				}else{  //可能是域名或者错误的IP地址格式或者含有.的纯文本

					 _,err := net.LookupHost(textRequestBody.Content)
					 
					if err == nil {   //能正常DNS解析认为是用户输入域名，查询信息
						var ipinfo ResIpinfoBody
						ipinfo.Status = "fail"
						buff, _ := HTTPGet(GetIpinfoUrl+textRequestBody.Content)
						json.Unmarshal(buff,&ipinfo)

						if ipinfo.Status == "success" {
							msg := "域名"+textRequestBody.Content +"的信息： \n"+ "所在国家：" + ipinfo.Country + "\n所在地区：" + ipinfo.RegionName + "\n所在城市："+ ipinfo.City +"\nISP："+ ipinfo.Isp +"\nAS："+ ipinfo.As
							responseBody, err = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":surfing_man:")+msg))
							w.Header().Set("Content-Type", "text/xml")
							fmt.Fprintf(w, string(responseBody))
							return
	
						}else{
	
							responseBody, err = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":disappointed_face:")+"暂时获取不到域名信息，请稍后再试。"))
							w.Header().Set("Content-Type", "text/xml")
							fmt.Fprintf(w, string(responseBody))
							return
					
						}

				 }
			}
	}
	
	if IsNumber(textRequestBody.Content)   {  //收到的首位为1的7位以上全数字或者86打头的9位以上全数字
		if  ( len(textRequestBody.Content) >= 7 && strings.HasPrefix(textRequestBody.Content,"1") ) || ( len(textRequestBody.Content) >= 9 && strings.HasPrefix(textRequestBody.Content,"861") ) {

			var teleinfo ResInnertelephoneinfoBody
			gettingnum := textRequestBody.Content
			
			if strings.HasPrefix(textRequestBody.Content,"86") {
				gettingnum = strings.TrimPrefix(textRequestBody.Content,"86")
			}

			buff, _ := HTTPGet(GetInntelnuminfoUrl+gettingnum)                         //用baidu提供的号码归属地查询接口
			js, _ := simplejson.NewJson(buff)
			
			teleinfo.Status = js.Get("responseHeader").Get("status").MustInt()
			teleinfo.Location = js.Get("response").Get(gettingnum).Get("location").MustString()

			if teleinfo.Status == 200 && teleinfo.Location != ""{
				responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,emoji.Parse(":telephone_receiver:")+textRequestBody.Content+"是"+teleinfo.Location+"的号码。")
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprintf(w, string(responseBody))
				return
			}else{
				responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,emoji.Parse(":disappointed_face:")+"暂时查询不到"+textRequestBody.Content+"的号码信息，请重新输入或者换个号码试试。")
				w.Header().Set("Content-Type", "text/xml")
				fmt.Fprintf(w, string(responseBody))
				return

			}
			
		}
	}

	if strings.HasPrefix(textRequestBody.Content,"+") && IsNumber(strings.ReplaceAll(textRequestBody.Content,"+","")){   //首位为＋的除首位外的全数字
		
		y,m,d := time.Now().Date()
		tempstr := "QueryIntTele:"+strconv.Itoa(y)+strconv.Itoa(int(m))+strconv.Itoa(d)
		isQuery,_ := redis.String(redisconn.Do("GET", tempstr))
		queriedCount := 0
		msg := ""

		if isQuery == ""{
				redisconn.Do("INCR", tempstr)
				redisconn.Do("EXPIRE", tempstr,86400)
		}else{
			queriedCount, _ =  strconv.Atoi(isQuery)
		}
			
		if queriedCount < 150 {
				
				var teleinfo ResOuttelephoneinfoBody
				teleinfo.Status = "fail"
				buff, _ := HTTPGet(GetOuttelnuminfoUrl+textRequestBody.Content)
				json.Unmarshal(buff,&teleinfo)

				redisconn.Do("INCR", tempstr)

				if teleinfo.Status == "success" {
					msg = emoji.Parse(":telephone_receiver:") + "号码"+textRequestBody.Content +"的信息： \n"+ "所在国家：" + teleinfo.Country + "\n所在地区：" + teleinfo.Phoneregion + "\n运营商：" + teleinfo.Carrier + "\n号码类型："+ teleinfo.Phonetype
				}else{
					msg = "查询失败，请换个国际号码试试。"
				}
		}else{

			msg = "国际号码查询信息次数已达上限，请明天再试"

		}

		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}

	if strings.HasPrefix(textRequestBody.Content,"T:") || strings.HasPrefix(textRequestBody.Content,"T：") || strings.HasPrefix(textRequestBody.Content,"t:") || strings.HasPrefix(textRequestBody.Content,"t："){  //翻译接口
		var msg string=""
		textRequestBody.Content = strings.Replace(textRequestBody.Content,"T:","",1)  //去掉前缀
		textRequestBody.Content = strings.Replace(textRequestBody.Content,"T：","",1)
		textRequestBody.Content = strings.Replace(textRequestBody.Content,"t:","",1)
		textRequestBody.Content = strings.Replace(textRequestBody.Content,"t：","",1)
		buff, err := HTTPGet(GetEntocnUrl+url.QueryEscape(textRequestBody.Content)) 
		if err != nil{
			log.Println(err.Error())
			msg  = (emoji.Parse(":disappointed_face:")+ "获取翻译接口失败，请稍后再使用。")
		}else{
			var trans TransRsp 
			trans.ErrorCode = 1   //先赋值为1，再用json解码后重新赋值
			json.Unmarshal(buff,&trans)		
			if trans.ErrorCode == 0 {
				msg = "翻译类型:" + trans.Type + "  翻译为：\n" 
				msg += trans.TranslateResult[0][0].Tgt
			}else{
				msg  = (emoji.Parse(":disappointed_face:")+ "很遗憾，翻译失败了，请稍后再使用。")					
			}
		}
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}

	if strings.HasPrefix(textRequestBody.Content,"su:") {          //管理功能，暂定以su:打头开始
		msg := ""
		switch textRequestBody.Content{
		case "su:get history":
			values,err := redis.Values(redisconn.Do("lrange",textRequestBody.FromUserName,0,50))
			if err != nil{
    			log.Println("lrange err",err.Error())
			}
			msg = "输入历史（最大前50个）是：\n"
			for _,v := range values{
				msg += string( v.([]byte) )
				msg += "\n"
			}
			
		case "su:delete history":
			redisconn.Do("DEL", textRequestBody.FromUserName)
			msg = "已清理历史记录"

		case "su:top":
			values,err := redis.Values(redisconn.Do("Zrevrangebyscore","keywordalltimes",999999,100,"WITHSCORES"))
			if err != nil{
    			log.Println("lrange err",err.Error())
			}
			msg = "输入最多的文本是：\n"
			for i,v := range values{
				msg += string( v.([]byte) )
				if isEven(i){
					msg += "："
				}else{
					msg += "\n"
				}
			}
		
		case "su:test":
			msg = `<a color="#5C3317">发送第一条消息</a>`

		default:
			msg = "不识别的管理指令，请检查你的输入。"

		}
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,msg)   //返回给微信服务器的响应必须一次发回
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return

	}

	matchedrst, err := redis.String(redisconn.Do("GET", textRequestBody.Content)) //查询是不是预定义的字符，如脏话、问好、查询时间等

	if err != nil && matchedrst != "" {                                  
        log.Println("Get redis error", err)
        
	}

	if matchedrst == "dirty"{
		torudereply, _ := redis.String(redisconn.Do("SRANDMEMBER", "rudetoreply"))
		if torudereply == ""{
			torudereply = "你要斯文一点哦"
		}
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":angry:") + torudereply))
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
	}

	if matchedrst == "hello"{
		tohelloreply, _ := redis.String(redisconn.Do("SRANDMEMBER", "hellotoreply"))
		if tohelloreply == ""{
			tohelloreply = "你好呀"
		}
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":smiling_face_with_smiling_eyes:") +tohelloreply))
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
	}

	if matchedrst == "time"{
		timestamp := time.Now().Unix()
		sample := "2006-01-02 15:04:05"
		strtime := emoji.Parse(":timer_clock:") + "小猪猪为您准确报时，当前时刻是：\n"
		strtime += (emoji.Parse(":flag_for_china:") + "北京:"+time.Unix(timestamp, 0).UTC().Add(8*time.Hour).Format(sample)+"\n")
		strtime += (emoji.Parse(":flag_for_united_states:") + "纽约: "+time.Unix(timestamp, 0).UTC().Add(-5*time.Hour).Format(sample)+"\n")
		strtime += (emoji.Parse(":flag_for_united_kingdom:") + "伦敦: "+time.Unix(timestamp, 0).UTC().Add(0*time.Hour).Format(sample)+"\n")
		strtime += (emoji.Parse(":flag_for_russia:") + "莫斯科:"+time.Unix(timestamp, 0).UTC().Add(3*time.Hour).Format(sample))
		responseBody, _ = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,strtime)
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, string(responseBody))
		return
	}

	if strings.Contains(textRequestBody.Content,"\n") {  //换行符替换成空格

		textRequestBody.Content = strings.Replace(textRequestBody.Content, "\n", " ", -1)

	}
	
	buff, err := HTTPGet(GetIndexUrl+url.QueryEscape(textRequestBody.Content))  //需要把用户输入的关键字用html格式编码，否则空格等不能传递
	if err != nil{
		log.Println("error:", err)
		//看看本地redis有没有
		temp, _ := redis.String(redisconn.Do("SRANDMEMBER", ("keyword:"+textRequestBody.Content)))
		if temp != ""{
			log.Println("Get Index Error,but Get rsp from redis!")
			
		}
		buff = []byte(temp)
		json.Unmarshal(buff,&rsp)

	}else{
		//这里增加keyword:用户输入内容为key值，value为返回的json数据的 集合sadd操作，可以作为一种容灾方式
		redisconn.Do("SADD", ("keyword:"+textRequestBody.Content),string(buff))

		err := json.Unmarshal(buff,&rsp)
		
		if err != nil {			
			log.Println("error:", err)
			}
	}

	if rsp.Status == "success" {
		switch rsp.Mediatype{
			case "image":
				responseBody, err = makeImageResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,rsp.Mediaid)
			//	log.Println(string(responseBody))
				
			case "voice":
				responseBody, err = makeVoiceResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,rsp.Mediaid)
			//	log.Println(string(responseBody))
										
			case "video":
				var video MediaVideoinfo
				video.Mediaid = rsp.Mediaid
				video.Title = rsp.Mediatitle
				video.Desc = rsp.Mediadigest
				video.Url = rsp.Mediaurl 
				responseBody, err = makeVideoResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,video)
			//	log.Println(string(responseBody))
				
			case "news":
				var news MediaNewsinfo
				news.Mediaid = rsp.Mediaid
				news.Title = rsp.Mediatitle
				news.Desc = rsp.Mediadigest
				news.Url = rsp.Mediaurl  
				news.Picurl = rsp.Mediathumb
				responseBody, err = makeNewsResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,news)
			//	log.Println(string(responseBody))
				default:
						
		}
			
	}else{    									// 关键字查询失败(包括不能命中或者其他失败）时回复默认

		commreply, _ := redis.String(redisconn.Do("SRANDMEMBER", "commtoreply"))
		if commreply == ""{
			commreply = CommMsg
		}	
		responseBody, err = makeTextResponseBody(textRequestBody.ToUserName,textRequestBody.FromUserName,(emoji.Parse(":disappointed_face:")+commreply))
						
	//	log.Println(string(responseBody))
		
		if err != nil {
	//		log.Println("Wechat Service: makeTextResponseBody error: ", err)
			fmt.Fprintf(w, string("success"))
			return
		}

	}
	w.Header().Set("Content-Type", "text/xml")
	log.Println(string(responseBody))
	fmt.Fprintf(w, string(responseBody))
			
}

var redisconn  redis.Conn//定义数据库连接全局变量

func main() {                                         //主函数入口
	var err error
	var (
		version = flag.Bool("version", false, "version v1.0")
		port    = flag.Int("port", 80, "listen port.")
	)

	flag.Parse()

	if *version {
		fmt.Println("v1.0")
		os.Exit(0)
	}
	
	if os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_HOST")!="" &&  os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_PORT")!="" && os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_PASSWORD")!="" {
    		RedisDB = os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_HOST")+":"+os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_PORT")
   	 		RedisPWD = os.Getenv("QOVERY_DATABASE_WECHAT_REDIS_PASSWORD")
	}else{
    		RedisDB = "redis-12069.c1.us-east1-2.gce.cloud.redislabs.com:12069"
    		RedisPWD ="Juju1234"
	}
	redisconn, err = redis.Dial("tcp", RedisDB,redis.DialKeepAlive(time.Hour*48),redis.DialPassword(RedisPWD))  //连接redis数据库，记录用户文本记录和预处理
	if err != nil {                                   //如果无法连接redis数据库，不返回继续处理
        log.Println("Connect to redis error", err)
        
	}
	log.Println("connected to redis:"+RedisDB)
	defer redisconn.Close()
/*
	if _, err = redisconn.Do("AUTH", "Juju1234"); err != nil {
		log.Println("Auth to redis error", err)
	}
*/


	log.Println("Wechat Service Starting")
	http.HandleFunc("/", procRequest)
	err = http.ListenAndServe((":"+strconv.Itoa(*port)), nil)
	if err != nil {
		log.Fatal("Wechat Service: ListenAndServe failed, ", err)
	}
	log.Println("Wechat Service: Stop!")
}
