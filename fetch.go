//爬虫相关
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fastjson"
)

const acLivePage = "https://m.acfun.cn/live/detail/"

const userAgent = "Mozilla/5.0 (iPad; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"

// 爬取页面
func fetchPage(page string) *goquery.Document {
	client := &http.Client{}
	req, err := http.NewRequest("GET", page, nil)
	checkErr(err)

	// 需要设置手机版user-agent
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	checkErr(err)
	return doc
}

// 爬取主播wap版直播页面
func fetchLivePage(uid uint) *goquery.Document {
	upLivePage := acLivePage + fmt.Sprint(uid)
	return fetchPage(upLivePage)
}

// 查看主播是否在直播
func (s streamer) isLiveOn() bool {
	doc := fetchLivePage(s.UID)

	if doc.Find("p.closed-tip").Text() == "直播已结束" {
		return false
	}
	return true
}

// 根据uid获取主播的id
func getID(uid uint) string {
	doc := fetchLivePage(uid)

	// 主播没在开播
	id := doc.Find("a.up-link").Text()
	if id != "" {
		return id
	}

	// 主播正在开播
	return doc.Find("div.user-nickname").Text()
}

// 获取主播直播的标题
func (s streamer) getTitle() string {
	doc := fetchLivePage(s.UID)

	return doc.Find("h1.live-content-title-text").Text()
}

// 获取AcFun的logo
func fetchAcLogo() {
	const acLogo = "https://cdn.aixifan.com/ico/favicon.ico"

	resp, err := http.Get(acLogo)
	checkErr(err)
	defer resp.Body.Close()

	newLogoFile, err := os.Create(logoFileLocation)
	checkErr(err)
	defer newLogoFile.Close()

	_, err = io.Copy(newLogoFile, resp.Body)
	checkErr(err)
}

// 获取AcFun的直播源，分为hls和flv两种
func (s streamer) getStreamURL() (hlsURL string, flvURL string) {
	const acPage = "https://m.acfun.cn"
	const loginPage = "https://id.app.acfun.cn/rest/app/visitor/login"

	client := &http.Client{}

	req, err := http.NewRequest("GET", acPage, nil)
	checkErr(err)
	// 需要设置手机版user-agent，下同
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()

	// 获取did（device ID）
	var didCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "_did" {
			didCookie = cookie
		}
	}
	did := didCookie.Value

	form := url.Values{}
	form.Set("sid", "acfun.api.visitor")
	req, err = http.NewRequest("POST", loginPage, strings.NewReader(form.Encode()))
	checkErr(err)

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// 需要did的cookie
	req.AddCookie(didCookie)

	resp, err = client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 0 {
		return "", ""
	}
	// 获取userId和对应的令牌
	userID := v.GetInt("userId")
	serviceToken := string(v.GetStringBytes("acfun.api.visitor_st"))

	// 获取直播源的地址需要userId、did和对应的令牌
	streamPage := "https://api.kuaishouzt.com/rest/zt/live/web/startPlay?subBiz=mainApp&kpn=ACFUN_APP&kpf=OUTSIDE_IOS_H5&userId=" + fmt.Sprint(userID) + "&did=" + did + "&acfun.api.visitor_st=" + serviceToken

	form = url.Values{}
	// authorId就是主播的uid
	form.Set("authorId", s.uidStr())
	req, err = http.NewRequest("POST", streamPage, strings.NewReader(form.Encode()))
	checkErr(err)

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	checkErr(err)

	v, err = p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 1 {
		return "", ""
	}
	videoPlayRes := v.GetStringBytes("data", "videoPlayRes")
	//fmt.Println(string(videoPlayRes))
	v, err = p.ParseBytes(videoPlayRes)
	checkErr(err)
	streamName := string(v.GetStringBytes("streamName"))

	representation := v.GetArray("liveAdaptiveManifest", "0", "adaptationSet", "representation")

	// 选择码率最高的flv视频源
	sort.Slice(representation, func(i, j int) bool {
		return representation[i].GetInt("bitrate") > representation[j].GetInt("bitrate")
	})
	flvURL = string(representation[0].GetStringBytes("url"))

	i := strings.Index(flvURL, streamName)
	streamHlsURL := strings.Replace(flvURL[0:i], "pull", "hlspull", 1)
	// 这是码率最高的hls视频源
	hlsURL = streamHlsURL + streamName + ".m3u8"

	return hlsURL, flvURL
}
