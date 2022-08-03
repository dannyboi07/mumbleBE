package controller

import (

	// "mumble-gateway-service/db"
	msgClient "mumble-gateway-service/grpc_clients"
	"mumble-gateway-service/utils"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
)

func GetMsgs(w http.ResponseWriter, r *http.Request) {

	var (
		userId, contactId int64
		err               error
	)
	// Get uId of requester
	userId = r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(int64)
	// Get friend's uId from url
	contactId, err = strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid URL param", http.StatusBadRequest)
		return
	}

	// Get msg offset value
	var offSet int64
	if offsetParam := r.URL.Query().Get("skip"); offsetParam == "" {
		http.Error(w, "Missing query params", http.StatusForbidden)
		utils.Log.Println("client error: Missing query params", r.RemoteAddr)
		return
	} else if offSet, err = strconv.ParseInt(offsetParam, 10, 64); err != nil {
		http.Error(w, "Invalid query params", http.StatusBadRequest)
		utils.Log.Println("client error: Invalid query param", err, r.RemoteAddr)
		return
	}

	var messages []byte
	messages, err = msgClient.GetMsgsMethod(userId, contactId, offSet)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("cntrl err: calling grpc method, err:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(messages)
}
