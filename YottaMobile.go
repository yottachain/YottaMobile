package YottaMobile

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yottachain/YTCoreService/api"
	"github.com/yottachain/YTCoreService/env"
)

func Register(userName, privateKey string) string {
	os.Setenv("YTFS.snlist", "conf/snlist.properties")

	env.SetVersionID("2.0.1.5")
	api.StartMobileAPI()

	_, err := api.NewClientV2(&env.UserInfo{
		UserName: userName,
		Privkey:  []string{privateKey}}, 3)
	if err != nil {
		logrus.Panicf(":%s\n", err)

	}

	return "用户：" + userName + " 注册成功."
}

func ListObjects(bucketName, publicKey string) []string {

	content := publicKey[3:]

	c := api.GetClient(content)

	bucketAccessor := c.NewBucketAccessor()

	names, err := bucketAccessor.ListBucket()

	if err != nil {
		logrus.Errorf("[ListBucket ]AuthSuper ERR:%s\n", err)
	}

	return names
}
