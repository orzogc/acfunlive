//爬虫相关
package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/orzogc/acfundanmu"
	"github.com/valyala/fastjson"
)

//const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36"
//const acUserInfo = "https://live.acfun.cn/rest/pc-direct/user/userInfo?userId=%d"
//const acAuthorID = "https://api-new.app.acfun.cn/rest/app/live/info?authorId=%d"

// 直播间的数据结构
type liveRoom struct {
	// 主播名字
	name string
	// 直播间标题
	title string
}

// liveRoom的map
var liveRooms struct {
	sync.Mutex
	rooms *map[int]liveRoom
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

// 获取全部AcFun直播间
func fetchAllRooms() {
	page := "0"
	allRooms := make(map[int]liveRoom)
	for page != "no_more" {
		rooms, nextPage := fetchLiveRoom(page)
		page = nextPage
		for uid, r := range *rooms {
			allRooms[uid] = r
		}
	}

	liveRooms.Lock()
	defer liveRooms.Unlock()
	liveRooms.rooms = &allRooms
}

// 获取指定页数的AcFun直播间
func fetchLiveRoom(page string) (r *map[int]liveRoom, nextPage string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in fetchLiveRoom(), the error is:", err)
			lPrintErr("获取AcFun直播间API的json时发生错误，尝试重新运行")
			// 延迟两秒，防止意外情况下刷屏
			time.Sleep(2 * time.Second)
			r, nextPage = fetchLiveRoom(page)
		}
	}()

	const acLive = "https://api-plus.app.acfun.cn/rest/app/live/channel"

	form := url.Values{}
	form.Set("count", "1000")
	form.Set("pcursor", page)

	resp, err := http.PostForm(acLive, form)
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 0 {
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

// 获取主播直播间的标题
func (s streamer) getTitleByInfo() string {
	v := getLiveInfo(s.UID)
	if v.Exists("title") {
		return string(v.GetStringBytes("title"))
	}
	return ""
}

// 查看主播是否在直播
func (s streamer) isLiveOnByInfo() bool {
	v := getLiveInfo(s.UID)
	if v.Exists("user", "liveId") {
		return true
	}
	return false
}

// 获取用户直播相关信息
func getLiveInfo(uid int) (v *fastjson.Value) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getLiveInfo(), the error is:", err)
			lPrintErr("获取uid为" + itoa(uid) + "的主播的直播信息时出错，尝试重新运行")
			time.Sleep(2 * time.Second)
			v = getLiveInfo(uid)
		}
	}()

	const acLiveInfo = "https://api-new.app.acfun.cn/rest/app/live/info?authorId=%d"

	resp, err := http.Get(fmt.Sprintf(acLiveInfo, uid))
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	var p fastjson.Parser
	v, err = p.ParseBytes(body)
	checkErr(err)

	return v
}

// 根据uid获取主播的名字
func getName(uid int) string {
	liveRooms.Lock()
	room, ok := (*liveRooms.rooms)[uid]
	liveRooms.Unlock()
	if ok {
		return room.name
	}

	v := getLiveInfo(uid)
	if v.Exists("user", "name") {
		return string(v.GetStringBytes("user", "name"))
	}
	return ""
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

	resp, err := http.Get(s.getURL())
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

	client := &http.Client{Timeout: 10 * time.Second}
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
		return "", "", "", cfg
	}
	// 获取userId和对应的令牌
	userID := v.GetInt("userId")
	serviceToken := string(v.GetStringBytes("acfun.api.visitor_st"))

	// 获取直播源的地址需要userId、did和对应的令牌
	streamURL := fmt.Sprintf(playURL, userID, deviceID, serviceToken)

	form = url.Values{}
	// authorId就是主播的uid
	form.Set("authorId", s.itoa())
	resp, err = http.PostForm(streamURL, form)
	checkErr(err)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	checkErr(err)

	v, err = p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 1 {
		return "", "", "", cfg
	}
	videoPlayRes := v.GetStringBytes("data", "videoPlayRes")
	v, err = p.ParseBytes(videoPlayRes)
	checkErr(err)
	streamName = string(v.GetStringBytes("streamName"))

	representation := v.GetArray("liveAdaptiveManifest", "0", "adaptationSet", "representation")

	// 选择s.Bitrate下码率最高的直播源
	index := 0
	if s.Bitrate == 0 {
		index = len(representation) - 1
	} else {
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
				if !s.isLiveOn() {
					notLive = append(notLive, s)
				}
			}
			streamers.Unlock()
			for _, s := range notLive {
				if s.isLiveOnByInfo() {
					title := s.getTitleByInfo()
					liveRooms.Lock()
					(*liveRooms.rooms)[s.UID] = liveRoom{name: s.Name, title: title}
					liveRooms.Unlock()
				}
			}
			// 每10秒循环一次
			time.Sleep(10 * time.Second)
		}
	}
}
