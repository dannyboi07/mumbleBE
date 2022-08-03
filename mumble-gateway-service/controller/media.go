package controller

import (
	"io"
	"mumble-gateway-service/s3Media"
	"mumble-gateway-service/utils"
	"net/http"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-chi/chi/v5"
)

func GetFromS3(w http.ResponseWriter, r *http.Request) {
	var objFolder string = chi.URLParam(r, "objFolder")
	var objName string = chi.URLParam(r, "objName")
	if objFolder == "" || objName == "" {
		http.Error(w, "Invalid media request", http.StatusBadRequest)
		return
	}

	var (
		imgResult *s3.GetObjectOutput
		errString string
		errCode   int
	)
	imgResult, errString, errCode = s3Media.GetS3Img(objFolder + "/" + objName)
	if errCode != 0 {
		http.Error(w, errString, errCode)
		return
	}
	defer imgResult.Body.Close()

	w.Header().Set("Content-Type", *imgResult.ContentType)
	_, err := io.Copy(w, imgResult.Body)
	if err != nil {
		utils.Log.Println("err getting media from s3, err:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
