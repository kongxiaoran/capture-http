package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
)

var staticExts = []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".woff", ".woff2", ".ttf", ".eot"}

func main() {

	var recRespDataFlag string
	var filterResource string
	var filterOtherHttp string
	fmt.Println(`------------------  合肥中台 http(s) 抓包工具 -------------------
$$$$$$$$\ $$$$$$\ $$\   $$\  $$$$$$\  $$$$$$\ $$\   $$\  $$$$$$\  
$$  _____|\_$$  _|$$$\  $$ |$$  __$$\ \_$$  _|$$$\  $$ |$$  __$$\ 
$$ |        $$ |  $$$$\ $$ |$$ /  \__|  $$ |  $$$$\ $$ |$$ /  $$ |
$$$$$\      $$ |  $$ $$\$$ |$$ |        $$ |  $$ $$\$$ |$$$$$$$$ |
$$  __|     $$ |  $$ \$$$$ |$$ |        $$ |  $$ \$$$$ |$$  __$$ |
$$ |        $$ |  $$ |\$$$ |$$ |  $$\   $$ |  $$ |\$$$ |$$ |  $$ |
$$ |      $$$$$$\ $$ | \$$ |\$$$$$$  |$$$$$$\ $$ | \$$ |$$ |  $$ |
\__|      \______|\__|  \__| \______/ \______|\__|  \__|\__|  \__|		  
	`)

	fmt.Println("是否 打印接口返回数据（1：是，0：否，默认不打印): 请输入1或0或回车")
	_, err := fmt.Scanln(&recRespDataFlag)
	if err != nil {
		recRespDataFlag = "0"
	}

	fmt.Println("是否屏蔽非 finchina请求(1：是，0：否，默认屏蔽): 请输入1或0或回车")
	_, err2 := fmt.Scanln(&filterOtherHttp)
	if err2 != nil {
		filterOtherHttp = "1"
	}

	fmt.Println("是否屏蔽静态资源类请求(1：是，0：否，默认屏蔽): 请输入1或0或回车")
	_, err1 := fmt.Scanln(&filterResource)
	if err1 != nil {
		filterResource = "1"
	}

	proxy := goproxy.NewProxyHttpServer()

	// 开启详细日志
	proxy.Verbose = false

	// 设置HTTPS拦截
	//proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.Tr.Proxy = http.ProxyFromEnvironment
	proxy.Tr.DialContext = nil
	proxy.Tr.Dial = nil
	proxy.Tr.DialTLS = nil
	proxy.Tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// 创建一个带有过滤器的Logger
	filteredLogger := log.New(FilteredWriter{w: os.Stderr}, "", log.LstdFlags)
	// 将goproxy的Logger设置为我们的过滤Logger
	proxy.Logger = filteredLogger

	// 对CONNECT方法请求动态决定是否执行MITM
	proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		// 检查请求的目标域名
		if strings.Contains(host, "cdn") {
			// 对于目标域名包含 cdn 的请求，不执行MITM，直接转发
			return goproxy.OkConnect, host
		}
		// 其他所有请求默认执行MITM
		return goproxy.MitmConnect, host
	})

	// 设置请求处理器来修改请求
	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			path := r.URL.String()
			// 获取请求的路径
			if strings.Contains(path, ".js") || strings.Contains(path, ".css") || strings.Contains(path, ".html") {
				r.URL.Host = "appdev.finchina.com"
				r.Host = "appdev.finchina.com"
				r.URL.Scheme = "https"
			}
			if "1" == filterResource {
				// 检查路径是否以静态资源的扩展名结尾
				for _, ext := range staticExts {
					if strings.HasSuffix(path, ext) {
						// 是静态资源，不打印日志并直接返回请求
						return r, nil
					}
				}
			}
			if "1" == filterOtherHttp && !strings.Contains(path, "finchinaAPP") {
				return r, nil
			}

			contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if r.Method == "GET" {
				log.Printf("接收到 GET请求: %s\n", r.URL.String())
				message := map[string]string{
					"id":     "notification",
					"time":   time.Now().Format("15:04:05"),
					"method": "Get",
					"path":   r.URL.String(),
					"param":  "",
					"resp":   "",
				}
				SendWebsocket(message)
			} else {
				log.Printf("接收到 POST请求: %s\n", r.URL.String())
				switch contentType {
				case "application/json":
					var requestBody map[string]interface{}
					// 读取请求体
					body, _ := io.ReadAll(r.Body)
					defer r.Body.Close()

					// 解析JSON
					_ = json.Unmarshal(body, &requestBody)

					// 使用json.MarshalIndent函数打印格式化后的JSON数据
					formattedData, err := json.MarshalIndent(requestBody, "", "    ")
					if err != nil {
						log.Fatalf("格式化JSON失败: %v", err)
					}
					log.Printf("JSON 参数:\n%s\n", formattedData)
					r.Body = io.NopCloser(bytes.NewBuffer(body))

					message := map[string]string{
						"time":   time.Now().Format("15:04:05"),
						"method": "POST",
						"path":   r.URL.String(),
						"param":  string(formattedData),
					}
					SendWebsocket(message)
				case "application/x-www-form-urlencoded":
					fmt.Println("Form 参数如下: ")
					// 解析请求体中的表单数据
					if err := r.ParseForm(); err != nil {
					}
					form := ""
					// 遍历所有表单参数并打印
					for key, values := range r.Form {
						// 因为同一个键可能对应多个值，所以 values 是一个字符串切片
						for _, value := range values {
							fmt.Printf("%s : %s\n", key, value)
							form += key + ":" + value + "<br/>"
						}
					}

					message := map[string]string{
						"time":   time.Now().Format("15:04:05"),
						"method": "POST",
						"path":   r.URL.String(),
						"param":  form,
					}
					SendWebsocket(message)

				default:
					fmt.Printf("目前不支持的 content type: %s\n", contentType)
				}
			}
			return r, nil
		})

	// 设置响应处理器来修改响应
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if "1" == recRespDataFlag {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				defer resp.Body.Close() // 关闭Body

				bodyString := string(bodyBytes)
				// 这里可以根据需要修改resp（响应）
				log.Printf("接口响应：\n %s\n", bodyString)
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
			return resp
		})

	go WebsocketAndHTML()

	// 监听并服务
	fmt.Println("--------------- 开始监听 本机9999端口的Https请求 ---------------------")
	log.Fatal(http.ListenAndServe(":9999", proxy))
}

type FilteredWriter struct {
	w io.Writer
}

func (fw FilteredWriter) Write(p []byte) (n int, err error) {
	if strings.Contains(string(p), "WARN: Cannot handshake") {
		// 如果日志消息包含特定字符串，就过滤掉该消息
		return len(p), nil // 返回成功，但不实际写入
	}
	// 否则，将消息写入原始的写入器
	return fw.w.Write(p)
}
