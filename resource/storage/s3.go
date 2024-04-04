package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

const (
	userKey = "users"
)

var (
	storages = map[string]Storage{}
)

type storageConfiguration struct {
	Region         string
	Endpoint       string
	Expires        time.Duration
	ExpiresForPush time.Duration
}

type Storage interface {
	ListAll() ([]string, error)
	Put(filename string, body []byte) error
	Delete(filename string) error
	DeleteByPrefix(prefix string) error
	PresignedUrl(filename string) (string, error)
	PresignedUrlForPush(filename string) (string, error)
}

type storageResource struct {
	config    *storageConfiguration
	client    *s3.Client
	bucket    string
	prefixKey string
}

func (s *storageResource) saveKey(filename string) string {
	return fmt.Sprintf("%s/%s", s.prefixKey, filename)
}

func (s *storageResource) ListAll() ([]string, error) {
	prefix := s.saveKey("")

	res, e := s.client.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket: &s.bucket,
		Prefix: &prefix,
	}).Send(context.Background())

	if e != nil {
		return nil, e
	}

	paths := []string{}

	for _, c := range res.Contents {
		paths = append(paths, *c.Key)
	}

	return paths, nil
}

// Put ファイルアップロード。
func (s *storageResource) Put(filename string, body []byte) error {
	reader := bytes.NewReader(body)

	key := s.saveKey(filename)

	upParams := &s3manager.UploadInput{
		Bucket: &s.bucket,
		Key:    &key,
		Body:   reader,
	}

	uploader := s3manager.NewUploaderWithClient(s.svc())

	_, err := uploader.Upload(upParams)
	if err != nil {
		return err
	}

	return nil
}

func (s *storageResource) Delete(filename string) error {
	svc := s.svc()

	key := s.saveKey(filename)

	deleteParams := &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	req := svc.DeleteObjectRequest(deleteParams)
	_, err := req.Send(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (s *storageResource) DeleteByPrefix(prefix string) error {
	svc := s.svc()

	prefix = s.saveKey(prefix)

	iterator := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: &s.bucket,
		Prefix: &prefix,
	})

	return s3manager.NewBatchDeleteWithClient(svc).Delete(context.Background(), iterator)
}

// PresignedUrlWithExpireSeconds Expire時間を指定してPresignedUrlを取得する。
func (s *storageResource) PresignedUrlWithExpireSeconds(filename string,
	seconds time.Duration) (string, error) {
	svc := s.svc()

	key := s.saveKey(filename)

	req := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})

	return req.Presign(seconds)
}

// PresignedUrl PresingedUrlを取得する。
func (s *storageResource) PresignedUrl(filename string) (string, error) {
	return s.PresignedUrlWithExpireSeconds(filename, s.config.Expires)
}

// PresignedUrlForPush PushのためのPresingedUrlを取得する。
func (s *storageResource) PresignedUrlForPush(filename string) (string, error) {
	return s.PresignedUrlWithExpireSeconds(filename, s.config.ExpiresForPush)
}

func (s *storageResource) svc() *s3.Client {
	if s.client == nil {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			return nil
		}
		cfg.Region = s.config.Region
		if len(s.config.Endpoint) > 0 {
			// https://github.com/aws/aws-sdk-go-v2/blob/master/example/aws/endpoints/customEndpoint/customEndpoint.go

			defaultResolver := endpoints.NewDefaultResolver()
			resolveFn := func(service, region string) (aws.Endpoint, error) {
				if service == "s3" {
					return aws.Endpoint{
						URL: s.config.Endpoint,
					}, nil
				}
				return defaultResolver.ResolveEndpoint(service, region)
			}
			cfg.EndpointResolver = aws.EndpointResolverFunc(resolveFn)
		}
		s.client = s3.New(cfg)
		if len(s.config.Endpoint) > 0 {
			s.client.ForcePathStyle = true
		}
	}
	return s.client
}

func newStorageResource(config *storageConfiguration, bucket, prefixKey string) *storageResource {
	return &storageResource{
		config:    config,
		bucket:    bucket,
		prefixKey: prefixKey,
	}
}

func UserStorage() Storage {
	return storages[userKey]
}

// SetupStorage 設定値をグローバルとして保存する。
func SetupStorage(region, endpoint, bucket string, expires int, expiresForPush int) {
	if _, ok := storages[userKey]; !ok {
		config := &storageConfiguration{
			Region:         region,
			Endpoint:       endpoint,
			Expires:        time.Duration(expires) * time.Minute,
			ExpiresForPush: time.Duration(expiresForPush) * time.Minute,
		}
		storages[userKey] = newStorageResource(config, bucket, userKey)
	}
}
