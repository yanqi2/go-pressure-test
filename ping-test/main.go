package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"strings"
	"sttc.com/websocket-client/util"
	"sync"
	"time"
)

// array 自定义数组参数
type array []string

func (a *array) String() string {
	return fmt.Sprint(*a)
}

func (a array) Set(s string) error {
	panic("implement me")
}

var (
	requestURL = "ws://localhost:11033/websocket" // 压测的url 目前支持，http/https ws/wss
	//requestURL           = "ws://10.232.10.62:11033/websocket" // 压测的url 目前支持，http/https ws/wss
	concurrency    int64 = 1    // 并发数
	sendDurationMS int64 = 1000 // 发送ping间隔
	sleepTime      int64 = 10   // ping结束后sleep的时间
	sendSecond     int64 = 500  // 发送总秒数
	reqHeaders     array        // 自定义头信息传递给服务器
)

func init() {
	flag.StringVar(&requestURL, "u", requestURL, "压测地址")
	flag.Int64Var(&concurrency, "c", concurrency, "并发数")
	flag.Int64Var(&sendDurationMS, "sdms", sendDurationMS, "发送ping间隔")
	flag.Int64Var(&sleepTime, "sleep", sleepTime, "ping结束后sleep的时间")
	flag.Int64Var(&sendSecond, "ss", sendSecond, "发送总秒数")
	flag.Var(&reqHeaders, "H", "自定义头信息传递给服务器 示例:-H 'Content-Type: application/json'")

	// 解析参数
	flag.Parse()
}

func getHeaderValue(v string, headers map[string]string) {
	index := strings.Index(v, ":")
	if index < 0 {
		return
	}
	vIndex := index + 1
	if len(v) >= vIndex {
		value := strings.TrimPrefix(v[vIndex:], " ")
		if _, ok := headers[v[:index]]; ok {
			headers[v[:index]] = fmt.Sprintf("%s; %s", headers[v[:index]], value)
		} else {
			headers[v[:index]] = value
		}
	}
}

func connect(ws *util.WebSocket, wg *sync.WaitGroup) {
	headers := make(map[string]string)
	for _, v := range reqHeaders {
		getHeaderValue(v, headers)
	}

	err := ws.GetConn()
	if err != nil {
		ws.IsConnected = false
		fmt.Println("连接失败:", err)
	} else {
		ws.IsConnected = true
		//fmt.Println("连接成功")
	}
	// TODO bind

	wg.Done()
}

func writePing(ws *util.WebSocket, wg *sync.WaitGroup) {
	// 循环发送数据
	start := time.Now().Unix()
	for {
		if time.Now().Unix() > (start + sendSecond) {
			// 超过发送时间
			break
		}

		ctrl := []byte(fmt.Sprintf(""))
		w, err := ws.Conn.NewFrameWriter(websocket.PingFrame)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		_, err = w.Write(ctrl)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		//fmt.Println("发送了: ", b, " 字节")

		time.Sleep(time.Duration(sendDurationMS) * time.Millisecond)
	}
	wg.Done()
}

func readPong(ws *util.WebSocket, total int) {
	for i := 0; i < total; i++ {
		if content, err := ws.Read(); err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(content))
		}
	}
}

func main() {
	fmt.Println("requestURL: ", requestURL)
	fmt.Println("concurrency: ", concurrency)
	fmt.Println("sendDurationMS: ", sendDurationMS)
	fmt.Println("sleepTime: ", sleepTime)
	fmt.Println("sendSecond: ", sendSecond)

	wsList := make([]*util.WebSocket, 0)

	fmt.Println(time.Now().Unix(), " : 开始建立连接")

	connWg := new(sync.WaitGroup)
	connWg.Add(int(concurrency))
	for i := int64(0); i < concurrency; i++ {
		ws := util.NewWebSocket(requestURL)
		wsList = append(wsList, ws)
		go connect(ws, connWg)

		//time.Sleep(2 * time.Millisecond)
	}

	connWg.Wait()
	connected := 0
	for _, ws := range wsList {
		if ws.IsConnected {
			connected++
		}
	}
	fmt.Println(time.Now().Unix(), " : 建立了 ", connected, " 个连接")

	// 读取pong
	//readWg := new(sync.WaitGroup)
	//readWg.Add(connected)
	//// 计算理论接收 pong 个数
	//total := sendSecond / (sendDurationMS / 1000)
	//for _, ws := range wsList {
	//	if ws.IsConnected {
	//		go readPong(ws, int(total))
	//	}
	//}

	fmt.Println(time.Now().Unix(), " : 将发送ping")
	writeWg := new(sync.WaitGroup)
	writeWg.Add(connected)
	for i, ws := range wsList {
		if ws.IsConnected {
			go writePing(ws, writeWg)
			if i%5000 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
	}
	fmt.Println(time.Now().Unix(), " : 正在发送ping")
	writeWg.Wait()
	fmt.Println(time.Now().Unix(), " : 完成发送ping")

	// 阻塞
	time.Sleep(time.Duration(sleepTime) * time.Second)

	for _, ws := range wsList {
		ws.Close()
	}
}
