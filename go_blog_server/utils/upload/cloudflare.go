package upload

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"mime/multipart"
	"path/filepath"
	"server/global"
	"server/utils"
	"strings"
	"time"
)

type CloudFlare struct{}

type R2EndpointResolver struct {
	Endpoint string
	Region   string
}

func getCloudflareConfig() (bucketName, accountId, accessKeyId, accessKeySecret string) {
	cfg := global.Config.Cloudflare
	//fmt.Println(cfg)
	return cfg.BucketName, cfg.AccountId, cfg.AccessKeyId, cfg.AccessKeySecret
}

func (r R2EndpointResolver) ResolveEndpoint(service, region string) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL:               r.Endpoint,
		SigningRegion:     r.Region,
		HostnameImmutable: true,
	}, nil
}

func InitS3Client(acId, aKId, aKSecret string) *s3.Client {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", acId)
	region := "auto"

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(aKId, aKSecret, "")),
		config.WithEndpointResolver(R2EndpointResolver{
			Endpoint: endpoint,
			Region:   region,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Cloudflare R2 必须
	})
}

func (*CloudFlare) UploadImage(file *multipart.FileHeader) (string, string, error) {
	size := float64(file.Size) / float64(1024*1024)
	if size >= float64(global.Config.Upload.Size) {
		return "", "", fmt.Errorf("the image size exceeds the set size, the current size is: %.2f MB, the set size is: %d MB", size, global.Config.Upload.Size)

	}

	ext := filepath.Ext(file.Filename)
	name := strings.TrimSuffix(file.Filename, ext)
	if _, exists := WhiteImageList[ext]; !exists {
		return "", "", errors.New("don't upload files that aren't image types")
	}

	filename := utils.MD5V([]byte(name)) + "-" + time.Now().Format("20060102150405") + ext

	data, err := file.Open()
	if err != nil {
		return "", "", err
	}
	defer data.Close()

	bucketName, accountId, accessKeyId, accessKeySecret := getCloudflareConfig()
	client := InitS3Client(accountId, accessKeyId, accessKeySecret)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    aws.String(filename), // 对象键（文件名）
		Body:   data,                 // 文件内容
	})
	if err != nil {
		log.Fatal(err)
		return "", "", nil
	}
	//https://pub-935c9ac1411f458aa99886322598440f.r2.dev/1.png
	url := fmt.Sprintf("https://pub-935c9ac1411f458aa99886322598440f.r2.dev/%s", filename)
	fmt.Printf("-------------Successfully wrote-------------- %s\n", name)
	//return true, url

	return url, filename, nil
}

func (*CloudFlare) DeleteImage(key string) error {
	bucketName, accountId, accessKeyId, accessKeySecret := getCloudflareConfig()
	client := InitS3Client(accountId, accessKeyId, accessKeySecret)
	// 删除对象
	_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatalf("failed to delete object: %v", err)
	}

	fmt.Println("对象删除成功")
	return err

}
