package main

import (
		"github.com/vanderbr/controller"
		"github.com/julienschmidt/httprouter"
		"github.com/syndtr/goleveldb/leveldb/util"
		"strings"
		"encoding/json"
		"io/ioutil"
		"net/http"
		//"strconv"
		"fmt"
		)

const (
		profiles_db = "pfd"
		tags_index_db = "tid"
		user_index_db = "uid"
		)

type User map[string]string
type Profile map[string]string

var app *ctrl.Ctrl

var tempUsers []User

func main() {
	
	app = ctrl.Start(false, []string{profiles_db, tags_index_db, user_index_db})

	tempUsers = []User{}
	NewUsers([]string{"alex","dave","clive","paul","ben","mark","william","chantelle","lucy","carol","jane"})
	// server

	router := httprouter.New()
	
	// profiles
	router.GET("/profile/:username", profilesGetHandler)
	router.POST("/profile/:tagname", profilesPostHandler)

	router.GET("/profiles/:tags", profilesSearchHandler)

	host := "leadinglocally.cryptoapi.info"
	port := ":443"

	cert, key := app.UseCertificates(host)

	app.Log(host+" Listening on port "+port)

	panic(http.Serve(ctrl.NewTLSListener(port, cert, key), router))
}

func NewUsers(names []string) bool {
	for _, username := range names {
		u := make(map[string]string)
		u["username"] = username
		u["name"] = "Random Username"
		u["title"] = "This is my default title string!"
		u["description"] = "blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah blah "
		b, err := json.Marshal(u); if err != nil { return app.LogErr("NewUsers", err) }
		app.Send(profiles_db, username, b)
		AddUserTag(username, "golang")
	}
	return true
}

// string tools

func Clean(s string) string { return strings.ToLower(strings.Replace(s, " ", "", -1)) }
func Sanitize(s string) string { return s }

// user tools

func UsernameExists(u string) bool {
	ok, _ := app.Get("profiles", u)
	return ok
}

// handlers

func ValidLength(i int, s string) (bool, string) { l := len(s); if l > 0 && l <= i { if len(strings.Replace(s, " ", "", -1)) > 3 { return true, Sanitize(s) } }; return false, "" }

func GetProfile(id string) (bool, Profile) {
	ok, p := app.Get(profiles_db, id)
	for ok {
		var np Profile
		err := json.Unmarshal(p, &np); if err != nil { app.LogErr("GetProfile", err); break }
		return true, np
	}
	return false, nil
}

type QIndex map[string]bool

func profilesSearchHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {
	mem := []QIndex{}
	tags := strings.Split(p.ByName("tags"), "-")
	
	for _, tag := range tags {
		app.Log("TAG: "+tag)
		index := make(QIndex)
		ok, list := GetAssociated(tags_index_db, tag)
		app.DebugJSON(list)
		if ok { for _, username := range list { index[username] = true } }
		mem = append(mem, index)
	}

	if len(mem) > 1 {
		for k, _ := range mem[0] {
			for _, userMap := range mem[1:] { if !userMap[k] { delete(mem[0], k) } }
		}
	}

	out := "["

	app.DebugJSON(mem[0])

	for id, _ := range mem[0] {
		ok, p := app.Get(profiles_db, id)
		if ok {
			out += string(p)+","
		}
	}

	fmt.Fprintf(res, out+"null]")

	return
}

func profilesGetHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {

}

func profilesPostHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {

	thisFunc := "profilesPostHandler"
	msg := "Invalid API request!"
	code := 400

	for {
		
		username := Clean(p.ByName("username"))
		if UsernameExists(username) { msg = "username exists in system"; break }

		b, err := ioutil.ReadAll(r.Body); if err != nil { app.LogErr(thisFunc, err); break }
		
		// create new profile and add data to structure
		np := make(Profile)
		err = json.Unmarshal(b, &np); if err != nil { app.LogErr(thisFunc, err); break }
		// add authenticated username
		np["username"] = username
		// sanitize all rows
		for k, v := range np {
			switch k {
				case "username": 		continue
				case "name": 			ValidLength(20, v)
				case "title": 			ValidLength(40, v)
				case "description": 	ValidLength(360, v)
				default: 				msg = "invalid object parameter"; break
			}
		}
		
		msg = "internal response error"
		code = 500

		newProfile, err := json.Marshal(np); if err != nil { app.LogErr(thisFunc, err); break }

		if !app.Send(profiles_db, username, newProfile) { app.LogErr(thisFunc, err); break }

		return

	}

	app.HttpError(res, msg, code)
}


func AddUserTag(userName, tagName string) {
	value := []byte("X")
	app.Send(tags_index_db, tagName+"_"+userName, value)
	app.Send(user_index_db, userName+"_"+tagName, value)
}

func RemoveUserTag(userName, tagName string) {
	app.Delete(tags_index_db, tagName+"_"+userName)
	app.Delete(user_index_db, userName+"_"+tagName)
}

func GetAssociated(index, id string) (bool, []string) {
	app.Log("ASSOC: "+index+" "+id)
	for {
		list := []string{}
		iter := app.LevelDB(index).NewIterator(util.BytesPrefix([]byte(id+"_")), nil)
		for iter.Next() { list = append(list, strings.Split(string(iter.Key()), "_")[1]) }
		iter.Release()
		if !app.ChkErr("GetAssociated", iter.Error()) { break }
		return true, list
	}	
	return false, nil
}



