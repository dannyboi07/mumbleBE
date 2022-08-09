package controller

import (
	"bufio"
	"encoding/json"
	"io"
	"mime/multipart"
	grpcClients "mumble-gateway-service/grpc_clients"
	"mumble-gateway-service/s3Media"
	"mumble-gateway-service/types"
	"mumble-gateway-service/user_service_proto"
	"mumble-gateway-service/utils"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/golang-jwt/jwt/v4"
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var maxFileSize int = 1000 * 1000 * 2
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxFileSize+2000000))
	if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var (
		reader *multipart.Reader
		err    error
	)
	reader, err = r.MultipartReader()
	if err != nil {
		http.Error(w, "Interval Server Error", http.StatusInternalServerError)
		utils.Log.Println("cntrl error: initing multipart reader", err)
		return
	}

	// Section: Validate name
	var nameField *multipart.Part

	nameField, err = reader.NextPart()
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		utils.Log.Println("client error: nextPart:nameField", err, r.RemoteAddr)
		return
	} else if err == io.EOF || nameField.FormName() != "name" {
		http.Error(w, "Expected missing field: name", http.StatusBadRequest)
		utils.Log.Println("client error: 'name' field name not found", r.RemoteAddr)
		return
	}

	var name string = utils.ReadPartToString(nameField)
	if !utils.ValidName(&name) {
		http.Error(w, "Invalid name, must be greater than 3 and consist of only alphabetic characters", http.StatusBadRequest)
		return
	}

	// Section: Validate email
	var emailPart *multipart.Part
	emailPart, err = reader.NextPart()
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		utils.Log.Println("client error: nextPart:emailField", err, r.RemoteAddr)
		return
	} else if err == io.EOF || emailPart.FormName() != "email" {
		http.Error(w, "Expected missing field: email", http.StatusBadRequest)
		utils.Log.Println("client error: 'email' field name not found", r.RemoteAddr)
		return
	}

	var email string = utils.ReadPartToString(emailPart)

	// Section: Validate password
	var passwordField *multipart.Part

	passwordField, err = reader.NextPart()
	if err != nil && err != io.EOF {

		http.Error(w, "Missing field: password", http.StatusBadRequest)
		utils.Log.Println("client error: nextPart:password", err, r.RemoteAddr)
		return

	} else if err == io.EOF || passwordField.FormName() != "password" {

		http.Error(w, "Missing field: password", http.StatusBadRequest)
		utils.Log.Println("client error: 'password' field name not found", r.RemoteAddr)
		return

	}
	var password string = utils.ReadPartToString(passwordField)

	// Section: Validate profile picture
	var profilePic *multipart.Part

	profilePic, err = reader.NextPart()
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		utils.Log.Println("cntrl error: ", err)
		return
	} else if err == io.EOF {

		var picStr string = ""
		statusInt, err := grpcClients.RegisterUserMethod(types.RegisterUser{
			Name:        &name,
			Email:       &email,
			Password:    &password,
			Profile_Pic: &picStr,
		})
		if err != nil {
			http.Error(w, err.Error(), statusInt)
			utils.Log.Println("grpc err: registering user, err:", err)
			return
		}

		w.WriteHeader(http.StatusOK)
	} else if err != io.EOF {
		var (
			validFile bool
			buf       *bufio.Reader
			fileMime  *mimetype.MIME
		)
		validFile, buf, fileMime = utils.ValidFileType(profilePic, &utils.ProfImgValRegEx)
		if !validFile {
			http.Error(w, "Unacceptable file type", http.StatusUnprocessableEntity)
			utils.Log.Println("client error: 'Invalid prof-pic file type'", r.RemoteAddr)
			return
		}

		var (
			parsedFile   io.Reader
			statusInt    int
			err          error
			fileRandLink string
		)
		parsedFile, fileRandLink, statusInt = utils.ValidFile(buf, profilePic, fileMime.String(), maxFileSize)
		switch statusInt {
		case 0:
			break
		case 413:
			http.Error(w, "Profile picture is too large, max size of 2MB", statusInt)
			utils.Log.Println("client err: Profile picture over file size limit", r.RemoteAddr)
			return
		case 500:
			http.Error(w, "Internal server error", statusInt)
			utils.Log.Println("cntrl err:", r.RemoteAddr)
			return
		}

		fileFullLink := "https://mumble.daniel-dev.tech/mumbleapi/media/profile-images/" + fileRandLink
		statusInt, err = grpcClients.RegisterUserMethod(types.RegisterUser{
			Name:        &name,
			Email:       &email,
			Password:    &password,
			Profile_Pic: &fileFullLink,
		})
		if err != nil {
			http.Error(w, err.Error(), statusInt)
			utils.Log.Println("grpc err: registering user, err:", err, r.RemoteAddr)
			return
		}

		err = s3Media.S3UploadImage(parsedFile, "profile-images/"+fileRandLink, fileMime.String())
		if err != nil {
			utils.Log.Println("err uploading profile pic while registering user, err:", err)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1000000)
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Inacceptable content type", http.StatusBadRequest)
		utils.Log.Println("client error: invalid content type", r.RemoteAddr)
		return
	}

	jDec := json.NewDecoder(r.Body)
	jDec.DisallowUnknownFields()

	// Validate JSON containing user details
	var (
		userLogin  types.LoginUser
		statusCode int
		err        error
	)
	statusCode, err = utils.JsonReqErrCheck(jDec.Decode(&userLogin))
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		utils.Log.Println("cntrl/client error: ", err)
		return
	}
	if userLogin.Email == nil {
		http.Error(w, "Email field is empty", http.StatusBadRequest)
		utils.Log.Println("client error: Missing email field", r.RemoteAddr)
		return
	} else if userLogin.Password == nil {
		http.Error(w, "Password field is empty", http.StatusBadRequest)
		utils.Log.Println("client error: Missing password field", r.RemoteAddr)
		return
	}

	loginResp, statusInt, err := grpcClients.LoginUserMethod(userLogin)
	if err != nil {
		utils.Log.Println("err logging in user, err:", err, r.RemoteAddr)
		http.Error(w, err.Error(), statusInt)
		return
	}
	user := types.User{
		UserId:      loginResp.UserId,
		Name:        loginResp.Name,
		Email:       loginResp.Email,
		Profile_pic: loginResp.ProfilePic,
	}

	var (
		accTkCookie *http.Cookie
		refTkCookie *http.Cookie
	)
	accTkCookie = &http.Cookie{Name: "accessToken", Value: loginResp.AccessToken, MaxAge: int(loginResp.AccessTokenExp), Path: "/mumbleapi", HttpOnly: true, Secure: true, SameSite: http.SameSiteDefaultMode}
	refTkCookie = &http.Cookie{Name: "refreshToken", Value: loginResp.RefreshToken, MaxAge: int(loginResp.RefreshTokenExp), Path: "/mumbleapi/auth", HttpOnly: true, Secure: true, SameSite: http.SameSiteDefaultMode}
	http.SetCookie(w, accTkCookie)
	http.SetCookie(w, refTkCookie)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {

	var (
		accTkCookie *http.Cookie
		err         error
	)
	accTkCookie, err = r.Cookie("accessToken")

	// If access token cookie didn't expire by MAX AGE, access it and mark it for deletion
	if err == nil {
		accTkCookie.MaxAge = -1
		accTkCookie.Path = "/api"
		http.SetCookie(w, accTkCookie)
	}

	var refTkCookie *http.Cookie
	refTkCookie, err = r.Cookie("refreshToken")
	if err == nil {

		var mapClaims jwt.MapClaims
		mapClaims, err, statusInt := utils.VerifyUserToken(refTkCookie.Value)
		if err == nil || err != nil && err.Error() == "Token expired" {

			if userIdFloat, ok := mapClaims["UserId"].(float64); !ok {
				http.Error(w, "Malformed token", http.StatusForbidden)
				utils.Log.Println("Malformed refresh token, err:", err, r.RemoteAddr)
				return

			} else {
				var userId int64 = int64(userIdFloat)
				statusInt, err = grpcClients.LogoutMethod(userId)
				if err != nil {
					http.Error(w, err.Error(), statusInt)
					utils.Log.Println("grpc err requesting for logout, err:", err, r.RemoteAddr)
				}
			}

		} else if err != nil {
			http.Error(w, err.Error(), statusInt)
			utils.Log.Println("err verifying refTk, err:", err, r.RemoteAddr)
			// Not returning deliberately
		}

		refTkCookie.MaxAge = -1
		refTkCookie.Path = "/api/auth"

		http.SetCookie(w, refTkCookie)
	}

	w.WriteHeader(http.StatusOK)
}

