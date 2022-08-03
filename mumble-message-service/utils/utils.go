package utils

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var Log stdlog.Logger

var ProfImgValRegEx regexp.Regexp = *regexp.MustCompile("^image/jpg|jpeg|png|heif|heic|gif$")
var NameValRegEx regexp.Regexp = *regexp.MustCompile(`^[\p{L}\p{M} .'-]+$`)
var EmailValRegEx regexp.Regexp = *regexp.MustCompile("[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?.)+(?:[A-Z]{2}|com|org|net|gov|mil|biz|info|mobi|name|aero|jobs|museum)\b")
var PrivKey *rsa.PrivateKey
var PubKey *rsa.PublicKey

func InitLogger() {
	Log = *stdlog.New(os.Stdout, "Log: ", stdlog.Lshortfile|stdlog.LUTC)
}

// func ValidName(name *string) bool {
// 	if *name = strings.TrimSpace(*name); len(*name) < 3 {
// 		return false
// 	} else if !NameValRegEx.MatchString(*name) {
// 		return false
// 	}
// 	return true
// }

func DecodeJsonBody(w http.ResponseWriter, r *http.Request, dataType interface{}) (int, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return http.StatusBadRequest, errors.New("Only JSON requests accepted")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1000000)

	jDec := json.NewDecoder(r.Body)
	jDec.DisallowUnknownFields()

	err := jDec.Decode(&dataType)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unMarshallTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			return http.StatusBadRequest, fmt.Errorf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return http.StatusBadRequest, fmt.Errorf("Request body contains badly formed JSON")

		case errors.As(err, &unMarshallTypeError):
			return http.StatusBadRequest, fmt.Errorf("Request body contains an invalid value for field: %q (at position: %d)", unMarshallTypeError.Field, unMarshallTypeError.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			var fieldName string = strings.TrimPrefix(err.Error(), "json: unknown field ")
			return http.StatusBadRequest, fmt.Errorf("Request body contains unknown field %s", fieldName)

		case errors.Is(err, io.EOF):
			return http.StatusBadRequest, errors.New("Request body cannot be empty")

		case err.Error() == "http: request body too large":
			return http.StatusRequestEntityTooLarge, errors.New("Request body cannot be larger than 1MB")

		default:
			return http.StatusBadRequest, err
		}
	}
	err = jDec.Decode(&struct{}{})
	if err != io.EOF {
		return http.StatusBadRequest, errors.New("Request body must only contain single JSON object")
	}
	return 0, nil
}

func JsonReqErrCheck(err error) (int, error) {
	if err != nil {
		var syntaxError *json.SyntaxError
		var unMarshallTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			return http.StatusBadRequest, fmt.Errorf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return http.StatusBadRequest, fmt.Errorf("Request body contains badly formed JSON")

		case errors.As(err, &unMarshallTypeError):
			return http.StatusBadRequest, fmt.Errorf("Request body contains an invalid value for field: %q (at position: %d)", unMarshallTypeError.Field, unMarshallTypeError.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return http.StatusBadRequest, fmt.Errorf("Request body contains unknown field %s", fieldName)

		case errors.Is(err, io.EOF):
			return http.StatusBadRequest, errors.New("Request body cannot be empty")

		case err.Error() == "http: request body too large":
			return http.StatusRequestEntityTooLarge, errors.New("Request body cannot be larger than 1MB")

		default:
			return http.StatusBadRequest, err
		}
	}
	return 0, nil
}

// func ValidFileType(file *multipart.Part, fileExtType *regexp.Regexp) (bool, *bufio.Reader, *mimetype.MIME) {
// 	buf := bufio.NewReader(file)
// 	sniff, _ := buf.Peek(512)

// 	fileContentType := mimetype.Detect(sniff)
// 	if fileExtType.MatchString(fileContentType.String()) {
// 		return true, buf, fileContentType
// 	} else {
// 		return false, nil, nil
// 	}
// }

// func ReadPartToString(multiPart *multipart.Part) string {
// 	// fieldPart, err := multiPart
// 	buf := bytes.NewBuffer(nil)
// 	buf.ReadFrom(multiPart)
// 	// fmt.Println(buf, buf.String(), multiPart)
// 	return buf.String()
// 	// _, err := io.Copy(buf, multiPart)
// 	// if err != nil {
// 	// 	fmt.Print(buf.String(), buf)
// 	// 	return "", err
// 	// }
// 	// return buf.String(), nil
// }

// const alphaForRand = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// func RandFileName(ext string) string {
// 	timeNow := strconv.FormatInt(time.Now().Unix(), 10)

// 	b := make([]byte, 50)
// 	for i := range b {
// 		// Log.Println("rand", rand.Int63(), "index pre", int64(len(alphaForRand)), "index val", rand.Int63()%int64(len(alphaForRand)))
// 		b[i] = alphaForRand[rand.Int63()%int64(len(alphaForRand))]
// 	}

// 	return timeNow + "-" + string(b) + ext
// }

// func FileUpload(file *multipart.Part, buf *bufio.Reader, dir string, ext *mimetype.MIME, maxFileSize int64) (int, error, string) {

// 	timeNow := time.Now().Format("2006-01-02-15-04-05")
// 	tempFile, err := ioutil.TempFile(dir, timeNow+"-*"+ext.Extension())
// 	if err != nil {
// 		return http.StatusInternalServerError, err, ""
// 	}
// 	defer tempFile.Close()

