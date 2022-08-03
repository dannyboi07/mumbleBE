package utils

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"mumble-user-service/types"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
)

var Log log.Logger

var NameValRegEx regexp.Regexp = *regexp.MustCompile(`^[\p{L}\p{M} .'-]+$`)
var EmailValRegEx regexp.Regexp = *regexp.MustCompile("[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?.)+(?:[A-Z]{2}|com|org|net|gov|mil|biz|info|mobi|name|aero|jobs|museum)\b")
var pwAlphaRegEx regexp.Regexp = *regexp.MustCompile("[a-zA-Z]") // *regexp.MustCompile("^[ -~]{8,100}$")
var pwNumCharRegEx regexp.Regexp = *regexp.MustCompile("[0-9]|[ -@]|[[-\x60]|[{-~]")

var PrivKey *rsa.PrivateKey
var PubKey *rsa.PublicKey

func InitLogger() {
	Log = *log.New(os.Stdout, "Log: ", log.Lshortfile|log.LUTC)
}

func ValidName(name string) error {
	if name = strings.TrimSpace(name); len(name) < 3 {
		return errors.New("Name is too short")
	} else if !NameValRegEx.MatchString(name) {
		return errors.New("Name contains invalid characters")
	}

	return nil
}

func ValidEmail(email string) bool {
	return EmailValRegEx.MatchString(email)
}

func ValidPassword(pwd string) (string, bool) {
	if len(strings.Trim(pwd, " ")) < 8 || len(pwd) > 100 {
		return "Password must be between 8-100 characters, and contain a minimum of 3 numbers or special characters (@/#?...)", false
	}
	// 0th idx represents alphabet count
	// 1st idx represents num and ascii symbol count
	var countMap map[int]int = map[int]int{0: 0, 1: 0}

	for _, b := range pwd {
		// Is an alphabet?
		if pwAlphaRegEx.MatchString(string(b)) {
			countMap[0] += 1
			// Is a number or a ascii symbol?
		} else if pwNumCharRegEx.MatchString(string(b)) {
			countMap[1] += 1
		} else {
			return "Password contains invalid characters", false
		}
	}

	if countMap[0] > 4 && countMap[1] > 2 {
		return "", true
	}
	return "Password must be between 8-100 characters, and contain a minimum of 3 numbers or special characters (@/#?...)", false
}

func HashPassword(pwd string, cost int) (string, error) {
	hashedpw, err := bcrypt.GenerateFromPassword([]byte(pwd), cost)
	if err != nil {
		return "", err
	}
	return string(hashedpw), err
}

func AuthPassword(hashedPw, pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPw), []byte(pwd)) == nil
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

func CreateRefreshJwt(userDetails types.UserForToken) (string, time.Duration, error) {
	token := jwt.New(jwt.GetSigningMethod("RS256"))

	createdTime := time.Now()
	expireTime := createdTime.AddDate(0, 0, 7)

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
	return signedToken, expireTime.Sub(createdTime), nil
}

func VerifyJwt(token string) (jwt.MapClaims, codes.Code, error) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("Unexpected signing method: %v in jwt", t.Method.Alg())
		}
		return PubKey, nil
	})

	if parsedToken.Valid {
		if jwtMapClaims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			return jwtMapClaims, 0, nil
		}

		return nil, codes.PermissionDenied, errors.New("Invalid token claims")
	} else if ve, ok := err.(*jwt.ValidationError); ok {

		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			Log.Println("Malformed token")
			return nil, codes.InvalidArgument, errors.New("Malformed token")

		} else if ve.Errors&(jwt.ValidationErrorExpired) != 0 {
			return nil, codes.Unauthenticated, errors.New("Token expired")

		} else {
			Log.Println("Couldn't handle token validation error, err:", err)
			return nil, codes.Internal, errors.New("Internal server error")

		}
	} else {
		Log.Println("Couldn't handle token validation error, err:", err)
		return nil, codes.Internal, errors.New("Internal server error")
	}
}
