package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	IrisRecover "github.com/kataras/iris/v12/middleware/recover"
	"github.com/skip2/go-qrcode"
)

var app *iris.Application

const QRheader string = "data:image/png;base64,"

type heyfooqr_msg struct {
	URL string `json:"url"`
}
type heyfooqr_reg struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Image string `json:"image"`
}

// 错误处理
func PanicHandler() {
	exeName := os.Args[0]                                             //获取程序名称
	now := time.Now()                                                 //获取当前时间
	pid := os.Getpid()                                                //获取进程ID
	time_str := now.Format("20060102150405")                          //设定时间格式
	fname := fmt.Sprintf("%s-%d-%s-dump.log", exeName, pid, time_str) //保存错误信息文件名:程序名-进程ID-当前时间（年月日时分秒）
	fmt.Println("dump to file", fname)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()
	if err := recover(); err != nil {
		f.WriteString(fmt.Sprintf("%v\r\n", err)) //输出panic信息
		f.WriteString("========\r\n")
	}
	f.WriteString(string(debug.Stack())) //输出堆栈信息
}
func todayFilename() string {
	today := time.Now().Format("Jan 02 2006")
	return today + ".txt"
}
func newLogFile() *os.File {
	filename := "./log/" + todayFilename()
	// Open the file, this will append to the today's file if server restarted.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	return f
}
func main() {

	defer PanicHandler() // // 错误处理
	app = iris.New()
	app.Logger().SetLevel("error") //日志
	// 设置recover从panics恢复，设置log记录
	app.Use(logger.New())
	app.Use(IrisRecover.New())
	// 优雅的关闭程序
	serverWG := new(sync.WaitGroup)
	defer serverWG.Wait()
	iris.RegisterOnInterrupt(func() {
		serverWG.Add(1)
		defer serverWG.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		app.Shutdown(ctx)
		// 关闭全部主机
		for {
			fmt.Println("退出关闭,请输入exit")
			input := bufio.NewScanner(os.Stdin)
			input.Scan()
			code := input.Text()
			if code == "exit" {
				break
			}
		}
		app.Logger().Error("退出关闭")
		time.Sleep(1 * time.Second)
	})
	//通用
	app.Get("/test", test) //调试使用，正式部署注释
	app.Post("/heyfooqr", heyfooqr)

	fmt.Println("---------------------->>> 服务初始化成功!")
	app.Listen("127.0.0.1:5555", iris.WithoutServerError(iris.ErrServerClosed))
}

func test(ctx iris.Context) {
	ctx.WriteString("communication success!")
}
func heyfooqr(ctx iris.Context) {
	Msg := &heyfooqr_msg{}
	if err := ctx.ReadJSON(Msg); err != nil {
		var Res heyfooqr_reg
		Res.Code = -1
		Res.Msg = "ReadJSON 错误:" + err.Error()
		bu, _ := json.Marshal(Res)
		ctx.Write(bu)
		return
	} else {
		var Res heyfooqr_reg
		keyPNG := "0004" + Msg.URL
		buu, err := qrcode.Encode(keyPNG, qrcode.Medium, 256)
		if err != nil {
			Res.Code = -1
			Res.Msg = "QR 错误:" + err.Error()
			bu, _ := json.Marshal(Res)
			ctx.Write(bu)
			return
		}
		Res.Code = 0
		Res.Msg = "success"
		Res.Image = QRheader + base64.StdEncoding.EncodeToString(buu)
		bu, _ := json.Marshal(Res)
		ctx.Write(bu)
		return
	}
}
