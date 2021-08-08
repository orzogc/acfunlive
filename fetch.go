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

var (
	fetchRoomPool      fastjson.ParserPool
	fetchLiveInfoPool  fastjson.ParserPool
	fetchMedalListPool fastjson.ParserPool
)

// 直播间的数据结构
type liveRoom struct {
	name      string // 主播名字
	title     string // 直播间标题
	liveID    string // 直播ID
	startTime int64  // 直播开始时间
}

// 守护徽章信息
type medalInfo struct {
	uid  int64  // 主播uid
	name string // 主播名字
}

// liveRoom的map
var liveRooms struct {
	sync.Mutex                   // rooms的锁
	rooms      map[int]*liveRoom // 现在的liveRoom
	newRooms   map[int]*liveRoom // 新的liveRoom
}

// liveRoom的pool
var liveRoomPool = &sync.Pool{
	New: func() interface{} {
		return new(liveRoom)
	},
}

// medalInfo的pool
var medalInfoPool = &sync.Pool{
	New: func() interface{} {
		return new(medalInfo)
	},
}

// 获取主播的直播链接
func getURL(uid int) string {
	const livePage = "https://live.acfun.cn/live/"
	return livePage + itoa(uid)
}

// 获取主播的直播链接
func (s *streamer) getURL() string {
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
		req.Header.SetMethod(fasthttp.MethodGet)
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

	req.Header.Set("Accept-Encoding", "gzip")

	err := c.client.Do(req, resp)
	checkErr(err)

	return resp, nil
}

// 获取响应body
func getBody(resp *fasthttp.Response) []byte {
	if string(resp.Header.Peek("content-encoding")) == "gzip" || string(resp.Header.Peek("Content-Encoding")) == "gzip" {
		body, err := resp.BodyGunzip()
		if err == nil {
			return body
		}
	}

	return resp.Body()
}

// 获取全部AcFun直播间
func fetchAllRooms() bool {
	for count := 10000; count < 1e9; count *= 10 {
		rooms, all, err := fetchLiveRoom(count)
		if err != nil {
			lPrintErr(err)
			return false
		}
		if all {
			liveRooms.newRooms = rooms
			return true
		}
		if count == 1e8 {
			lPrintErr("获取正在直播的直播间列表失败")
		}
	}
	return false
}

// 获取指定数量的AcFun直播间列表
func fetchLiveRoom(count int) (rooms map[int]*liveRoom, all bool, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("fetchLiveRoom() error: %w", err)
		}
	}()

	//const liveListURL = "https://live.acfun.cn/rest/pc-direct/live/channel"
	const liveListURL = "https://live.acfun.cn/api/channel/list?count=%d&pcursor=0"

	client := &httpClient{
		url:    fmt.Sprintf(liveListURL, count),
		method: fasthttp.MethodGet,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := getBody(resp)

	p := fetchRoomPool.Get()
	defer fetchRoomPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)
	v = v.Get("channelListData")
	if !v.Exists("result") || v.GetInt("result") != 0 {
		panic(fmt.Errorf("无法获取AcFun直播间列表，响应为：%s", string(body)))
	}

	if string(v.GetStringBytes("pcursor")) != "no_more" {
		return nil, false, nil
	}

	liveList := v.GetArray("liveList")
	rooms = make(map[int]*liveRoom, len(liveList))
	for _, live := range liveList {
		uid := live.GetInt("authorId")
		room := liveRoomPool.Get().(*liveRoom)
		room.name = string(live.GetStringBytes("user", "name"))
		room.title = string(live.GetStringBytes("title"))
		room.liveID = string(live.GetStringBytes("liveId"))
		room.startTime = live.GetInt64("createTime")
		rooms[uid] = room
	}

	return rooms, true, nil
}

// 根据uid获取主播的名字，可能需要检查返回是否为空
func getName(uid int) string {
	liveRooms.Lock()
	room, ok := liveRooms.rooms[uid]
	if ok {
		name := room.name
		liveRooms.Unlock()
		return name
	}
	liveRooms.Unlock()

	_, room, err := tryFetchLiveInfo(uid)
	if err != nil {
		return ""
	}
	defer liveRoomPool.Put(room)
	return room.name
}

// 根据uid获取主播直播间的标题
func getTitle(uid int) string {
	liveRooms.Lock()
	room, ok := liveRooms.rooms[uid]
	if ok {
		title := room.title
		liveRooms.Unlock()
		return title
	}
	liveRooms.Unlock()

	if isLive, room, err := tryFetchLiveInfo(uid); err == nil {
		defer liveRoomPool.Put(room)
		if isLive {
			return room.title
		}
	}
	return ""
}

