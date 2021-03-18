package main

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"os"
	"log"
	"bytes"
//	"io"
)

const GetTokenUrl = "http://token.zhujq.ga:1080/token?appid=wxf183d5e1fe4d5204"

const GetMaterialSum = "https://api.weixin.qq.com/cgi-bin/material/get_materialcount?access_token="
const GetMaterial = "https://api.weixin.qq.com/cgi-bin/material/batchget_material?access_token="
//const Dbconn = "freedbtech_zhujq:Juju1234@tcp(freedb.tech:3306)/freedbtech_wechat"
var Dbconn string
//const Dbconn = "zhujq:Juju1234@tcp(wechat-mysql:3306)/wechat"
const GetMediainfo = "https://api.weixin.qq.com/cgi-bin/material/get_material?access_token="

type RequestMaterial struct {
	MaterialType string `json:"type"`
	Offset  uint32 `json:"offset"`
	Count   int `json:"count"`
}

type MediaCount struct {
	VoiceCount uint32 `json:"voice_count"`
	VideoCount  uint32 `json:"video_count"`
	ImageCount   uint32 `json:"image_count"`
	NewsCount uint32 `json:"news_count"`
}

type News_item struct {
	Newstitle  			string 	`json:"title"`                   //news时使用
	Newsthumbmediaid 	string  `json:"thumb_media_id"`
	Newsdigest 			string  `json:"digest"`
	Newsurl 			string  `json:"url"`
}

type MaterialNewsInfo struct {
	Newsitem  []News_item  `json:"news_item"`             //news时使用
}

type MaterialItemInfo struct {
	Mediaid  string `json:"media_id"`
	Content   MaterialNewsInfo `json:"content"`             //news时使用
	Name  string `json:"name"`                //图片、音频、视频时使用
	Url   string `json:"url"`                 //图片、音频、视频时使用,增加url字段
}

type MaterialInfo struct {
	MediaTotalount int `json:"total_count"`     
	MediaItemount int `json:"item_count"`         
	MediaItemInfo []MaterialItemInfo `json:"item"`

}

type Token struct {
	AccessToken string `json:"access_token"`
}

type MediaVideoinfo struct {
	Title string `json:"title"`
	Desc  string `json:"description"`
	Url   string `json:"down_url"`
}

type RequestMedia struct {                             //获取视频和图文详细信息使用
	Mediaid string `json:"media_id"`
}

/*
func init() {                                         //初始，日志文件生成
	file := "./" +"dblog"+ ".txt"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
			panic(err)
	}
	log.SetOutput(logFile) // 将文件设置为log输出的文件
	log.SetPrefix("[wechat-db]")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	return
}
*/

func httpClient() *http.Client {
	return &http.Client{ }
}

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

