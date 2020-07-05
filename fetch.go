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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/orzogc/acfundanmu"
	"github.com/valyala/fastjson"
)

const livePage = "https://live.acfun.cn/live/"

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
	return livePage + itoa(uid)
}

// 获取主播的直播链接
func (s streamer) getURL() string {
	return livePage + s.itoa()
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

	const acLive = "https://live.acfun.cn/api/channel/list?pcursor=%s"

	resp, err := http.Get(fmt.Sprintf(acLive, page))
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("channelListData", "result") != 0 {
		return nil, ""
	}

	var rooms = make(map[int]liveRoom)
	liveList := v.GetArray("channelListData", "liveList")
	for _, live := range liveList {
		uid := live.GetInt("authorId")
		room := liveRoom{
			name:  string(live.GetStringBytes("user", "name")),
			title: string(live.GetStringBytes("title")),
		}
		rooms[uid] = room
	}

	nextPage = string(v.GetStringBytes("channelListData", "pcursor"))

	return &rooms, nextPage
}

// 查看主播是否在直播
func (s streamer) isLiveOn() bool {
	liveRooms.Lock()
	defer liveRooms.Unlock()
	_, ok := (*liveRooms.rooms)[s.UID]
	return ok
}

// 获取主播直播的标题
func (s streamer) getTitle() string {
	liveRooms.Lock()
	defer liveRooms.Unlock()
	if room, ok := (*liveRooms.rooms)[s.UID]; ok {
		return room.title
	}
	return ""
}

// 根据uid获取主播的名字
func getName(uid int) (name string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getName(), the error is:", err)
			lPrintErr("获取uid为" + itoa(uid) + "的主播的名字时出现错误，尝试重新运行")
			time.Sleep(2 * time.Second)
			name = getName(uid)
		}
	}()

	liveRooms.Lock()
	if room, ok := (*liveRooms.rooms)[uid]; ok {
		liveRooms.Unlock()
		return room.name
	}
	liveRooms.Unlock()

	const acUser = "https://www.acfun.cn/rest/pc-direct/user/userInfo?userId=%d"
	const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36"

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf(acUser, uid), nil)
	checkErr(err)
	// 需要浏览器user-agent
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	checkErr(err)
	if v.GetInt("result") != 0 {
		return ""
	}

	return string(v.GetStringBytes("profile", "name"))
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

	// 选择码率最高的flv视频源
	sort.Slice(representation, func(i, j int) bool {
		return representation[i].GetInt("bitrate") > representation[j].GetInt("bitrate")
	})
	flvURL = string(representation[0].GetStringBytes("url"))

	bitrate := representation[0].GetInt("bitrate")
	switch {
	case bitrate >= 4000:
		cfg = subConfigs[1080]
	case bitrate >= 3000:
		cfg = subConfigs[720]
	case bitrate == 0:
		cfg = subConfigs[0]
	default:
		cfg = subConfigs[540]
	}

	i := strings.Index(flvURL, streamName)
	// 这是码率最高的hls视频源
	hlsURL = strings.ReplaceAll(flvURL[0:i], "pull", "hlspull") + streamName + ".m3u8"

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
	s := streamer{UID: uid, Name: name}

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
			// 每10秒循环一次
			time.Sleep(10 * time.Second)
		}
	}
}