// 根据uid获取liveID，结果准确，可能需要检查返回是否为空
func getLiveID(uid int) string {
	if isLive, room, err := tryFetchLiveInfo(uid); err == nil {
		defer liveRoomPool.Put(room)
		if isLive {
			return room.liveID
		}
	}
	return ""
}

// 根据uid获取直播开始时间，可能需要检查返回是否为0
func getStartTime(uid int) int64 {
	liveRooms.Lock()
	room, ok := liveRooms.rooms[uid]
	if ok {
		startTime := room.startTime
		liveRooms.Unlock()
		return startTime
	}
	liveRooms.Unlock()

	if isLive, room, err := tryFetchLiveInfo(uid); err == nil {
		defer liveRoomPool.Put(room)
		if isLive {
			return room.startTime
		}
	}
	return 0
}

// 根据uid查看主播是否正在直播
func isLiveOn(uid int) bool {
	liveRooms.Lock()
	_, ok := liveRooms.rooms[uid]
	liveRooms.Unlock()
	if ok {
		return true
	}

	if isLive, room, err := tryFetchLiveInfo(uid); err == nil {
		defer liveRoomPool.Put(room)
		return isLive
	}
	return false
}

// 获取主播直播间的标题
func (s *streamer) getTitle() string {
	return getTitle(s.UID)
}

// 获取liveID，由于AcFun的bug，结果不一定准确，可能需要检查返回是否为空
func (s *streamer) getLiveID() string {
	liveRooms.Lock()
	defer liveRooms.Unlock()
	room, ok := liveRooms.rooms[s.UID]
	if ok {
		return room.liveID
	}
	return ""
}

// 查看主播是否在直播，由于AcFun的bug，结果不一定准确
func (s *streamer) isLiveOn() bool {
	liveRooms.Lock()
	defer liveRooms.Unlock()
	_, ok := liveRooms.rooms[s.UID]
	return ok
}

// 获取用户直播相关信息，可能要将room放回liveRoomPool
func fetchLiveInfo(uid int) (isLive bool, room *liveRoom, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("fetchLiveInfo() error: %w", err)
		}
	}()

	//const acLiveInfo = "https://live.acfun.cn/rest/pc-direct/live/info"
	//const acLiveInfo = "https://api-new.acfunchina.com/rest/app/live/info?authorId=%d"
	//const acLiveInfo = "https://live.acfun.cn/rest/pc-direct/user/userInfo?userId=%d"
	const acLiveInfo = "https://live.acfun.cn/api/live/info?authorId=%d"

	client := &httpClient{
		url:    fmt.Sprintf(acLiveInfo, uid),
		method: fasthttp.MethodGet,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := getBody(resp)

	p := fetchLiveInfoPool.Get()
	defer fetchLiveInfoPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)
	if !v.Exists("result") || v.GetInt("result") != 0 {
		return false, nil, fmt.Errorf("无法获取uid为%d的主播的直播信息，响应为：%s", uid, string(body))
	}

	room = liveRoomPool.Get().(*liveRoom)
	if v.Exists("liveId") {
		isLive = true
		room.title = string(v.GetStringBytes("title"))
		room.liveID = string(v.GetStringBytes("liveId"))
		room.startTime = v.GetInt64("createTime")
	} else {
		isLive = false
		room.title = ""
		room.liveID = ""
		room.startTime = 0
	}

	room.name = string(v.GetStringBytes("user", "name"))

	return isLive, room, nil
}

// 获取登陆用户的守护徽章列表
func fetchMedalList() (medalList []*medalInfo, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("fetchMedalList() error: %w", err)
		}
	}()

	const medalListURL = "https://www.acfun.cn/rest/pc-direct/fansClub/fans/medal/list"

	if len(acfunCookies) == 0 {
		return nil, fmt.Errorf("没有登陆AcFun帐号")
	}

	client := &httpClient{
		url:     medalListURL,
		method:  fasthttp.MethodGet,
		cookies: acfunCookies,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := getBody(resp)

	p := fetchMedalListPool.Get()
	defer fetchMedalListPool.Put(p)
	v, err := p.ParseBytes(body)
	checkErr(err)

	if !v.Exists("result") || v.GetInt("result") != 0 {
		return nil, fmt.Errorf("获取登陆帐号拥有的守护徽章列表失败，响应为 %s", string(body))
	}

	list := v.GetArray("medalList")
	medalList = make([]*medalInfo, 0, len(list))
	for _, l := range list {
		medal := medalInfoPool.Get().(*medalInfo)
		medal.uid = l.GetInt64("uperId")
		medal.name = string(l.GetStringBytes("uperName"))
		medalList = append(medalList, medal)
	}

	return medalList, nil
}

