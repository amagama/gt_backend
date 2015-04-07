package main

import (
		"github.com/vanderbr/controller"
		"github.com/julienschmidt/httprouter"
		"github.com/syndtr/goleveldb/leveldb/util"
		"strings"
		"encoding/json"
		"io/ioutil"
		"net/http"
		"strconv"
		"fmt"
		)

type Profile map[string]string

var app *ctrl.Ctrl

func main() {
	
	app = ctrl.Start(false, []string{"profiles", "tags"})

	ok, num := app.Get("entities", "storedObjects"); if !ok { app.Err("GET FAILED"); app.KillProgram(nil) }
	fmt.Println("storedObjects", string(num))

	router := httprouter.New()
	
	// profiles
	router.GET("/profiles/:username", profilesGetHandler)
	router.POST("/profiles/:username", profilesPostHandler)
	
	// tags
	router.GET("/tags/:tagname", tagsGetHandler)
	router.POST("/tags/:tagname", tagsPostHandler)

	host := "leadinglocally.cryptoapi.info"
	port := ":443"

	cert, key := app.UseCertificates(host)

	app.Log(host+" Listening on port "+port)

	panic(http.Serve(ctrl.NewTLSListener(port, cert, key), router))
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

func ValidLength(i int, s string) (bool, string) {
	l := len(s)
	if l > 0 && l <= i {
		if len(strings.Replace(s, " ", "", -1)) > 3 { return true, Sanitize(s) }
	}
	return false, ""
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

		if !app.Send("profiles", username, newProfile) { app.LogErr(thisFunc, err); break }

		return

	}

	app.HttpError(res, msg, code)
}

func tagsGetHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {
	thisFunc := "tagsGetHandler"
	res.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))

	for {
		searchTerm := []byte(Clean(p.ByName("tagname")))
		quantity, err := strconv.Atoi(p.ByName("quantity")); if !app.ChkErr(thisFunc, err) { break }

		iter := app.LevelDB("tags").NewIterator(util.BytesPrefix(searchTerm), nil)
		i := 0

		bk := "\","
		out := "["

		for iter.Next() && i < quantity {
			out += "\""+string(iter.Value())+bk
			i++
		}
		fmt.Fprintf(res, out+"\"\"]")

		iter.Release()
		if !app.ChkErr(thisFunc, iter.Error()) { break }
		return
	}
	// send http error response code
	app.HttpError(res, "SEARCH REQUEST FAILED", 400)
}

func tagsPostHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {
}







