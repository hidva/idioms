package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/golang/glog"
)

const (
	kMsg500 = `{"success": false, "message": "内部错误"}`
	kMsgOK  = `{"success": true}`

	kJSONContentType = "application/json; charset=utf-8"

	kOffsetDef = 0
	kLengthDef = 16
)

/*
构造一个 JSON response 写入到 resp_writer 中.

*/
func httpJSONResponse(resp_writer http.ResponseWriter, status_code int, body string) error {
	resp_writer.Header().Set("Content-type", kJSONContentType)
	resp_writer.WriteHeader(status_code)
	write_rc, err := io.WriteString(resp_writer, body)
	if err != nil {
		err = fmt.Errorf("err, httpJSONResponse, WriteString, err: %q, sc: %d, body: %s", err, status_code, body)
	} else if write_rc != len(body) {
		err = fmt.Errorf("err, httpJSONResponse, WriteString, err: 未完全写入; expect: %d, actual: %d, sc: %d, body: %s",
			len(body), write_rc, status_code, body)
	}
	return err
}

func onError(url string, loc string, err error, resp http.ResponseWriter) {
	glog.Errorf("%s, %s, err: %s", url, loc, err)

	if resp != nil {
		if err = httpJSONResponse(resp, http.StatusInternalServerError, kMsg500); err != nil {
			glog.Errorf("%s, %s, httpJSONResponse, err: %s", url, err)
		}
	}
	return
}

func getIntForm(request *http.Request, key string) (int, error) {
	value := request.Form.Get(key)
	if len(value) != 0 {
		tmp, err := strconv.ParseInt(value, 0, 0)
		return int(tmp), err
	}
	return -12138, fmt.Errorf("request form not exists; key: %s", key)
}

func getOffsetLength(request *http.Request) (int, int) {
	offset, err := getIntForm(request, "o")
	if err != nil {
		offset = kOffsetDef
	}

	length, err := getIntForm(request, "l")
	if err != nil {
		length = kLengthDef
	}

	return offset, length
}

/*
/api/b?b=$WORD&o=3&l=10

b; 必选,
o; 默认为 0
l; 默认为 16

*/
func onApiB(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		onError("/api/b", "parse form", err, w)
		return
	}

	bword := req.Form.Get("b")
	if len(bword) == 0 {
		onError("/api/b", "get b form", fmt.Errorf("缺少 bword"), w)
		return
	}

	offset, length := getOffsetLength(req)

	response_bytes, err := json.Marshal(g_idiom_graph.BeginWith([]rune(bword)[0], offset, length))
	if err != nil {
		onError("/api/b", "json.Marshal", err, w)
		return
	}
	err = httpJSONResponse(w, 200, string(response_bytes))
	if err != nil {
		onError("/api/b", "httpJSONResponse", err, nil)
	}
	return
}

/*
/api/e?e=$WORD&o=3&l=10
*/
func onApiE(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		onError("/api/e", "parse form", err, w)
		return
	}

	eword := req.Form.Get("e")
	if len(eword) == 0 {
		onError("/api/e", "get e form", fmt.Errorf("缺少 eword"), w)
		return
	}

	offset, length := getOffsetLength(req)

	response_bytes, err := json.Marshal(g_idiom_graph.EndWith([]rune(eword)[0], offset, length))
	if err != nil {
		onError("/api/e", "json.Marshal", err, w)
		return
	}
	err = httpJSONResponse(w, 200, string(response_bytes))
	if err != nil {
		onError("/api/e", "httpJSONResponse", err, nil)
	}
	return
}

/*
/api/be?b=$WORD&e=$WORD&o=3&l=10
*/
func onApiBE(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		onError("/api/be", "parse form", err, w)
		return
	}

	eword := req.Form.Get("e")
	if len(eword) == 0 {
		onError("/api/be", "get e form", fmt.Errorf("缺少 eword"), w)
		return
	}

	bword := req.Form.Get("b")
	if len(bword) == 0 {
		onError("/api/be", "get b form", fmt.Errorf("缺少 bword"), w)
		return
	}

	offset, length := getOffsetLength(req)

	response_bytes, err := json.Marshal(
		g_idiom_graph.BeginEndWith([]rune(bword)[0], []rune(eword)[0],
			offset, length))
	if err != nil {
		onError("/api/be", "json.Marshal", err, w)
		return
	}
	err = httpJSONResponse(w, 200, string(response_bytes))
	if err != nil {
		onError("/api/be", "httpJSONResponse", err, nil)
	}
	return
}

/*
/api/path?b=$WORD&e=$WORD
*/
func onApiPath(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		onError("/api/path", "parse form", err, w)
		return
	}

	eword := req.Form.Get("e")
	if len(eword) == 0 {
		onError("/api/path", "get e form", fmt.Errorf("缺少 eword"), w)
		return
	}

	bword := req.Form.Get("b")
	if len(bword) == 0 {
		onError("/api/path", "get b form", fmt.Errorf("缺少 bword"), w)
		return
	}

	bnode := g_idiom_graph.Find(bword)
	enode := g_idiom_graph.Find(eword)
	if bnode == nil || enode == nil {
		err = httpJSONResponse(w, 400, `"你这输入的是成语么?!!!"`)
		if err != nil {
			onError("/api/path", "12138", err, nil)
		}
		return
	}

	response_bytes, err := json.Marshal(g_idiom_graph.ShortestPath(bnode, enode))
	if err != nil {
		onError("/api/path", "json.Marshal", err, w)
		return
	}
	err = httpJSONResponse(w, 200, string(response_bytes))
	if err != nil {
		onError("/api/path", "httpJSONResponse", err, nil)
	}
	return
}

func init() {
	http.HandleFunc("/api/b", onApiB)
	http.HandleFunc("/api/e", onApiE)
	http.HandleFunc("/api/be", onApiBE)
	http.HandleFunc("/api/path", onApiPath)
}