// 获取用户直播相关信息，可能要将room放回liveRoomPool
func tryFetchLiveInfo(uid int) (isLive bool, room *liveRoom, err error) {
	err = run(func() (err error) {
		isLive, room, err = fetchLiveInfo(uid)
		return err
	})
	return isLive, room, err
}

// 通过wap版网页查看主播是否在直播
func (s *streamer) isLiveOnByPage() (isLive bool) {
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
		method:    fasthttp.MethodGet,
		userAgent: userAgent,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := getBody(resp)

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	checkErr(err)
	return doc.Find("p.closed-tip").Text() != "直播已结束"
}

// 获取AcFun的logo
func fetchAcLogo() {
	const acLogo = "https://cdn.aixifan.com/ico/favicon.ico"

	client := &httpClient{
		url:    acLogo,
		method: fasthttp.MethodGet,
	}
	resp, err := client.doRequest()
	checkErr(err)
	defer fasthttp.ReleaseResponse(resp)
	body := getBody(resp)

	newLogoFile, err := os.Create(logoFileLocation)
	checkErr(err)
	defer newLogoFile.Close()

	_, err = newLogoFile.Write(body)
	checkErr(err)
}

// 获取AcFun的直播源信息，分为hls和flv两种
func (s *streamer) getStreamInfo() (info streamInfo, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("getStreamURL() error: %w", err)
		}
	}()

	var ac *acfundanmu.AcFunLive
	var err error
	err = run(func() error {
		ac, err = acfundanmu.NewAcFunLive(acfundanmu.SetLiverUID(int64(s.UID)))
		return err
	})
	checkErr(err)
	sInfo := ac.GetStreamInfo()
	info.StreamInfo = *sInfo

	index := 0
	if s.Bitrate == 0 {
		// s.Bitrate为0时选择码率最高的直播源
		index = len(sInfo.StreamList) - 1
	} else {
		// 选择s.Bitrate下码率最高的直播源
		for i, stream := range sInfo.StreamList {
			if s.Bitrate >= stream.Bitrate {
				index = i
			} else {
				break
			}
		}
	}

	info.flvURL = sInfo.StreamList[index].URL

	bitrate := sInfo.StreamList[index].Bitrate
	switch {
	case bitrate >= 4000:
		info.cfg = subConfigs[1080]
	case len(sInfo.StreamList) >= 2 && bitrate >= 2000:
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

// 根据config.Source获取直播信息
func (s *streamer) getLiveInfo() (info liveInfo, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("getLiveInfo() error: %w", err)
		}
	}()

	info.uid = s.UID

	var err error
	info.streamInfo, err = s.getStreamInfo()
	checkErr(err)

	switch config.Source {
	case "hls":
		info.streamURL = info.hlsURL
	case "flv":
		info.streamURL = info.flvURL
	default:
		return info, fmt.Errorf("%s里的Source必须是hls或flv", configFile)
	}
	return info, nil
}

// 查看指定主播是否在直播和输出其直播源
func printStreamURL(uid int) (string, string) {
	s, ok := getStreamer(uid)
	if !ok {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return "", ""
		}
		s = streamer{UID: uid, Name: name}
	}

	if isLiveOn(s.UID) {
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
/*
func getLiveOnByInfo(ss []streamer) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, s := range ss {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			if isLive, room, err := tryFetchLiveInfo(uid); err == nil {
				if isLive {
					mu.Lock()
					liveRooms.newRooms[uid] = room
					mu.Unlock()
				} else {
					liveRoomPool.Put(room)
				}
			}
		}(s.UID)
	}
	wg.Wait()
}
*/

// 循环获取AcFun直播间数据
func cycleFetch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := fetchAllRooms(); ok {
				if len(liveRooms.newRooms) == 0 {
					lPrintWarn("没有人在直播")
				}
				/*
					streamers.Lock()
					notLive := make([]streamer, 0, len(streamers.crt))
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
				*/

				liveRooms.Lock()
				for _, room := range liveRooms.rooms {
					liveRoomPool.Put(room)
				}
				liveRooms.rooms = liveRooms.newRooms
				liveRooms.Unlock()
			}

			// 每10秒循环一次
			time.Sleep(10 * time.Second)
		}
	}
}