func ChangePassword(w http.ResponseWriter, r *http.Request) {

	r.Body = http.MaxBytesReader(w, r.Body, 1000000)
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Inacceptable content type", http.StatusBadRequest)
		utils.Log.Println("client error: invalid content type", r.RemoteAddr)
		return
	}

	var (
		userId int64
		err    error
		// ok     bool
	)
	if userIdFloat, ok := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(float64); !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("err casting userId from accTk, err:", err)
		return
	} else {
		userId = int64(userIdFloat)
	}

	jDec := json.NewDecoder(r.Body)
	jDec.DisallowUnknownFields()

	var (
		userChangePw types.UserChangePwInput
		statusCode   int
	)
	statusCode, err = utils.JsonReqErrCheck(jDec.Decode(&userChangePw))
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		utils.Log.Println("cntrl/client error: ", err)
		return
	}

	if userChangePw.OldPassword == nil {
		http.Error(w, "Old password is required", http.StatusBadRequest)
		return
	} else if userChangePw.NewPassword == nil {
		http.Error(w, "New password field is empty", http.StatusBadRequest)
		utils.Log.Println("client error: Missing new password field", r.RemoteAddr)
		return
	}

	statusInt, err := grpcClients.ChangePwdMethod(&user_service_proto.ChangePwdReq{
		UserId:      userId,
		OldPassword: *userChangePw.OldPassword,
		NewPassword: *userChangePw.NewPassword,
	})
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("Failed to update user's pw, userId:", userId, "err:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ChangeDP(w http.ResponseWriter, r *http.Request) {
	var maxFileSize int = 2000000
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxFileSize))

	var (
		reader *multipart.Reader
		err    error
	)
	reader, err = r.MultipartReader()
	if err != nil {
		if err.Error() == "request Content-Type isn't multipart/form-data" {
			http.Error(w, "Invalid content type", http.StatusBadRequest)
			utils.Log.Println("client err: ", err)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("cntrl err: ", err)
		return
	}

	var profilePic *multipart.Part
	profilePic, err = reader.NextPart()
	if err != nil {
		if err != io.EOF {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			utils.Log.Println("cntrl err", err)
			return
		}
		http.Error(w, "Profile picture not present", http.StatusBadRequest)
		utils.Log.Println("client err: Missing profile picture")
		return

	} else if profilePic.FormName() != "profilePic" {
		http.Error(w, "Unrecognized form field", http.StatusBadRequest)
		utils.Log.Println("client err: Unrecognized form field")
		return
	}

	var (
		validFile bool
		buf       *bufio.Reader
		fileType  *mimetype.MIME
	)
	validFile, buf, fileType = utils.ValidFileType(profilePic, &utils.ProfImgValRegEx)
	if !validFile {
		http.Error(w, "Unacceptable file type", http.StatusUnprocessableEntity)
		utils.Log.Println("client error: 'Invalid prof-pic file type'", r.RemoteAddr)
		return
	}

	var (
		parsedFile   io.Reader
		fileRankLink string
		statusInt    int
	)
	// utils.S3FileUpload(buf, profilePic, "profile-images/", fileType, int64(maxFileSize))
	parsedFile, fileRankLink, statusInt = utils.ValidFile(buf, profilePic, fileType.Extension(), maxFileSize)
	switch statusInt {
	case 0:
		break
	case 413:
		http.Error(w, "Profile picture is too large, max size of 2MB", statusInt)
		utils.Log.Println("cntrl err: Uploading prof img to s3")
		return
	case 500:
		http.Error(w, "Internal server error", statusInt)
		utils.Log.Println("cntrl err: uploading changed profile picture to s3")
		return
	}

	var (
		userId int64
		// ok     bool
	)
	if userIdFloat, ok := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(float64); !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("Err typecasting userId:", userId, "err:", err)
		return
	} else {
		userId = int64(userIdFloat)
	}

	fileFullLink := "https://mumbleapi.daniel-dev.tech/mumbleapi/media/profile-images/" + fileRankLink
	statusInt, err = grpcClients.ChangeDpMethod(userId, fileFullLink)
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("grpc err change dp method err:", err, r.RemoteAddr)
		return
	}

	if err = s3Media.S3UploadImage(parsedFile, "profile-images/"+fileRankLink, fileType.String()); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("Err uploading new dp to s3, userId:", userId, "err:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.UserProfileLink{ProfileImgLink: fileFullLink})
}

