package controller

// "mumble-message-service/redis"

// func GetMsgs(w http.ResponseWriter, r *http.Request) {
// 	userId := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(int64)
// 	contactId, err := strconv.Atoi(chi.URLParam(r, "id"))
// 	if err != nil {
// 		http.Error(w, "Invalid URL param", http.StatusBadRequest)
// 		return
// 	}

// 	var offSet int64
// 	if offsetParam := r.URL.Query().Get("skip"); offsetParam == "" {
// 		http.Error(w, "Missing query params", http.StatusForbidden)
// 		utils.Log.Println("client error: Missing query params", r.RemoteAddr)
// 		return

// 	} else if offSet, err = strconv.ParseInt(offsetParam, 10, 64); err != nil {
// 		http.Error(w, "Invalid query params", http.StatusBadRequest)
// 		utils.Log.Println("client error: Invalid query param", err, r.RemoteAddr)
// 		return
// 	}

// 	messages, err := db.ContactMsgs(userId, int64(contactId), offSet)
// 	if err != nil {
// 		http.Error(w, "Error getting your messages", http.StatusInternalServerError)
// 		utils.Log.Println("cntrl error: getting messages from db", err)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(messages)
// }

// func GetOnline(w http.ResponseWriter, r *http.Request) {
// 	// userId := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(int64)
// 	contactId, err := strconv.Atoi(chi.URLParam(r, "id"))
// 	if err != nil {
// 		http.Error(w, "Invalid URL param", http.StatusBadRequest)
// 		utils.Log.Println("client error: ", err, r.RemoteAddr)
// 		return
// 	}
// 	status := redis.IsUserOnline(int64(contactId))
// 	if status {
// 		var userOnline = types.UserOnline{UserOnlineStatus: true}
// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(userOnline)
// 		return
// 	} else {
// 		// userLastSeenTime, err := db.GetUserLastSeen(int64(contactId))
// 		// if err != nil {
// 		// 	http.Error(w, "Interval Server Error", http.StatusInternalServerError)
// 		// 	fmt.Println("Error getting user last seen comms.go", err)
// 		// 	return
// 		// }
// 		userLastSeen, err := redis.CheckUStatus(int64(contactId))
// 		if err != nil {
// 			http.Error(w, "Interval Server Error", http.StatusInternalServerError)
// 			//fmt.Println("Error getting user last seen comms.go", err)
// 			utils.Log.Println("cntrl error: getting user's last seen", err)
// 			return
// 		}
// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(userLastSeen)
// 	}
// }
