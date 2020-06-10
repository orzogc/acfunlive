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
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fastjson"
)

// 爬取主播wap版直播页面
func fetchLivePage(uid uint) (doc *goquery.Document) {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in fetchLivePage(), the error is:", err)
			timePrintln("获取uid为" + strconv.Itoa(int(uid)) + "的直播页面时出错，尝试重新运行")
			doc = fetchLivePage(uid)
		}
	}()

	const acLivePage = "https://m.acfun.cn/live/detail/"
	const userAgent = "Mozilla/5.0 (iPad; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"

	upLivePage := acLivePage + strconv.Itoa(int(uid))
	client := &http.Client{}
	req, err := http.NewRequest("GET", upLivePage, nil)
	checkErr(err)

	// 需要设置手机版user-agent
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()

	doc, err = goquery.NewDocumentFromReader(resp.Body)
	checkErr(err)

	return doc
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
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in getStreamURL(), the error is:", err)
			timePrintln("获取" + s.longID() + "的直播源时出错，尝试重新运行")
			hlsURL, flvURL = s.getStreamURL()
		}
	}()

	const acLivePage = "https://live.acfun.cn/live/"
	const loginPage = "https://id.app.acfun.cn/rest/app/visitor/login"
	const playURL = "https://api.kuaishouzt.com/rest/zt/live/web/startPlay?subBiz=mainApp&kpn=ACFUN_APP&kpf=PC_WEB&userId=%d&did=%s&acfun.api.visitor_st=%s"

	resp, err := http.Get(acLivePage + strconv.Itoa(int(s.UID)))
	checkErr(err)
	defer resp.Body.Close()

	// 获取did（device ID）
	var didCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "_did" {
			didCookie = cookie
		}
	}
	deviceID := didCookie.Value

	client := &http.Client{}
	form := url.Values{}
	form.Set("sid", "acfun.api.visitor")
	req, err := http.NewRequest("POST", loginPage, strings.NewReader(form.Encode()))
	checkErr(err)

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
	streamURL := fmt.Sprintf(playURL, userID, deviceID, serviceToken)

	form = url.Values{}
	// authorId就是主播的uid
	form.Set("authorId", s.uidStr())
	resp, err = http.PostForm(streamURL, form)
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
	// 这是码率最高的hls视频源
	hlsURL = strings.ReplaceAll(flvURL[0:i], "pull", "hlspull") + streamName + ".m3u8"

	return hlsURL, flvURL
}

// 查看指定主播是否在直播和输出其直播源
func printStreamURL(uid uint) {
	id := getID(uid)
	s := streamer{UID: uid, ID: id}

	if s.isLiveOn() {
		title := s.getTitle()
		hlsURL, flvURL := s.getStreamURL()
		logger.Println(s.longID() + "正在直播：" + title)
		if flvURL == "" {
			logger.Println("无法获取" + s.longID() + "的直播源，请重新运行命令")
		} else {
			logger.Println(s.longID() + "直播源的hls和flv地址分别是：" + "\n" + hlsURL + "\n" + flvURL)
		}
	} else {
		logger.Println(s.longID() + "不在直播")
	}
}
