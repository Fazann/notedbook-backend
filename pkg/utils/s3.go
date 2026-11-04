package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	AccessKey     string
	SecretKey     string
	Region        string
	Bucket        string
	ProjectPrefix string
}

type S3UploadRes struct {
	URL  string
	Path string
}

// UploadFile uploads the file to S3 with the given Content-Type, so browsers treat it as that type.
func UploadFile(filePath string, fileName string, contentType string) (*S3UploadRes, error) {
	ctx := context.Background()
	cfg, err := s3Config.LoadDefaultConfig(ctx, s3Config.WithRegion(config.S3Region), s3Config.WithCredentialsProvider(
		aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     config.S3ID,
				SecretAccessKey: config.S3Secret,
			}, nil
		})),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	tm := transfermanager.New(client)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	halfPath := GenerateS3Key(fileName)
	key := fmt.Sprintf("%s/%s", config.S3ProjectName, halfPath)
	_, err = tm.UploadObject(ctx,
		&transfermanager.UploadObjectInput{
			Bucket:      &config.S3Bucket,
			Key:         &key,
			Body:        file,
			ContentType: &contentType,
		},
		func(o *transfermanager.Options) {
			o.PartSizeBytes = 10 * 1024 * 1024 // 10MB
			o.Concurrency = 2
		},
	)

	if err != nil {
		return nil, err
	}

	return &S3UploadRes{
		URL:  GetFullFileUrl(halfPath),
		Path: halfPath,
	}, nil
}

func GenerateFolder() string {
	now := time.Now()

	month := now.Month()
	year := now.Year()
	day := now.Day()

	return fmt.Sprintf("%d/%d/%d", year, month, day)
}

func GenerateS3Key(filename string) string {
	return fmt.Sprintf("upload/%s/%s", GenerateFolder(), filename)
}

func GetFullFileUrl(halfPath string) string {
	return fmt.Sprintf("%s/%s/%s", config.S3PrefixUrl, config.S3ProjectName, halfPath)
}
