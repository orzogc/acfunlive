package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var port = ":51880"

var dispatch = map[string]func(uint) bool{
	"addnotify":   addNotify,
	"delnotify":   delNotify,
	"addrecord":   addRecord,
	"delrecord":   delRecord,
	"startrecord": startRec,
	"stoprecord":  stopRec,
}

func handleDispatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := strconv.Atoi(vars["uid"])
	checkErr(err)
	fmt.Fprint(w, dispatch[mux.CurrentRoute(r).GetName()](uint(uid)))
}

func handleStreamURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := strconv.Atoi(vars["uid"])
	checkErr(err)
	hlsURL, flvURL := printStreamURL(uint(uid))
	fmt.Fprint(w, hlsURL+"\n"+flvURL)
}

func handleListLive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, listLive())
}

func handleListRecord(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, listRecord())
}

func handleQuit(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "quit")
	quitRun()
}

func server() {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in server(), the error is:", err)
			timePrintln("web服务器发生错误，尝试重启web服务器")
			time.Sleep(2 * time.Second)
			go server()
		}
	}()

	r := mux.NewRouter()
	s := r.Methods("GET").Subrouter()
	for str := range dispatch {
		s.HandleFunc(fmt.Sprintf("/%s/{uid:[1-9][0-9]*}", str), handleDispatch).Name(str)
	}
	s.HandleFunc("/getdlurl/{uid:[1-9][0-9]*}", handleStreamURL)
	s.HandleFunc("/listlive", handleListLive)
	s.HandleFunc("/listrecord", handleListRecord)
	s.HandleFunc("/quit", handleQuit)

	log.Fatal(http.ListenAndServe(port, s))
}
