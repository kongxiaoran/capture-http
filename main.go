package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("Received request: %s %s\n", r.Method, r.URL)
	//for name, values := range r.Header {
	//	for _, value := range values {
	//		fmt.Printf("%s: %s\n", name, value)
	//	}
	//}

	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if r.Method == "GET" {
		queryParams := r.URL.Query()
		for name, values := range queryParams {
			for _, value := range values {
				fmt.Printf("请求参数: %s = %s\n", name, value)
			}
		}
	} else {
		switch contentType {
		case "application/json":
			var jsonData map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&jsonData); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("JSON 参数:")
			for key, value := range jsonData {
				fmt.Printf("  %s: %v\n", key, value)
			}
		case "application/x-www-form-urlencoded":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("Form 参数:")
			for key, values := range r.PostForm {
				for _, value := range values {
					fmt.Printf("  %s: %s\n", key, value)
				}
			}
		// Add more cases for other content types as needed
		default:
			fmt.Printf("目前不支持的 content type: %s\n", contentType)
		}
	}

	// Create a new request to send to the target server
	targetURL, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, "错误 URL", http.StatusBadRequest)
		fmt.Printf("错误 URL: %v\n", err)
		return
	}

	newRequest, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, "请求发生错误", http.StatusInternalServerError)
		fmt.Printf("错误发送请求: %v\n", err)
		return
	}

	copyHeader(newRequest.Header, r.Header)

	client := &http.Client{}
	resp, err := client.Do(newRequest)
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		fmt.Printf("Server error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	http.HandleFunc("/", handleRequest)
	fmt.Println("启动代理服务 在端口:9999")
	if err := http.ListenAndServe(":9999", nil); err != nil {
		fmt.Printf("Error starting proxy server: %v\n", err)
	}
}
