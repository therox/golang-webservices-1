package main

import (
	"context"
	"encoding/json"
	"net/http"
)
 
func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler

	switch r.URL.Path { 
		case "/user/profile":
		switch r.Method { 	
		case "GET": 
				h = http.HandlerFunc(srv.ProfileHandler)	
		case "POST": 
				h = http.HandlerFunc(srv.ProfileHandler)
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			rd := responseData{
				Error: "bad method",
			}
			w.Write(rd.Create())
			return
		}
		case "/user/create":
		switch r.Method { 	
		case "POST": 
				h = http.HandlerFunc(srv.CreateHandler)
				h = AuthorizeWrappers(h)
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			rd := responseData{
				Error: "bad method",
			}
			w.Write(rd.Create())
			return
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		rd := responseData{
			Error: "unknown method",
		}
		w.Write(rd.Create())
		return
	}
	h.ServeHTTP(w, r)
}


func (srv *MyApi) ProfileHandler(w http.ResponseWriter, r *http.Request) {

	rd := &responseData{}
	
	
	ctx := context.Background()

	// Подготавливаем структуру для параметров
	in := ProfileParams{}
	// Парсим форму
	r.ParseForm()
	//Заполняем параметры
	err := PopulateStruct(&in, r.Form)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	err = validateStruct(in)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	// Нужно в конечном итоге запустить
	user, err := srv.Profile(ctx, in)
	if err != nil {
		if e, ok := err.(ApiError); ok {
			rd.Error = e.Err.Error()
			http.Error(w, string(rd.Create()), e.HTTPStatus)
			return
		}
		rd.Error = err.Error()
		http.Error(w, string(rd.Create()), http.StatusInternalServerError)
		return
	}
	bs, _ := json.Marshal(user)
	rd.Response = bs
	w.Write(rd.Create())
}

func (srv *MyApi) CreateHandler(w http.ResponseWriter, r *http.Request) {

	rd := &responseData{}
	
	
	ctx := context.Background()

	// Подготавливаем структуру для параметров
	in := CreateParams{}
	// Парсим форму
	r.ParseForm()
	//Заполняем параметры
	err := PopulateStruct(&in, r.Form)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	err = validateStruct(in)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	// Нужно в конечном итоге запустить
	user, err := srv.Create(ctx, in)
	if err != nil {
		if e, ok := err.(ApiError); ok {
			rd.Error = e.Err.Error()
			http.Error(w, string(rd.Create()), e.HTTPStatus)
			return
		}
		rd.Error = err.Error()
		http.Error(w, string(rd.Create()), http.StatusInternalServerError)
		return
	}
	bs, _ := json.Marshal(user)
	rd.Response = bs
	w.Write(rd.Create())
}
 
func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler

	switch r.URL.Path { 
		case "/user/create":
		switch r.Method { 	
		case "POST": 
				h = http.HandlerFunc(srv.CreateHandler)
				h = AuthorizeWrappers(h)
		default:
			w.WriteHeader(http.StatusNotAcceptable)
			rd := responseData{
				Error: "bad method",
			}
			w.Write(rd.Create())
			return
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		rd := responseData{
			Error: "unknown method",
		}
		w.Write(rd.Create())
		return
	}
	h.ServeHTTP(w, r)
}


func (srv *OtherApi) CreateHandler(w http.ResponseWriter, r *http.Request) {

	rd := &responseData{}
	
	
	ctx := context.Background()

	// Подготавливаем структуру для параметров
	in := OtherCreateParams{}
	// Парсим форму
	r.ParseForm()
	//Заполняем параметры
	err := PopulateStruct(&in, r.Form)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	err = validateStruct(in)
	if err != nil {
		rd.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		w.Write(rd.Create())
		return
	}
	// Нужно в конечном итоге запустить
	user, err := srv.Create(ctx, in)
	if err != nil {
		if e, ok := err.(ApiError); ok {
			rd.Error = e.Err.Error()
			http.Error(w, string(rd.Create()), e.HTTPStatus)
			return
		}
		rd.Error = err.Error()
		http.Error(w, string(rd.Create()), http.StatusInternalServerError)
		return
	}
	bs, _ := json.Marshal(user)
	rd.Response = bs
	w.Write(rd.Create())
}
