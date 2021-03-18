package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"database/sql"
//	"io"
	"net/http"
	"strings"
	_ "github.com/go-sql-driver/mysql"
//	"database/sql"
	"github.com/devfeel/dotweb"
)

type App struct {
	Web      *dotweb.DotWeb
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

//const Dbconn = "zhujq:Juju1234@tcp(wechat-mysql:3306)/wechat"
//const Dbconn = "freedbtech_zhujq:Juju1234@tcp(freedb.tech:3306)/freedbtech_wechat"
var Dbconn string

func NewApp() *App {
	var a = &App{}
	a.Web = dotweb.New()
	return a
}

var app = NewApp()

/*
func init() {                                         //初始，日志文件生成
	file := "./" +"logindex"+ ".txt"
	logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
			panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile) // 将文件设置为log输出的文件
//	mw := io.MultiWriter(os.Stdout,logFile) //同时输出到文件和控制台
//  log.SetOutput(mw)
	log.SetPrefix("[wechat-index]")
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	return
}
*/

var db *sql.DB

func main() {
	var err error

	var (
		version = flag.Bool("version", false, "version v1.0")
		port    = flag.Int("port", 8080, "listen port.")
	)

	flag.Parse()

	if *version {
		fmt.Println("v1.0")
		os.Exit(0)
	}

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	Dbconn = os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_USERNAME") + ":"+os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_PASSWORD") + "@tcp(" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_HOST") + ":" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_PORT") + ")/" + os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_NAME")
	if os.Getenv("QOVERY_DATABASE_WECHAT_MYSQL_HOST") == ""{
    		Dbconn = "freedbtech_zhujq:Juju1234@tcp(freedb.tech:3306)/freedbtech_wechat"
	}	

	db, err = sql.Open("mysql",Dbconn)
	db.SetConnMaxLifetime(0)
	defer db.Close()
	err = db.Ping()
	if err != nil{
		log.Println("error:", err)	
		return 
	}

	InitRoute(app.Web.HttpServer)
	log.Println("Start Wechat Index Server on ", *port)
	app.Web.StartServer(*port)
}

func indexHandler(ctx dotweb.Context) error {
	keyword := ctx.QueryString("keyword")
	log.Println(keyword)
	querytype := "default"

	var message = ResBody{
		Status:      "failed",
		Mediatype: "",
		Mediaid: "",
		Mediaurl: "",
		Mediadigest: "",
		Mediathumb: "",
	}

	if keyword == "" {
		log.Println("ERROR: 没有提供keyword")
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}

	if strings.HasPrefix(keyword,"poem:+"){

		keyword = strings.Replace(keyword, "poem:+", "", -1 ) 
		querytype = "poem"

	}

	for {                                            //去掉Keyword首尾空格
		if strings.HasPrefix(keyword," ") || strings.HasSuffix(keyword," "){
			keyword = strings.TrimPrefix(keyword," ")
			keyword = strings.TrimSuffix(keyword," ")		 
		}else{
			break
		}

	}

	var sqlstr string = ""

	

	if querytype == "default"{

		switch keyword{
		case "help","帮助":
			sqlstr = `select mediatype,mediaid,title,url,digest,thumbmedia from media where title = "公众号使用帮助" and mediatype = "news"  order by rand() limit 1; `
		case "about me","关于我","aboutme":
			sqlstr = `select mediatype,mediaid,title,url,digest,thumbmedia from media where title = "about me" and mediatype = "news" order by rand() limit 1; `
		case "list","文章","文章列表","ls":
			sqlstr = `select mediatype,mediaid,title,url,digest,thumbmedia from media where title = "原创文章列表" and mediatype = "news" order by rand() limit 1; `
		default:
			keyword = strings.ReplaceAll(keyword,` `,`%" and title like "%`)
			if strings.LastIndex(keyword,"+") == (len(keyword) - 2) && strings.LastIndex(keyword,"+") != -1 && len(keyword) >= 3 {        //关键字含有+ 且 倒数第二字节是+ 
				lastword := string(keyword[len(keyword)-1:])
				prefix := string(keyword[0:len(keyword)-2])
				switch lastword{
				case "V","v":
					sqlstr = `select a.mediatype,a.mediaid,a.title,a.url,a.digest,a.thumbmedia from media a inner join (select id from media  where title like "%` + prefix + `%"  and mediatype = "video" order by rand() limit 1) b on a.id=b.id; ` //20200330优化随机返回结果
				case "A","a":
					sqlstr = `select a.mediatype,a.mediaid,a.title,a.url,a.digest,a.thumbmedia from media a inner join (select id from media  where title like "%` + prefix + `%"  and mediatype = "news" order by rand() limit 1) b on a.id=b.id; ` //20200330优化随机返回结果
				case "I","i":
					sqlstr = `select a.mediatype,a.mediaid,a.title,a.url,a.digest,a.thumbmedia from media a inner join (select id from media  where title like "%` + prefix + `%"  and mediatype = "image" order by rand() limit 1) b on a.id=b.id; ` //20200330优化随机返回结果		
				default:
					sqlstr = `select a.mediatype,a.mediaid,a.title,a.url,a.digest,a.thumbmedia from media a inner join (select id from media  where title like "%` + keyword + `%"  order by rand() limit 1) b on a.id=b.id; ` //20200330优化随机返回结果

				}

			}else{
				sqlstr = `select a.mediatype,a.mediaid,a.title,a.url,a.digest,a.thumbmedia from media  a inner join (select id from media  where title like "%` + keyword + `%"  order by rand() limit 1) b on a.id=b.id; ` //20200330优化随机返回结果
	
			}
		}
	}	
	if querytype == "poem"{
		keyword = strings.ReplaceAll(keyword,` `,`%" and content like "%`)
		sqlstr = `select a.content from poem  a inner join (select id from poem  where content like "%` + keyword + `%"  order by rand() limit 1) b on a.id=b.id; `
	}
	log.Println(sqlstr)

	row, err := db.Query(sqlstr)
	defer row.Close()
	if err != nil {
		log.Println("error:", err)	
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}

	if err = row.Err(); err != nil {
		log.Println("error:", err)	
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}
	
	count := 0
	for row.Next() {
		if querytype == "default" {
			if err := row.Scan(&message.Mediatype,&message.Mediaid,&message.Mediatitle,&message.Mediaurl,&message.Mediadigest,&message.Mediathumb); err != nil {
				log.Println("error:", err)	
				return ctx.WriteJsonC(http.StatusNotFound, message)
			}	
		}
		if 	querytype == "poem"{
			if err := row.Scan(&message.Mediadigest); err != nil {
				log.Println("error:", err)	
				return ctx.WriteJsonC(http.StatusNotFound, message)
			}	
			message.Mediatype = "poem"

		}
		count += 1;
		message.Status = "success"
	}

	if count ==0 {
		message.Status = "failed"
		return ctx.WriteJsonC(http.StatusNotFound, message)
	}	

	if message.Mediatype == "news"{    //图文类型时把封面图片的mediaid转换为Picurl
		sqlstr := `select url from media where mediaid = "` + message.Mediathumb+ `"; `
		rows, _ := db.Query(sqlstr)
		defer rows.Close()
		for rows.Next() {
			rows.Scan(&message.Mediathumb)
		}
	}
	
	return ctx.WriteJson(message)	
}

func InitRoute(server *dotweb.HttpServer) {
	server.GET("/", indexHandler)
}

