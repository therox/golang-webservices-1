package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// код писать тут

func TestSearchClient_FindUsers(t *testing.T) {
	// Создаём сервер
	var withTimeout bool
	var returnStatusCode int
	var returnStatusBadRequestBody string
	var returnBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if withTimeout {
			time.Sleep(2 * time.Second)
		}
		w.WriteHeader(returnStatusCode)
		if returnStatusCode == http.StatusBadRequest {
			w.Write([]byte(returnStatusBadRequestBody))
		}
		if len(returnBody) > 0 {
			w.Write(returnBody)
		}

	}))
	defer server.Close()

	var cl SearchClient
	var req SearchRequest

	reset := func() {
		req = SearchRequest{
			Limit:      0,
			Offset:     0,
			Query:      "",
			OrderField: "",
			OrderBy:    0,
		}
		cl = SearchClient{
			AccessToken: "",
			URL:         server.URL,
		}
		returnStatusCode = http.StatusOK
		withTimeout = false
		returnStatusBadRequestBody = ""
	}

	t.Run("Test Limit <0 ", func(t *testing.T) {
		reset()
		req.Limit = -1
		wantErr := true
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)
	})

	t.Run("Limit >25", func(t *testing.T) {
		reset()
		req.Limit = 26
		wantErr := true
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)

	})
	t.Run("Offset <0", func(t *testing.T) {
		reset()
		req.Offset = -1
		wantErr := true
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)
	})
	t.Run("Timeout > 1s", func(t *testing.T) {
		reset()
		withTimeout = true
		wantErr := true
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)
	})
	t.Run("Server stopped", func(t *testing.T) {
		reset()
		cl.URL = ""
		wantErr := true
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)
	})
	t.Run("Checking error statuses", func(t *testing.T) {
		cases := []struct {
			status     int
			returnBody string
		}{
			{
				http.StatusUnauthorized,
				"",
			},
			{
				http.StatusInternalServerError,
				"",
			},
			{
				http.StatusBadRequest,
				"",
			},
			{
				http.StatusBadRequest,
				"{\"error\": \"ErrorBadOrderField\"}",
			},
			{
				http.StatusBadRequest,
				"{\"error\": \"Unknown error\"}",
			},
		}
		for _, tt := range cases {
			reset()
			returnStatusCode = tt.status
			returnStatusBadRequestBody = tt.returnBody
			wantErr := true
			_, err := cl.FindUsers(req)
			assertHasError(t, err, wantErr)
		}
	})
	t.Run("Return correct Data", func(t *testing.T) {
		u := []User{
			{
				Id:     1,
				Name:   "User 1",
				Age:    1,
				About:  "About 1",
				Gender: "Male",
			},
			{
				Id:     2,
				Name:   "User 2",
				Age:    1,
				About:  "About 2",
				Gender: "Female",
			},
		}
		// Проверяем выдачу без ограничения лимита
		reset()
		bs, _ := json.Marshal(u)
		req.Limit = 2
		wantErr := false
		returnBody = bs
		_, err := cl.FindUsers(req)
		assertHasError(t, err, wantErr)

		// Проверяем выдачу с ограничением лимита
		reset()
		bs, _ = json.Marshal(u)
		req.Limit = 1
		wantErr = false
		returnBody = bs
		_, err = cl.FindUsers(req)
		assertHasError(t, err, wantErr)

	})
}

func assertHasError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		t.Errorf("FindUsers() error = %v, wantErr %v", err, wantErr)
	}
}
