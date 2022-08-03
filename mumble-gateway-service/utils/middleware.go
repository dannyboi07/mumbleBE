package utils

import (
	"context"
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		accessToken, err := r.Cookie("accessToken")
		if err != nil {
			http.Error(w, "Missing access token", http.StatusUnauthorized)
			return
		}

		mapClaims, err, statusCode := VerifyUserToken(accessToken.Value)
		if err != nil {
			http.Error(w, err.Error(), statusCode)
			Log.Println("midwre/client error: ", err, r.RemoteAddr)
			return
		}

		if userId, ok := mapClaims["UserId"].(float64); !ok {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			Log.Println("err typecasting userId:", userId)
			return
		} else {
			mapClaims["UserId"] = int64(userId)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "userDetails", mapClaims)))
	})
}

// exists, err := db.UserExistsById(int64(mapClaims["UserId"].(float64)))
// if err != nil {
// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 	Log.Println("midwre error: checking user existence", err)
// 	return
// } else if !exists {
// 	http.Error(w, "User doesn't exist", http.StatusUnauthorized)
// 	Log.Println("midwre-client error: user not found", r.RemoteAddr)
// 	return
// }

// func AuthMiddleware(w http.ResponseWriter, r *http.Request) {
// 	unverifiedToken := strings.Split(r.Header.Get("Authorization"), "Bearer ")[1]
// 	if unverifiedToken == "" {
// 		http.Error(w, "Missing token", http.StatusBadRequest)
// 		return
// 	}
// 	mapClaims, err, statusCode := VerifyUserToken(unverifiedToken)
// 	if err != nil {
// 		http.Error(w, err.Error(), statusCode)
// 		return
// 	}
// 	fmt.Println("claims", mapClaims, "email", mapClaims["UserId"])
// 	exists, err := db.UserExistsById(mapClaims["UserId"].(int64))
// 	if err != nil {
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	} else if !exists {
// 		http.Error(w, "User doesn't exist", http.StatusUnauthorized)
// 		return
// 	}
// 	ctxWithUserDetails := context.WithValue(r.Context(), "userDetails", mapClaims)
// 	http.serhtt
// 	// r.Context()
// 	// fmt.Println("req headers", strings.Split(r.Header.Get("Authorization"), "Bearer "))
// 	// fmt.Println("req context", r.Context())
// }
