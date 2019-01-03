package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"
)

type localClient struct {
	client *http.Client
}

func main() {
	lc := &localClient{}

	// ダミーのmockServerを用意している。実際の処理ではいらない
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.URL, r.Header, r.Body) // requestの内容を確認するためにログ出力
		time.Sleep(10 * time.Second)	// request cancelを試すためにわざと応答をsleepしている
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	// context.Backgroundでcontextを生成している
	ctx, cancel := context.WithCancel(context.Background())

	// client側の処理
	lc.client = &http.Client{}

	// request bodyの準備
	values := url.Values{}
	values.Set("token", "testtoken")

	req, err := http.NewRequest(
		http.MethodPost,
		ts.URL, // mockServerへのrequestにしてあるが実際の処理ではちゃんとendpointを書く
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		panic(err)
	}

	req = req.WithContext(ctx)

	// request Headerを指定
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// response受け取り用のchannel
	ch := make(chan []byte)
	var res []byte

	// requestのgoroutine
	go func() {
		resp, err := lc.request(req)
		if err != nil {
			ch <- nil
			return
		}
		ch <- resp
	}()

	// 5秒でcancelしている
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	// requestのgoroutineがchannelへデータを入れるまでwaitする
	res = <-ch

	log.Print("response:", string(res))
}

func (lc localClient) request(req *http.Request) ([]byte, error) {
	log.Print("start request")

	resp, err := lc.client.Do(req)
	if err != nil {
		log.Print("error request :", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("error response body")
		return nil, err
	}

	log.Print("end request")

	return body, nil
}
