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
	"time"

	"github.com/valyala/fastjson"
)

const acLive = "https://live.acfun.cn/api/channel/list?pcursor=%s"

type liveRoom struct {
	// 主播名字
	id string
	// 直播间标题
	title string
}

// map[uint]liveRoom
var liveRooms *map[uint]liveRoom

func fetchAllRooms() {
	page := "0"
	var allRooms = make(map[uint]liveRoom)
	for page != "no_more" {
		rooms, nextPage := fetchLiveRoom(page)
		page = nextPage
		for uid, room := range *rooms {
			allRooms[uid] = room
		}
	}

	liveRooms = &allRooms
}

// 获取AcFun直播间API的json
func fetchLiveRoom(page string) (r *map[uint]liveRoom, nextPage string) {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in fetchLiveRoom(), the error is:", err)
			timePrintln("获取AcFun直播间API的json时发生错误，尝试重新运行")
			// 延迟两秒，防止意外情况下刷屏
			time.Sleep(2 * time.Second)
			r, nextPage = fetchLiveRoom(page)
		}
	}()

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

	var rooms = make(map[uint]liveRoom)
	liveList := v.GetArray("channelListData", "liveList")
	for _, live := range liveList {
		uid := live.GetUint("authorId")
		room := liveRoom{
			id:    string(live.GetStringBytes("user", "name")),
			title: string(live.GetStringBytes("title")),
		}
		rooms[uid] = room
	}

	nextPage = string(v.GetStringBytes("channelListData", "pcursor"))

	return &rooms, nextPage
}

// 查看主播是否在直播
func (s streamer) isLiveOn() bool {
	_, ok := (*liveRooms)[s.UID]
	return ok
}

// 获取主播直播的标题
func (s streamer) getTitle() string {
	room, ok := (*liveRooms)[s.UID]
	if ok {
		return room.title
	}
	return ""
}

// 根据uid获取主播的id
func getID(uid uint) (id string) {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in getID(), the error is:", err)
			timePrintln("获取uid为" + strconv.Itoa(int(uid)) + "的主播的ID时出现错误，尝试重新运行")
			time.Sleep(2 * time.Second)
			id = getID(uid)
		}
	}()

	const acUser = "https://www.acfun.cn/rest/pc-direct/user/userInfo?userId="
	userAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36"

	client := &http.Client{}
	req, err := http.NewRequest("GET", acUser+strconv.Itoa(int(uid)), nil)
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
func (s streamer) getStreamURL() (hlsURL string, flvURL string) {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in getStreamURL(), the error is:", err)
			timePrintln("获取" + s.longID() + "的直播源时出错，尝试重新运行")
			time.Sleep(2 * time.Second)
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
