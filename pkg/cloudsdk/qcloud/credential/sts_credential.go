package credential

import (
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

// CustomCredential use user defined SecretId and SecretKey
type STSCredential struct {
	lock                              sync.Mutex
	longRegion, clusterId, appId, uin string
	expiredDuration                   time.Duration

	stsClient *STS
}

func NewSTSCredential(longRegion, clusterId, appId, uin, stsSecretId, stsSecretKey, stsEndpoint string, expiredDuration time.Duration) QCloudCredential {
	sts := &STSCredential{
		longRegion:      longRegion,
		clusterId:       clusterId,
		appId:           appId,
		uin:             uin,
		expiredDuration: expiredDuration,
	}
	klog.Infof("region: %v, clusterId: %v, appId: %v, uin: %v", longRegion, clusterId, appId, uin)
	sts.stsClient = NewSts(longRegion, Config{Endpoint: stsEndpoint, SecretID: stsSecretId, SecretKey: stsSecretKey})
	return sts
}

func (s *STSCredential) GetQCloudCredential() *common.Credential {
	cred, err := s.stsClient.Get(s.uin)
	if err != nil {
		klog.Error(err)
		return &common.Credential{}
	}
	klog.V(6).Infof("cred.TmpSecretID: %v, cred.TmpSecretKey: %v, cred.Token: %v", cred.TmpSecretID, cred.TmpSecretKey, cred.Token)
	return &common.Credential{
		SecretId:  cred.TmpSecretID,
		SecretKey: cred.TmpSecretKey,
		Token:     cred.Token,
	}
}

func (s *STSCredential) UpdateQCloudCustomCredential(secretId, secretKey string) *common.Credential {
	return &common.Credential{}
}
