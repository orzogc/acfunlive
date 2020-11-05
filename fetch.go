//爬虫相关
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/orzogc/acfundanmu"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

//const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36"
//const acUserInfo = "https://live.acfun.cn/rest/pc-direct/user/userInfo?userId=%d"
//const acAuthorID = "https://api-new.app.acfun.cn/rest/app/live/info?authorId=%d"
//const acLiveChannel = "https://api-plus.app.acfun.cn/rest/app/live/channel"
//const acUserInfo2 = "https://api-new.app.acfun.cn/rest/app/user/userInfo?userId=%d"

type httpClient struct {
	client      *fasthttp.Client
	url         string
	body        []byte
	method      string
	cookies     []*fasthttp.Cookie
	userAgent   string
	contentType string
	referer     string
}

var defaultClient = &fasthttp.Client{
	MaxIdleConnDuration: 90 * time.Second,
	ReadTimeout:         10 * time.Second,
	WriteTimeout:        10 * time.Second,
}

var didCookie string

var (
	fetchRoomPool   fastjson.ParserPool
	getLiveInfoPool fastjson.ParserPool
)

// 直播间的数据结构
type liveRoom struct {
	// 主播名字
	name string
	// 直播间标题
	title string
}

// liveRoom的map
var liveRooms struct {
	sync.Mutex                  // rooms的锁
	rooms      map[int]liveRoom // 现在的liveRoom
	newRooms   map[int]liveRoom // 新的liveRoom
}

// 获取主播的直播链接
func getURL(uid int) string {
	const livePage = "https://live.acfun.cn/live/"
	return livePage + itoa(uid)
}

// 获取主播的直播链接
func (s streamer) getURL() string {
	return getURL(s.UID)
}

// http请求，调用后需要 defer fasthttp.ReleaseResponse(resp)
func (c *httpClient) doRequest() (resp *fasthttp.Response, e error) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErrf("Recovering from panic in doRequest(), the error is: %v", err)
			e = fmt.Errorf("请求 %s 时出错，错误为 %w", c.url, err)
			fasthttp.ReleaseResponse(resp)
		}
	}()

	if c.client == nil {
		c.client = defaultClient
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp = fasthttp.AcquireResponse()

	if c.url != "" {
		req.SetRequestURI(c.url)
	} else {
		fasthttp.ReleaseResponse(resp)
		return nil, fmt.Errorf("请求的url不能为空")
	}

	if len(c.body) != 0 {
		req.SetBody(c.body)
	}

	if c.method != "" {
		req.Header.SetMethod(c.method)
	} else {
		// 默认为GET
		req.Header.SetMethod("GET")
	}

	if len(c.cookies) != 0 {
		for _, cookie := range c.cookies {
			req.Header.SetCookieBytesKV(cookie.Key(), cookie.Value())
		}
	}

	if c.userAgent != "" {
		req.Header.SetUserAgent(c.userAgent)
	}

	if c.contentType != "" {
		req.Header.SetContentType(c.contentType)
	}

	if c.referer != "" {
		req.Header.SetReferer(c.referer)
	}

	err := c.client.Do(req, resp)
	checkErr(err)

	return resp, nil
}

// 获取全部AcFun直播间
func fetchAllRooms() {
	page := "0"
	liveRooms.newRooms = make(map[int]liveRoom)
	for page != "no_more" && page != "" {
		rooms, nextPage := fetchLiveRoom(page)
		if rooms == nil && nextPage == "" {
			break
		}
		page = nextPage
		for uid, r := range rooms {
			liveRooms.newRooms[uid] = r
		}
	}
}

// 获取指定页数的AcFun直播间
func fetchLiveRoom(page string) (r map[int]liveRoom, nextPage string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in fetchLiveRoom(), the error is:", err)
			lPrintErr("获取AcFun直播间列表时发生错误")
		}
	}()

	//const acLive = "https://api-new.app.acfun.cn/rest/app/live/channel"
	const acLive = "https://live.acfun.cn/api/channel/list?count=1000&pcursor=%s"

	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	err := cookie.Parse(didCookie)
	checkErr(err)
	client := &httpClient{
		url:     fmt.Sprintf(acLive, page),
		method:  "GET",
		cookies: []*fasthttp.Cookie{cookie}, // 需要didCookie
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	p := fetchRoomPool.Get()
	defer fetchRoomPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)
	v = v.Get("channelListData")
	if !v.Exists("result") || v.GetInt("result") != 0 {
		lPrintErrf("无法获取AcFun直播间列表，响应为：%s", string(body))
		return nil, ""
	}

	var rooms = make(map[int]liveRoom)
	liveList := v.GetArray("liveList")
	for _, live := range liveList {
		uid := live.GetInt("authorId")
		room := liveRoom{
			name:  string(live.GetStringBytes("user", "name")),
			title: string(live.GetStringBytes("title")),
		}
		rooms[uid] = room
	}

	nextPage = string(v.GetStringBytes("pcursor"))

	return rooms, nextPage
}

