package utils

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"math/rand"
	"mime/multipart"
	"mumble-gateway-service/s3Media"
	"mumble-gateway-service/types"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"

	"github.com/gabriel-vasile/mimetype"

	"github.com/golang-jwt/jwt/v4"
)

var Log stdlog.Logger
var Hostname string
var ProfImgValRegEx regexp.Regexp = *regexp.MustCompile("^image/jpg|jpeg|png|heif|heic|gif$")
var NameValRegEx regexp.Regexp = *regexp.MustCompile(`^[\p{L}\p{M} .'-]+$`)
var EmailValRegEx regexp.Regexp = *regexp.MustCompile("[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?.)+(?:[A-Z]{2}|com|org|net|gov|mil|biz|info|mobi|name|aero|jobs|museum)\b")
var PrivKey *rsa.PrivateKey
var PubKey *rsa.PublicKey

func InitLogger() {
	Log = *stdlog.New(os.Stdout, "Log: ", stdlog.Lshortfile|stdlog.LUTC)
}

func ValidName(name *string) bool {
	if *name = strings.TrimSpace(*name); len(*name) < 3 {
		return false
	} else if !NameValRegEx.MatchString(*name) {
		return false
	}
	return true
}

func DecodeJsonBody(w http.ResponseWriter, r *http.Request, dataType interface{}) (error, int) {
	if r.Header.Get("Content-Type") != "application/json" {
		return errors.New("Only JSON requests accepted"), http.StatusBadRequest
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
			return errors.New(fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)), http.StatusBadRequest

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New(fmt.Sprintf("Request body contains badly formed JSON")), http.StatusBadRequest

		case errors.As(err, &unMarshallTypeError):
			return errors.New(fmt.Sprintf("Request body contains an invalid value for field: %q (at position: %d)", unMarshallTypeError.Field, unMarshallTypeError.Offset)), http.StatusBadRequest

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return errors.New(fmt.Sprintf("Request body contains unknown field %s", fieldName)), http.StatusBadRequest

		case errors.Is(err, io.EOF):
			return errors.New("Request body cannot be empty"), http.StatusBadRequest

		case err.Error() == "http: request body too large":
			return errors.New("Request body cannot be larger than 1MB"), http.StatusRequestEntityTooLarge

		default:
			return err, http.StatusBadRequest
		}
	}
	err = jDec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("Request body must only contain single JSON object"), http.StatusBadRequest
	}
	return nil, 0
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

func MapGrpcErrors(grpcCode codes.Code) int {
	switch grpcCode {
	case codes.PermissionDenied: // PERMISSION DENIED
		return 403

	case codes.Unauthenticated: // UNAUTHENTICATED
		return 401

	case codes.InvalidArgument: // INVALID_ARGUMENT
		return 400

	case codes.NotFound: // NOT_FOUND
		return 404

	case codes.DeadlineExceeded: // DEADLINE_EXCEEDED
		return 408

	case codes.AlreadyExists: // ALREADY_EXISTS
		return 409 // HTTP CODE CONFLICT

	default:
		return 500
	}
}

func ValidFileType(file *multipart.Part, fileExtType *regexp.Regexp) (bool, *bufio.Reader, *mimetype.MIME) {
	buf := bufio.NewReader(file)
	sniff, _ := buf.Peek(512)

	fileContentType := mimetype.Detect(sniff)
	if fileExtType.MatchString(fileContentType.String()) {
		return true, buf, fileContentType
	} else {
		return false, nil, nil
	}
}

func ReadPartToString(multiPart *multipart.Part) string {

	buf := bytes.NewBuffer(nil)
	buf.ReadFrom(multiPart)

	return buf.String()
}

const alphaForRand = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandFileName(ext string) string {
	timeNow := strconv.FormatInt(time.Now().Unix(), 10)

	b := make([]byte, 50)
	for i := range b {
		b[i] = alphaForRand[rand.Int63()%int64(len(alphaForRand))]
	}

	return timeNow + "-" + string(b) + ext
}

