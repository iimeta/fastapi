package chat

import (
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/crypto"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"google.golang.org/api/option"
	"slices"
	"time"
)

var gcpCache = cache.New() // [key]Token

func getGcpToken(ctx context.Context, key *model.Key, proxyURL string) string {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "getGcpToken time: %d", gtime.TimestampMilli()-now)
	}()

	if gcpTokenCacheValue := gcpCache.GetVal(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key)); gcpTokenCacheValue != nil {
		return gcpTokenCacheValue.(string)
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key)); err != nil {
			logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
		} else {
			if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
			}
		}

		return reply
	}

	result := gstr.Split(key.Key, "|")

	data := g.Map{
		"client_id":     result[1],
		"client_secret": result[2],
		"refresh_token": result[3],
		"grant_type":    "refresh_token",
	}

	getGcpTokenRes := new(model.GetGcpTokenRes)
	if err = util.HttpPost(ctx, config.Cfg.Gcp.GetTokenUrl, nil, data, &getGcpTokenRes, proxyURL); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
		return ""
	}

	if getGcpTokenRes.Error != "" {
		logger.Errorf(ctx, "getGcpToken key: %s, getGcpTokenRes.Error: %s", key.Key, getGcpTokenRes.Error)
		if err = grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			service.Key().DisabledModelKey(ctx, key)
		}); err != nil {
			logger.Error(ctx, err)
		}
		return ""
	}

	if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key), getGcpTokenRes.AccessToken, time.Second*time.Duration(getGcpTokenRes.ExpiresIn-60)); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key.Key), getGcpTokenRes.AccessToken, getGcpTokenRes.ExpiresIn-60); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
	}

	return getGcpTokenRes.AccessToken
}

type ApplicationDefaultCredentials struct {
	Type                    string `json:"type"`
	ProjectId               string `json:"project_id"`
	PrivateKeyId            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientId                string `json:"client_id"`
	AuthUri                 string `json:"auth_uri"`
	TokenUri                string `json:"token_uri"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

func getGcpTokenNew(ctx context.Context, key *model.Key, proxyURL string) (string, string, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "getGcpTokenNew time: %d", gtime.TimestampMilli()-now)
	}()

	adc := &ApplicationDefaultCredentials{}
	if err := gjson.Unmarshal([]byte(key.Key), adc); err != nil {
		logger.Errorf(ctx, "getGcpTokenNew gjson.Unmarshal key: %s, error: %v", key.Key, err)
		return "", "", err
	}

	if gcpTokenCacheValue := gcpCache.GetVal(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key))); gcpTokenCacheValue != nil {
		return adc.ProjectId, gcpTokenCacheValue.(string), nil
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key))); err != nil {
			logger.Errorf(ctx, "getGcpTokenNew key: %s, error: %v", key.Key, err)
		} else {
			if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getGcpTokenNew key: %s, error: %v", key.Key, err)
			}
		}

		return adc.ProjectId, reply, nil
	}

	client, err := credentials.NewIamCredentialsClient(ctx, option.WithCredentialsJSON([]byte(key.Key)))
	if err != nil {
		logger.Errorf(ctx, "getGcpTokenNew NewIamCredentialsClient key: %s, error: %v", key.Key, err)
		return "", "", err
	}

	defer func() {
		if err = client.Close(); err != nil {
			logger.Error(ctx, err)
		}
	}()

	request := &credentialspb.GenerateAccessTokenRequest{
		Name:  fmt.Sprintf("projects/-/serviceAccounts/%s", adc.ClientEmail),
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform"},
	}

	response, err := client.GenerateAccessToken(ctx, request)
	if err != nil {
		logger.Errorf(ctx, "getGcpTokenNew GenerateAccessToken key: %s, error: %v", key.Key, err)
		if slices.Contains(config.Cfg.Error.AutoDisabled, err.Error()) {
			if err = grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				service.Key().DisabledModelKey(ctx, key)
			}); err != nil {
				logger.Error(ctx, err)
			}
		}
		return "", "", err
	}

	if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), response.AccessToken, time.Minute*50); err != nil {
		logger.Errorf(ctx, "getGcpTokenNew key: %s, error: %v", key.Key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), response.AccessToken, 60*50); err != nil {
		logger.Errorf(ctx, "getGcpTokenNew key: %s, error: %v", key.Key, err)
	}

	return adc.ProjectId, response.AccessToken, nil
}