// 获取主播直播间的标题
func (s streamer) getTitle() string {
	liveRooms.Lock()
	room, ok := liveRooms.rooms[s.UID]
	liveRooms.Unlock()
	if ok {
		return room.title
	}

	if _, isLive, title, err := tryGetLiveInfo(s.UID); err == nil && isLive {
		return title
	}
	return ""
}

// 查看主播是否在直播
func (s streamer) isLiveOn() bool {
	liveRooms.Lock()
	_, ok := liveRooms.rooms[s.UID]
	liveRooms.Unlock()
	return ok
}

// 获取用户直播相关信息
func getLiveInfo(uid int) (name string, isLive bool, title string, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("getLiveInfo() error: %w", err)
		}
	}()

	const acLiveInfo = "https://api-new.app.acfun.cn/rest/app/live/info?authorId=%d"
	//const acLiveInfo = "https://api-new.acfunchina.com/rest/app/live/info?authorId=%d"

	client := &httpClient{
		url:    fmt.Sprintf(acLiveInfo, uid),
		method: "GET",
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	p := getLiveInfoPool.Get()
	defer getLiveInfoPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)

	if !v.Exists("result") || v.GetInt("result") != 0 {
		return "", false, "", fmt.Errorf("无法获取uid为%d的主播的直播信息，响应为：%s", uid, string(body))
	}

	name = string(v.GetStringBytes("user", "name"))

	if v.Exists("liveId") {
		isLive = true
	} else {
		isLive = false
	}

	title = string(v.GetStringBytes("title"))

	return name, isLive, title, nil
}

// 获取用户直播相关信息
func tryGetLiveInfo(uid int) (name string, isLive bool, title string, err error) {
	err = run(func() (err error) {
		name, isLive, title, err = getLiveInfo(uid)
		return err
	})
	return name, isLive, title, err
}

// 通过wap版网页查看主播是否在直播
func (s streamer) isLiveOnByPage() (isLive bool) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in isLiveOnByPage(), the error is:", err)
			lPrintErr("获取" + s.longID() + "的直播页面时出错")
		}
	}()

	const acLivePage = "https://m.acfun.cn/live/detail/"
	const userAgent = "Mozilla/5.0 (iPad; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"

	client := &httpClient{
		url:       acLivePage + itoa(s.UID),
		method:    "GET",
		userAgent: userAgent,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	checkErr(err)
	if doc.Find("p.closed-tip").Text() == "直播已结束" {
		return false
	}
	return true
}

// 根据uid获取主播的名字
func getName(uid int) string {
	liveRooms.Lock()
	room, ok := liveRooms.rooms[uid]
	liveRooms.Unlock()
	if ok {
		return room.name
	}

	name, _, _, err := tryGetLiveInfo(uid)
	if err != nil {
		return ""
	}
	return name
}

// 获取AcFun的logo
func fetchAcLogo() {
	const acLogo = "https://cdn.aixifan.com/ico/favicon.ico"

	client := &httpClient{
		url:    acLogo,
		method: "GET",
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	newLogoFile, err := os.Create(logoFileLocation)
	checkErr(err)
	defer newLogoFile.Close()

	_, err = newLogoFile.Write(body)
	checkErr(err)
}

// 获取did的cookie
func getDidCookie() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErrf("Recovering from panic in getDidCookie(), the error is: %v", err)
			lPrintErr("获取didCookie时出错，退出程序")
			os.Exit(1)
		}
	}()

	const mainPage = "https://live.acfun.cn"

	client := &httpClient{
		url:    mainPage,
		method: "GET",
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)

	// 获取did（device ID）
	resp.Header.VisitAllCookie(func(key, value []byte) {
		if string(key) == "_did" {
			if len(didCookie) == 0 {
				didCookie = string(value)
			}
		}
	})

	if len(didCookie) == 0 {
		lPrintErr("无法获取didCookie，退出程序")
		os.Exit(1)
	}
}

