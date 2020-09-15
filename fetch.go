//爬虫相关
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
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

type httpVars struct {
	url         string
	body        []byte
	method      string
	cookies     []*fasthttp.Cookie
	userAgent   string
	contentType string
	referer     string
}

var httpClient = &fasthttp.Client{
	ReadTimeout:  10 * time.Second,
	WriteTimeout: 10 * time.Second,
}

var didCookie string

var (
	fetchRoomPool   fastjson.ParserPool
	getLiveInfoPool fastjson.ParserPool
	getStreamPool1  fastjson.ParserPool
	getStreamPool2  fastjson.ParserPool
	getStreamPool3  fastjson.ParserPool
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
	sync.Mutex                   // rooms的锁
	rooms      *map[int]liveRoom // 现在的liveRoom
	newRooms   *map[int]liveRoom // 新的liveRoom
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
func (hv *httpVars) httpRequest() (resp *fasthttp.Response, e error) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErrf("Recovering from panic in httpRequest(), the error is: %v", err)
			lPrintErrf("请求 %s 时出错，错误为 %v", hv.url, err)
			var ok bool
			e, ok = err.(error)
			if !ok {
				e = fmt.Errorf("%s", err)
			}
		}
	}()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp = fasthttp.AcquireResponse()

	if hv.url != "" {
		req.SetRequestURI(hv.url)
	} else {
		return nil, fmt.Errorf("请求的url不能为空")
	}

	if len(hv.body) != 0 {
		req.SetBody(hv.body)
	}

	if hv.method != "" {
		req.Header.SetMethod(hv.method)
	} else {
		// 默认为GET
		req.Header.SetMethod("GET")
	}

	if len(hv.cookies) != 0 {
		for _, cookie := range hv.cookies {
			req.Header.SetCookieBytesKV(cookie.Key(), cookie.Value())
		}
	}

	if hv.userAgent != "" {
		req.Header.SetUserAgent(hv.userAgent)
	}

	if hv.contentType != "" {
		req.Header.SetContentType(hv.contentType)
	}

	if hv.referer != "" {
		req.Header.SetReferer(hv.referer)
	}

	err := httpClient.Do(req, resp)
	checkErr(err)

	return resp, nil
}

// 获取全部AcFun直播间
func fetchAllRooms() {
	page := "0"
	allRooms := make(map[int]liveRoom)
	for page != "no_more" && page != "" {
		rooms, nextPage := fetchLiveRoom(page)
		if rooms == nil && nextPage == "" {
			break
		}
		page = nextPage
		for uid, r := range *rooms {
			allRooms[uid] = r
		}
	}

	liveRooms.newRooms = &allRooms
}

// 获取指定页数的AcFun直播间
func fetchLiveRoom(page string) (r *map[int]liveRoom, nextPage string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in fetchLiveRoom(), the error is:", err)
			lPrintErr("获取AcFun直播间列表时发生错误，尝试重新运行")
			// 延迟两秒，防止意外情况下刷屏
			time.Sleep(2 * time.Second)
			r, nextPage = fetchLiveRoom(page)
		}
	}()

	const acLive = "https://live.acfun.cn/api/channel/list?count=1000&pcursor=%s"

	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	err := cookie.Parse(didCookie)
	checkErr(err)
	hv := &httpVars{
		url:     fmt.Sprintf(acLive, page),
		method:  "GET",
		cookies: []*fasthttp.Cookie{cookie}, // 需要didCookie
	}
	resp, err := hv.httpRequest()
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

	return &rooms, nextPage
}

// 获取主播直播间的标题
func (s streamer) getTitle() string {
	liveRooms.Lock()
	room, ok := (*liveRooms.rooms)[s.UID]
	liveRooms.Unlock()
	if ok {
		return room.title
	}
	return ""
}

// 查看主播是否在直播
func (s streamer) isLiveOn() bool {
	liveRooms.Lock()
	_, ok := (*liveRooms.rooms)[s.UID]
	liveRooms.Unlock()
	return ok
}

// 获取用户直播相关信息
func getLiveInfo(uid int) (name string, isLive bool, title string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getLiveInfo(), the error is:", err)
			lPrintErr("获取uid为" + itoa(uid) + "的主播的直播信息时出错，尝试重新运行")
			time.Sleep(2 * time.Second)
			name, isLive, title = getLiveInfo(uid)
		}
	}()

	//const acLiveInfo = "https://api-new.app.acfun.cn/rest/app/live/info?authorId=%d"
	const acLiveInfo = "https://api-new.acfunchina.com/rest/app/live/info?authorId=%d"

	hv := &httpVars{
		url:    fmt.Sprintf(acLiveInfo, uid),
		method: "GET",
	}
	resp, err := hv.httpRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	p := getLiveInfoPool.Get()
	defer getLiveInfoPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)

	if !v.Exists("result") || v.GetInt("result") != 0 {
		lPrintErrf("无法获取uid为%d的主播的直播信息，响应为：%s", uid, string(body))
		return "", false, ""
	}

	name = string(v.GetStringBytes("user", "name"))

	if v.Exists("liveId") {
		isLive = true
	} else {
		isLive = false
	}

	title = string(v.GetStringBytes("title"))

	return name, isLive, title
}

// 通过wap版网页查看主播是否在直播
func (s streamer) isLiveOnByPage() (isLive bool) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in isLiveOnByPage(), the error is:", err)
			lPrintErr("获取" + s.longID() + "的直播页面时出错，尝试重新运行")
			time.Sleep(2 * time.Second)
			isLive = s.isLiveOnByPage()
		}
	}()

	const acLivePage = "https://m.acfun.cn/live/detail/"
	const userAgent = "Mozilla/5.0 (iPad; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1"

	hv := &httpVars{
		url:       acLivePage + itoa(s.UID),
		method:    "GET",
		userAgent: userAgent,
	}
	resp, err := hv.httpRequest()
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
	room, ok := (*liveRooms.rooms)[uid]
	liveRooms.Unlock()
	if ok {
		return room.name
	}

	name, _, _ := getLiveInfo(uid)
	return name
}

