package controller

import (
	"encoding/json"
	grpcClient "mumble-gateway-service/grpc_clients"
	"mumble-gateway-service/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

func GetContacts(w http.ResponseWriter, r *http.Request) {
	userDetails := r.Context().Value("userDetails").(jwt.MapClaims)

	// contacts, err := db.UserContacts(userDetails["UserId"].(int64))
	contacts, statusInt, err := grpcClient.GetContactsMethod(userDetails["UserId"].(int64))
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Printf("cntrl error: getting user's contacts from user service, userId: %d, err: %v", userDetails["UserId"].(int64), err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}