// 获取AcFun的直播源信息，分为hls和flv两种
func (s streamer) getStreamInfo() (info streamInfo, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("getStreamURL() error: %w", err)
		}
	}()

	dq, err := acfundanmu.Init(int64(s.UID), nil)
	checkErr(err)
	liveInfo := dq.GetStreamInfo()
	info.StreamInfo = liveInfo

	index := 0
	if s.Bitrate == 0 {
		// s.Bitrate为0时选择码率最高的直播源
		index = len(liveInfo.StreamList) - 1
	} else {
		// 选择s.Bitrate下码率最高的直播源
		for i, stream := range liveInfo.StreamList {
			if s.Bitrate >= stream.Bitrate {
				index = i
			} else {
				break
			}
		}
	}

	info.flvURL = liveInfo.StreamList[index].URL

	bitrate := liveInfo.StreamList[index].Bitrate
	switch {
	case bitrate >= 4000:
		info.cfg = subConfigs[1080]
	case len(liveInfo.StreamList) >= 2 && bitrate >= 2000:
		info.cfg = subConfigs[720]
	case bitrate == 0:
		info.cfg = subConfigs[0]
	default:
		info.cfg = subConfigs[540]
	}

	i := strings.Index(info.flvURL, "flv?")
	// 这是flv对应的hls视频源
	info.hlsURL = strings.ReplaceAll(info.flvURL[0:i], "pull.etoote.com", "hlspull.etoote.com") + "m3u8"

	return info, nil
}

// 根据config.Source获取直播源
func (s streamer) getLiveURL() (liveURL string, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("getLiveURL() error: %w", err)
		}
	}()

	info, err := s.getStreamInfo()
	checkErr(err)

	switch config.Source {
	case "hls":
		liveURL = info.hlsURL
	case "flv":
		liveURL = info.flvURL
	default:
		return "", fmt.Errorf("%s里的Source必须是hls或flv", configFile)
	}
	return liveURL, nil
}

// 查看指定主播是否在直播和输出其直播源
func printStreamURL(uid int) (string, string) {
	name := getName(uid)
	if name == "" {
		lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
		return "", ""
	}
	streamers.Lock()
	s, ok := streamers.crt[uid]
	streamers.Unlock()
	if !ok {
		s = streamer{UID: uid, Name: name}
	}

	if s.isLiveOn() {
		title := s.getTitle()
		lPrintln(s.longID() + "正在直播：" + title)
		info, err := s.getStreamInfo()
		if err != nil {
			lPrintErr("无法获取" + s.longID() + "的直播源，请重新运行命令")
		} else {
			lPrintln(s.longID() + "直播源的hls和flv地址分别是：" + "\n" + info.hlsURL + "\n" + info.flvURL)
		}
		return info.hlsURL, info.flvURL
	}

	lPrintln(s.longID() + "不在直播")
	return "", ""
}

// 通过用户直播相关信息并行查看主播是否在直播
func getLiveOnByInfo(ss []streamer) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, s := range ss {
		wg.Add(1)
		go func(s streamer) {
			if _, isLive, title, err := tryGetLiveInfo(s.UID); err == nil && isLive {
				mu.Lock()
				liveRooms.newRooms[s.UID] = liveRoom{name: s.Name, title: title}
				mu.Unlock()
			}
			wg.Done()
		}(s)
	}
	wg.Wait()
}

// 循环获取AcFun直播间数据
func cycleFetch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fetchAllRooms()

			var notLive []streamer
			streamers.Lock()
			// 应付AcFun的API的bug：虚拟偶像区的主播开播几分钟才会出现在channel里
			for _, s := range streamers.crt {
				if _, ok := liveRooms.newRooms[s.UID]; !ok {
					notLive = append(notLive, s)
				}
			}
			streamers.Unlock()

			// 并行的请求不能太多
			const num = 10
			length := len(notLive)
			q := length / num
			r := length % num
			for i := 0; i < q; i++ {
				getLiveOnByInfo(notLive[i*num : (i+1)*num])
			}
			if r != 0 {
				getLiveOnByInfo(notLive[length-r : length])
			}

			liveRooms.Lock()
			liveRooms.rooms = liveRooms.newRooms
			liveRooms.Unlock()

			// 每10秒循环一次
			time.Sleep(10 * time.Second)
		}
	}
}