func FileUpload(file *multipart.Part, buf *bufio.Reader, dir string, ext *mimetype.MIME, maxFileSize int64) (int, error, string) {

	timeNow := time.Now().Format("2006-01-02-15-04-05")
	tempFile, err := ioutil.TempFile(dir, timeNow+"-*"+ext.Extension())
	if err != nil {
		return http.StatusInternalServerError, err, ""
	}
	defer tempFile.Close()

	lmt := io.MultiReader(buf, io.LimitReader(file, maxFileSize-511))
	written, err := io.Copy(tempFile, lmt)
	if err != nil && err != io.EOF {
		os.Remove(tempFile.Name())
		if err.Error() == "http: request body too large" {
			return http.StatusRequestEntityTooLarge, err, ""
		}
		return http.StatusInternalServerError, err, ""
	} else if written > maxFileSize {
		os.Remove(tempFile.Name())
		return http.StatusRequestEntityTooLarge, err, ""
	}

	return 0, nil, "http://localhost:8080/" + tempFile.Name()
}

func ValidFile(buf *bufio.Reader, file *multipart.Part, ext string, maxFileSize int) (io.Reader, string, int) {

	var fileToUpload io.Reader = io.MultiReader(buf, file)

	var (
		fileBuffer      []byte = []byte{}
		buff                   = make([]byte, 4096)
		totalSize, size int
		err             error
	)
	for {
		size, err = fileToUpload.Read(buff)
		totalSize += size
		fileBuffer = append(fileBuffer, buff[:size]...)

		if totalSize > maxFileSize {
			return nil, "", http.StatusRequestEntityTooLarge
		} else if err == io.EOF && totalSize <= maxFileSize {
			break
		} else if err != nil {
			return nil, "", http.StatusInternalServerError
		}
	}

	return bytes.NewReader(fileBuffer), RandFileName(ext), 0
}

func S3FileUpload(buf *bufio.Reader, file *multipart.Part, fileName, fileMime string, maxFileSize int64) int {
	// var fileName string = RandFileName(fileMime.Extension())
	var fileToUpload io.Reader = io.MultiReader(buf, file)

	var (
		fileBuffer      []byte = []byte{}
		buff                   = make([]byte, 4096)
		totalSize, size int
		err             error
	)
	for {
		size, err = fileToUpload.Read(buff)
		totalSize += size
		fileBuffer = append(fileBuffer, buff[:size]...)

		if totalSize > int(maxFileSize) {
			return http.StatusRequestEntityTooLarge

		} else if err == io.EOF && totalSize <= int(maxFileSize) {
			break

		} else if err != nil {
			return http.StatusInternalServerError
		}
	}
	err = s3Media.S3UploadImage(bytes.NewReader(fileBuffer), fileName, fileMime)
	if err != nil {
		Log.Println("err uploading to s3", err)
		return http.StatusInternalServerError
	}
	return 0
}

func HashPassword(password string, cost int) (string, error) {
	hashedPw, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedPw), nil
}

func AuthPassword(hashedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

type CustomClaims struct {
	*jwt.RegisteredClaims
	types.UserForToken
}

func CreateJwt(userDetails types.UserForToken) (string, int, error) {
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	createdTime := time.Now()
	expireTime := createdTime.Add(time.Minute * 15)

	token.Claims = &CustomClaims{
		&jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
		userDetails,
	}

	signedToken, err := token.SignedString(PrivKey)
	if err != nil {
		return "", 0, err
	}
	return signedToken, int(expireTime.Sub(createdTime).Seconds()), nil
}

func VerifyUserToken(token string) (jwt.MapClaims, error, int) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Method.Alg())
		}

		return PubKey, nil
	})

	if parsedToken.Valid {
		return parsedToken.Claims.(jwt.MapClaims), nil, http.StatusOK

	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			fmt.Println("Malformed token")
			return nil, errors.New("Malformed Token"), http.StatusBadRequest

		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			return nil, errors.New("Token expired"), http.StatusUnauthorized

		} else {
			fmt.Println("Couldn't handle this token:", err)
			return nil, err, http.StatusInternalServerError

		}
	} else {
		fmt.Println("Couldn't handle this token:", err)
		return nil, err, http.StatusInternalServerError
	}
}

func CreateRefreshJwt(userDetails types.UserForToken) (string, time.Duration, error) {
	refreshToken := jwt.New(jwt.GetSigningMethod("RS256"))
	createdTime := time.Now()
	expireTime := createdTime.AddDate(0, 0, 7)

	refreshToken.Claims = &CustomClaims{
		&jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
		},
		userDetails,
	}
	signedToken, err := refreshToken.SignedString(PrivKey)
	if err != nil {
		return "", 0, err
	}

	return signedToken, expireTime.Sub(createdTime), nil
}
