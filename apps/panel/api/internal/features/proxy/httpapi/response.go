package proxyhttpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	proxyservice "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/service"
)

type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func writeSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeEnvelope(w, status, APIResponse[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeEnvelope(w, status, APIResponse[any]{
		Code:    status,
		Message: message,
	})
}

func writeServiceError(w http.ResponseWriter, req *http.Request, err error, fallback string) {
	if err == nil {
		writeError(w, http.StatusInternalServerError, fallback)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if serviceErr, ok := err.(*proxyservice.Error); ok {
		writeError(w, serviceErr.Status, serviceErr.Message)
		return
	}
	log.Printf("link http service error method=%s path=%s code=%s err=%v", req.Method, req.URL.Path, fallback, err)
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeEnvelope[T any](w http.ResponseWriter, status int, payload APIResponse[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeMethodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
}