func RefreshData() bool {
	db, err := sql.Open("mysql",Dbconn)
	defer db.Close()
	err = db.Ping()
	if err != nil{
		log.Println("error:", err)	
		return false
	}
	var addedVoiceCount,addedImageCount,addedVideoCount,addedNewsCount uint32 = 0,0,0,0

	querysql := `select count(mediaid) from media where mediatype = "voice"`
	err  = db.QueryRow(querysql).Scan(&addedVoiceCount)
	querysql = `select count(mediaid) from media where mediatype = "image"`
	err  = db.QueryRow(querysql).Scan(&addedImageCount)
	querysql = `select count(mediaid) from media where mediatype = "video"`
	err  = db.QueryRow(querysql).Scan(&addedVideoCount)
	querysql = `select count(mediaid) from media where mediatype = "news"`
	err  = db.QueryRow(querysql).Scan(&addedNewsCount)
	if err != nil{
		log.Println("error:", err)	
		return false
	}

	buff, _ := HTTPGet(GetTokenUrl)	
	var t Token
	err = json.Unmarshal(buff,&t)
	if err != nil {
		log.Println("error:", err)	
		return false
	}

	buff, err = HTTPGet(GetMaterialSum + t.AccessToken)	
	if err != nil{
		log.Println("error:", err)	
		return false
	}

	var mc MediaCount
	err  =  json.Unmarshal(buff,&mc)		
	if err != nil{
		log.Println("error:", err)	
		return false
	}

	if mc.VoiceCount > 0 && mc.VoiceCount > addedVoiceCount {
		var requestm RequestMaterial
		var i uint32 
		for i = 0; i < (mc.VoiceCount - addedVoiceCount) ; i++ {
		//	requestm.Offset = (i+addedVoiceCount)
		    requestm.Offset = i
			requestm.Count = 1
			requestm.MaterialType = "voice"		
			buff, _ = PostJson(GetMaterial + t.AccessToken,requestm)
			var m MaterialInfo
			json.Unmarshal(buff,&m)
			var insertsql = `insert into media(mediatype,mediaid,title,url) values("voice","` + m.MediaItemInfo[0].Mediaid +`","`+ m.MediaItemInfo[0].Name +`","`+ m.MediaItemInfo[0].Url + `") ON DUPLICATE KEY UPDATE title = "` + m.MediaItemInfo[0].Name + `";`
			log.Println(insertsql)
			_,err = db.Exec(insertsql)
			if err != nil{
				log.Println("error:", err)
				return false
			}

		}
	}
	
	if mc.ImageCount > 0 && mc.ImageCount > addedImageCount {
		var requestm RequestMaterial
		var i uint32 
		for i = 0; i < (mc.ImageCount - addedImageCount) ; i++ {
		//	requestm.Offset = (i+addedImageCount)
			requestm.Offset = i
			requestm.Count = 1
			requestm.MaterialType = "image"		
			buff, _ = PostJson(GetMaterial + t.AccessToken,requestm)
			var m MaterialInfo
			json.Unmarshal(buff,&m)
			var insertsql = `insert into media(mediatype,mediaid,title,url) values("image","` + m.MediaItemInfo[0].Mediaid +`","`+ m.MediaItemInfo[0].Name +`","`+ m.MediaItemInfo[0].Url+`") ON DUPLICATE KEY UPDATE title = "` + m.MediaItemInfo[0].Name + `";`
			log.Println(insertsql)
			_,err = db.Exec(insertsql)
			if err != nil{
				log.Println("error:", err)
				return false
			}

		}
	}

	if mc.VideoCount > 0 && mc.VideoCount > addedVideoCount {
		var requestm RequestMaterial
		var i uint32 
		for i = 0; i < (mc.VideoCount - addedVideoCount) ; i++ {
		//	requestm.Offset = (i+addedVideoCount)
			requestm.Offset = i
			requestm.Count = 1
			requestm.MaterialType = "video"		
			buff, _ = PostJson(GetMaterial + t.AccessToken,requestm)
			var m MaterialInfo
			json.Unmarshal(buff,&m)
			var insertsql = `insert into media(mediatype,mediaid,title,url) values("video","` + m.MediaItemInfo[0].Mediaid +`","`+ m.MediaItemInfo[0].Name + `","`+ m.MediaItemInfo[0].Url+`") ON DUPLICATE KEY UPDATE title = "` + m.MediaItemInfo[0].Name + `";`
			log.Println(insertsql)
			_,err = db.Exec(insertsql)
			if err != nil{
				log.Println("error:", err)	
				return false
			}
			var requestm RequestMedia
			requestm.Mediaid = m.MediaItemInfo[0].Mediaid
			buff, _ = json.Marshal(requestm)
			buff, _ = PostJson(GetMediainfo+ t.AccessToken,requestm)
			var video MediaVideoinfo
			err  =  json.Unmarshal(buff,&video)
			if err == nil{
				var updatesql = `update media set digest = "` + video.Desc +`" where mediaid = "` + m.MediaItemInfo[0].Mediaid + `";`
			    db.Exec(updatesql)		
			}
		}
	}

	if mc.NewsCount > 0 && mc.NewsCount > addedNewsCount {
		var requestm RequestMaterial
		var i uint32 
		for i = 0; i < (mc.NewsCount - addedNewsCount); i++ {
		//	requestm.Offset = (i+addedNewsCount)
			requestm.Offset = i
			requestm.Count = 1
			requestm.MaterialType = "news"		
			buff, _ = PostJson(GetMaterial + t.AccessToken,requestm)
		//	log.Println(buff)
			var m MaterialInfo
			json.Unmarshal(buff,&m)
		//	log.Println(m)
			var insertsql = `insert into media(mediatype,mediaid,title,url,digest,thumbmedia) values("news","` + m.MediaItemInfo[0].Mediaid +`","`+ m.MediaItemInfo[0].Content.Newsitem[0].Newstitle + `","`+ m.MediaItemInfo[0].Content.Newsitem[0].Newsurl + `","`+ m.MediaItemInfo[0].Content.Newsitem[0].Newsdigest + `","`+ m.MediaItemInfo[0].Content.Newsitem[0].Newsthumbmediaid +`") ON DUPLICATE KEY UPDATE title = "` + m.MediaItemInfo[0].Content.Newsitem[0].Newstitle + `";`
			log.Println(insertsql)
			_,err = db.Exec(insertsql)
			if err != nil{
				log.Println("error:", err)
				return false
			}
		}
	}
	
	log.Println("Refresh Success",(mc.VoiceCount-addedVoiceCount),(mc.ImageCount-addedImageCount),(mc.VideoCount-addedVideoCount),(mc.NewsCount-addedNewsCount))
	return true
}

func main() {    //主函数入口
	Dbconn = os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_USERNAME") + ":"+os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_PASSWORD") + "@tcp(" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_HOST") + ":" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_PORT") + ")/" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_NAME")
	if Dbconn == ""{
    		Dbconn = "freedbtech_zhujq:Juju1234@tcp(freedb.tech:3306)/freedbtech_wechat"
	}	
	tick :=time.NewTicker( 24 * time.Hour)
	defer tick.Stop()
	RefreshData() 
  	for {
    	select {
			case <-tick.C:
				RefreshData() 
    	}
  	}
}
