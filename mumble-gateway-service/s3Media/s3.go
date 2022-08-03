package s3Media

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// var profImgUpParams = &s3manager.UploadInput{
// 	Bucket: aws.String(os.Getenv("S3_PROF_IMG_BUCK")),
// }

var sess *session.Session
var getObjSess *s3.S3

// var sess = session.Must(session.NewSession(&aws.Config{
// 	Region:      aws.String(os.Getenv("AWS_REGION")),
// 	Credentials: credentials.NewStaticCredentials(*aws.String(os.Getenv("AWS_ACCESS_KEY_ID")), *aws.String(os.Getenv("AWS_SECRET_KEY")), ""),
// }))

var uploader *s3manager.Uploader
var bucket string

func InitS3() {
	bucket = *aws.String(os.Getenv("S3_BUCKET"))
	sess = session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(os.Getenv("AWS_REGION")),
		Credentials: credentials.NewStaticCredentials(*aws.String(os.Getenv("AWS_ACCESS_KEY_ID")), *aws.String(os.Getenv("AWS_SECRET_KEY")), ""),
	}))
	getObjSess = s3.New(sess)
	uploader = s3manager.NewUploader(sess)
}

func S3UploadImage(fileBody io.Reader, key, contentType string) error {

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        fileBody,
		ContentType: &contentType,
	})
	return err
}

func GetS3Img(key string) (*s3.GetObjectOutput, string, int) {
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	result, err := getObjSess.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
				return nil, "Image not found", http.StatusNotFound
			case s3.ErrCodeInvalidObjectState:
				fmt.Println(s3.ErrCodeInvalidObjectState, aerr.Error())
				return nil, "Interval server error", http.StatusInternalServerError
			default:
				fmt.Println(aerr.Error())
				return nil, "Interval server error", http.StatusInternalServerError
			}
		}
	}
	return result, "", 0
}
