package config

// cloudflare R2配置
type Cloudflare struct {
	BucketName      string `json:"bucketName" yaml:"bucketName"`
	AccountId       string `json:"accountId" yaml:"accountId"`
	AccessKeyId     string `json:"accessKeyId" yaml:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret" yaml:"accessKeySecret"`
}