// 获取AcFun的logo
func fetchAcLogo() {
	const acLogo = "https://cdn.aixifan.com/ico/favicon.ico"

	hv := &httpVars{
		url:    acLogo,
		method: "GET",
	}
	resp, err := hv.httpRequest()
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

	hv := &httpVars{
		url:    mainPage,
		method: "GET",
	}
	resp, err := hv.httpRequest()
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

// 获取AcFun的直播源，分为hls和flv两种
func (s streamer) getStreamURL() (hlsURL string, flvURL string, streamName string, cfg acfundanmu.SubConfig) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getStreamURL(), the error is:", err)
			lPrintErr("获取" + s.longID() + "的直播源时出错，尝试重新运行")
			time.Sleep(2 * time.Second)
			hlsURL, flvURL, streamName, cfg = s.getStreamURL()
		}
	}()

	const loginPage = "https://id.app.acfun.cn/rest/app/visitor/login"
	const playURL = "https://api.kuaishouzt.com/rest/zt/live/web/startPlay?subBiz=mainApp&kpn=ACFUN_APP&kpf=PC_WEB&userId=%d&did=%s&acfun.api.visitor_st=%s"

	form := fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(form)
	form.Set("sid", "acfun.api.visitor")
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	err := cookie.Parse(didCookie)
	checkErr(err)
	hv := &httpVars{
		url:         loginPage,
		body:        form.QueryString(),
		method:      "POST",
		cookies:     []*fasthttp.Cookie{cookie},
		contentType: "application/x-www-form-urlencoded",
	}
	resp, err := hv.httpRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()

	p := getStreamPool1.Get()
	defer getStreamPool1.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)
	if !v.Exists("result") || v.GetInt("result") != 0 {
		return "", "", "", cfg
	}
	// 获取userId和对应的令牌
	userID := v.GetInt64("userId")
	serviceToken := string(v.GetStringBytes("acfun.api.visitor_st"))

	// 获取直播源的地址需要userId、did和对应的令牌
	streamURL := fmt.Sprintf(playURL, userID, cookie.Value(), serviceToken)

	form = fasthttp.AcquireArgs()
	defer fasthttp.ReleaseArgs(form)
	// authorId就是主播的uid
	form.Set("authorId", s.itoa())
	form.Set("pullStreamType", "FLV")
	hv = &httpVars{
		url:         streamURL,
		body:        form.QueryString(),
		method:      "POST",
		contentType: "application/x-www-form-urlencoded",
		referer:     s.getURL(), // 会验证 Referer
	}
	resp, err = hv.httpRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body = resp.Body()

	p = getStreamPool2.Get()
	defer getStreamPool2.Put(p)
	v, err = p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 1 {
		return "", "", "", cfg
	}
	videoPlayRes := v.GetStringBytes("data", "videoPlayRes")
	p = getStreamPool3.Get()
	defer getStreamPool3.Put(p)
	v, err = p.ParseBytes(videoPlayRes)
	checkErr(err)
	streamName = string(v.GetStringBytes("streamName"))

	representation := v.GetArray("liveAdaptiveManifest", "0", "adaptationSet", "representation")
	sort.Slice(representation, func(i, j int) bool {
		return representation[i].GetInt("bitrate") < representation[j].GetInt("bitrate")
	})

	index := 0
	if s.Bitrate == 0 {
		// s.Bitrate为0时选择码率最高的直播源
		index = len(representation) - 1
	} else {
		// 选择s.Bitrate下码率最高的直播源
		for i, r := range representation {
			if s.Bitrate >= r.GetInt("bitrate") {
				index = i
			} else {
				break
			}
		}
	}

	flvURL = string(representation[index].GetStringBytes("url"))

	bitrate := representation[index].GetInt("bitrate")
	switch {
	case bitrate >= 4000:
		cfg = subConfigs[1080]
	case len(representation) >= 2 && bitrate >= 2000:
		cfg = subConfigs[720]
	case bitrate == 0:
		cfg = subConfigs[0]
	default:
		cfg = subConfigs[540]
	}

	i := strings.Index(flvURL, "flv?")
	// 这是flv对应的hls视频源
	hlsURL = strings.ReplaceAll(flvURL[0:i], "pull.etoote.com", "hlspull.etoote.com") + "m3u8"

	return hlsURL, flvURL, streamName, cfg
}

// 根据config.Source获取直播源
func (s streamer) getLiveURL() (liveURL string) {
	switch config.Source {
	case "hls":
		liveURL, _, _, _ = s.getStreamURL()
	case "flv":
		_, liveURL, _, _ = s.getStreamURL()
	default:
		lPrintErr(configFile + "里的Source必须是hls或flv")
		return ""
	}
	return liveURL
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
		hlsURL, flvURL, _, _ := s.getStreamURL()
		lPrintln(s.longID() + "正在直播：" + title)
		if flvURL == "" {
			lPrintErr("无法获取" + s.longID() + "的直播源，请重新运行命令")
		} else {
			lPrintln(s.longID() + "直播源的hls和flv地址分别是：" + "\n" + hlsURL + "\n" + flvURL)
		}
		return hlsURL, flvURL
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
			if _, isLive, title := getLiveInfo(s.UID); isLive {
				mu.Lock()
				(*liveRooms.newRooms)[s.UID] = liveRoom{name: s.Name, title: title}
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
				if _, ok := (*liveRooms.newRooms)[s.UID]; !ok {
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