// 	lmt := io.MultiReader(buf, io.LimitReader(file, maxFileSize-511))
// 	written, err := io.Copy(tempFile, lmt)
// 	if err != nil && err != io.EOF {
// 		os.Remove(tempFile.Name())
// 		if err.Error() == "http: request body too large" {
// 			return http.StatusRequestEntityTooLarge, err, ""
// 		}
// 		return http.StatusInternalServerError, err, ""
// 	} else if written > maxFileSize {
// 		os.Remove(tempFile.Name())
// 		return http.StatusRequestEntityTooLarge, err, ""
// 	}

// 	return 0, nil, "http://localhost:8080/" + tempFile.Name()
// }

// func S3FileUpload(buf *bufio.Reader, file *multipart.Part, s3Dir string, fileMime *mimetype.MIME, maxFileSize int64) (string, int) {
// 	var fileName string = RandFileName(fileMime.Extension())
// 	var fileToUpload io.Reader = io.MultiReader(buf, file)

// 	var (
// 		fileBuffer      []byte = []byte{}
// 		buff                   = make([]byte, 4096)
// 		totalSize, size int
// 		err             error
// 	)
// 	for {
// 		size, err = fileToUpload.Read(buff)
// 		totalSize += size
// 		fileBuffer = append(fileBuffer, buff[:size]...)

// 		if totalSize > int(maxFileSize) {
// 			return "", http.StatusRequestEntityTooLarge

// 		} else if err == io.EOF && totalSize <= int(maxFileSize) {
// 			break

// 		} else if err != nil {
// 			return "", http.StatusInternalServerError
// 		}
// 	}
// 	err = s3Media.S3UploadImage(bytes.NewReader(fileBuffer), s3Dir+fileName, fileMime.String())
// 	if err != nil {
// 		Log.Println("err uploading to s3", err)
// 		return "", http.StatusInternalServerError
// 	}
// 	return "http://localhost:8080/gets3/" + s3Dir + fileName, 0
// }

// func HashPassword(password string, cost int) (string, error) {
// 	hashedPw, err := bcrypt.GenerateFromPassword([]byte(password), cost)
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(hashedPw), nil
// }

// func AuthPassword(hashedPassword string, password string) bool {
// 	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
// 	return err == nil
// }

// type CustomClaims struct {
// 	*jwt.RegisteredClaims
// 	types.UserForToken
// }

// func CreateJwt(userDetails types.UserForToken) (string, int, error) {
// 	token := jwt.New(jwt.GetSigningMethod("RS256"))
// 	createdTime := time.Now()
// 	expireTime := createdTime.Add(time.Minute * 15)
// 	// var expireTime int64 = 3600
// 	token.Claims = &CustomClaims{
// 		// &jwt.StandardClaims{
// 		// 	ExpiresAt: expireTime,
// 		// },
// 		&jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(expireTime),
// 		},
// 		userDetails,
// 	}
// 	signedToken, err := token.SignedString(PrivKey)
// 	if err != nil {
// 		return "", 0, err
// 	} //int(expireTime.Sub(createdTime).Seconds())
// 	// Log.Println(int(expireTime.Sub(createdTime)), expireTime.Sub(createdTime))
// 	return signedToken, int(expireTime.Sub(createdTime).Seconds()), nil
// }

// func VerifyUserToken(token string) (jwt.MapClaims, error, int) {
// 	// pubKey, err := jwt.ParseRSAPublicKeyFromPEM()
// 	// if err != nil {
// 	// 	fmt.Println("pubkey err", err)
// 	// 	return false, err
// 	// }
// 	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
// 		// if _, ok := parsedToken.Method.(*jwt.SigningMethodES256); !ok {
// 		// 	return nil, fmt.Errorf("Unexpected signing method: %v", parsedToken.Header["alg"])
// 		// }

// 		return PubKey, nil
// 	})

// 	if parsedToken.Valid {
// 		return parsedToken.Claims.(jwt.MapClaims), nil, http.StatusOK
// 	} else if ve, ok := err.(*jwt.ValidationError); ok {
// 		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
// 			fmt.Println("Malformed token")
// 			return nil, errors.New("Malformed Token"), http.StatusBadRequest
// 		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
// 			// Token is either expired or not active yet
// 			//fmt.Println("Token expired")
// 			return nil, errors.New("Token expired"), http.StatusUnauthorized
// 		} else {
// 			fmt.Println("Couldn't handle this token:", err)
// 			return nil, err, http.StatusInternalServerError
// 		}
// 	} else {
// 		fmt.Println("Couldn't handle this token:", err)
// 		return nil, err, http.StatusInternalServerError
// 	}
// }

// func CreateRefreshJwt(userDetails types.UserForToken) (string, time.Duration, error) {
// 	refreshToken := jwt.New(jwt.GetSigningMethod("RS256"))
// 	createdTime := time.Now()
// 	expireTime := createdTime.AddDate(0, 0, 7)

// 	refreshToken.Claims = &CustomClaims{
// 		&jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(expireTime),
// 		},
// 		userDetails,
// 	}
// 	signedToken, err := refreshToken.SignedString(PrivKey)
// 	if err != nil {
// 		return "", 0, err
// 	}

// 	return signedToken, expireTime.Sub(createdTime), nil
// }