func SearchUser(w http.ResponseWriter, r *http.Request) {
	var userEmail string
	if userEmail = r.URL.Query().Get("email"); userEmail == "" {
		http.Error(w, "Email field is empty", http.StatusBadRequest)
		utils.Log.Println("client error: email field is empty", r.RemoteAddr)
		return
	}

	var userId int64
	userIdFloat, ok := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(float64)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("err typecasting userId:", userId, r.RemoteAddr)
		return
	} else {
		userId = int64(userIdFloat)
	}

	jDec := json.NewDecoder(r.Body)
	jDec.DisallowUnknownFields()
	var SearchUserEmail types.SearchUserInput
	statusCode, err := utils.JsonReqErrCheck(jDec.Decode(&SearchUserEmail))
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		utils.Log.Println("err decoding json for searching err:", err)
		return
	}

	if SearchUserEmail.Email == nil {
		http.Error(w, "Enter a user's email", http.StatusBadRequest)
		utils.Log.Println("empty search email, userId", userId, r.RemoteAddr)
	}

	userResult, statusInt, err := grpcClients.SearchUserMethod(userId, *SearchUserEmail.Email)
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("err searching user in grpc, err:", err, r.RemoteAddr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResult)
}

func AddContact(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10000)

	var userId int64
	userIdFloat, ok := r.Context().Value("userDetails").(jwt.MapClaims)["UserId"].(float64)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("err typecasting acctk value")
		return
	} else {
		userId = int64(userIdFloat)
	}

	jDec := json.NewDecoder(r.Body)
	jDec.DisallowUnknownFields()

	var (
		contactId  types.AddContactId
		statusCode int
		err        error
	)
	statusCode, err = utils.JsonReqErrCheck(jDec.Decode(&contactId))

	if err != nil {
		http.Error(w, err.Error(), statusCode)
		utils.Log.Println("cntrl/client error: ", err)
		return
	} else if contactId.UserId == nil {
		http.Error(w, "Contact ID field is empty", http.StatusBadRequest)
		utils.Log.Println("client error: contactId field is empty", r.RemoteAddr)
		return
	} else if *contactId.UserId == userId {
		http.Error(w, "You cannot add yourself as a contact", http.StatusBadRequest)
		utils.Log.Println("client error: contact id is same as user id", userId, r.RemoteAddr)
		return
	}

	statusInt, err := grpcClients.AddContactMethod(*contactId.UserId, userId)
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("err adding contact, err:", err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func RefreshAccToken(w http.ResponseWriter, r *http.Request) {

	var (
		refreshToken *http.Cookie
		err          error
	)
	refreshToken, err = r.Cookie("refreshToken")
	if err != nil {
		http.Error(w, "Missing refresh token", http.StatusForbidden)
		utils.Log.Println("client error: refresh token missing", r.RemoteAddr)
		return
	}

	var (
		mapClaims jwt.MapClaims
		statusInt int
	)
	mapClaims, err, statusInt = utils.VerifyUserToken(refreshToken.Value)
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("client/server err:", err, r.RemoteAddr)
		return
	}

	var userId int64
	if userIdFloat, ok := mapClaims["UserId"].(float64); !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		utils.Log.Println("err typecasting refTk userId:", userId)
		return
	} else {
		userId = int64(userIdFloat)
	}

	accRefTk, statusInt, err := grpcClients.RefreshAccTokenMethod(userId, refreshToken.Value)
	if err != nil {
		http.Error(w, err.Error(), statusInt)
		utils.Log.Println("err grpc refreshing tokens, userId:", userId, "err:", err, r.RemoteAddr)
		return
	}
	// utils.Log.Println("response:", accRefTk, err)
	accTkCookie := &http.Cookie{Name: "accessToken", Value: accRefTk.AccessToken, MaxAge: int(accRefTk.AccessTokenExp), Path: "/api", HttpOnly: true}
	rekTkCookie := &http.Cookie{Name: "refreshToken", Value: accRefTk.RefreshToken, MaxAge: int(accRefTk.RefreshTokenExp), Path: "/api/auth", HttpOnly: true}

	http.SetCookie(w, accTkCookie)
	http.SetCookie(w, rekTkCookie)

	w.WriteHeader(http.StatusOK)
}
