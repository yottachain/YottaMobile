package YottaMobile

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yottachain/YTCoreService/api"
	"github.com/yottachain/YTCoreService/env"
	"github.com/yottachain/YottaMobile/conf/aes"
)

//export Register
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

//export ListObjects
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

func Add(x, y int) int {
	return x + y
}

func UploadObject(url, filePath, bucketName, userName, privateKey string) {

	env.SetVersionID("2.0.1.5")
	var fileName string
	fileName = filepath.Base(filePath)
	os.Setenv("YTFS.snlist", "conf/snlist.properties")
	api.StartMobileAPI()
	c, err := api.NewClientV2(&env.UserInfo{
		UserName: userName,
		Privkey:  []string{privateKey}}, 3)
	if err != nil {
		logrus.Panicf(":%s\n", err)
	}
	do := c.UploadPreEncode(bucketName, fileName)

	err1 := do.UploadFile(filePath)

	if err1 != nil {
		logrus.Panicf("err1: %s", err1)
	}
	ss := do.OutPath()
	newUrl := url + "/api/v1/saveFileToLocal"
	postFile(ss, newUrl)
}

func postFile(filename string, targetUrl string) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return err
	}

	//打开文件句柄操作
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return err
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(resp.Status)
	fmt.Println(string(resp_body))
	return nil
}

type User struct {
	UserName   string
	Num        uint32
	PrivateKey string
	PublicKey  string
}

//export DownloadObject
func DownloadObject(url, filePath, fileName, bucketName string) string {

	userdata := ReadUserInfo()
	var user User
	user = UserUnmarshal(userdata)

	userName := user.UserName
	var blockNum int
	blockNum = 0

	key, err := aes.NewKey(user.PrivateKey, user.Num)
	if err != nil {

	}
	index := strings.Index(fileName, "/")
	if index != -1 {
		directory := filePath + "/" + bucketName + "/" + fileName[:index]
		fname := fileName[index+1:]
		filePath = CreateDirectory(directory, fname)
	} else {
		directory := filePath + "/" + bucketName
		filePath = CreateDirectory(directory, fileName)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("erra:%s\n", err)
	}
	defer f.Close()

	for blockNum != -1 {
		data, err := DownBlock(url, userName, bucketName, fileName, blockNum)
		if err != nil {
			blockNum = -1
			//g.JSON(http.StatusAccepted, gin.H{"Msg": "[" + fileName + "] download is failure ."})
			return "[" + fileName + "] download is failure ."
		} else {
			if len(data) > 0 {

				block := aes.NewEncryptedBlock(data)
				err1 := block.Decode(key, f)

				if err1 != nil {
					fmt.Println(err1)
					//g.JSON(http.StatusAccepted, gin.H{"Msg": "[" + fileName + "] download is failure ."})
					return "[" + fileName + "] download is failure ."
				} else {
					blockNum++
				}

			} else {
				blockNum = -1
			}
		}
	}
	md5Value := Md5SumFile(filePath)

	return "File md5:" + md5Value + " ,[" + fileName + " ] download is successful."
}

func DownBlock(url, userName, bucketName, fileName string, blockNum int) ([]byte, error) {

	str2 := fmt.Sprintf("%d", blockNum)
	newUrl := url + "/api/v1/getBlockForSGX?userName=" + userName + "&bucketName=" + bucketName + "&fileName=" + fileName + "&blockNum=" + str2

	var data []byte

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//http cookie接口
	cookieJar, _ := cookiejar.New(nil)
	c := &http.Client{
		Jar:       cookieJar,
		Transport: tr,
	}
	resp, err := c.Get(newUrl)

	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		} else {
			err = json.Unmarshal(body, &data)
			//data = body
		}

	}
	return data, nil
}

func ReadUserInfo() []byte {
	fp, err := os.OpenFile("./user.json", os.O_RDONLY, 0755)
	defer fp.Close()
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	data := make([]byte, 1024)
	n, err := fp.Read(data)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	//fmt.Println(string(data[:n]))
	return data[:n]
}

func CreateDirectory(directory, fileName string) string {
	s, err := os.Stat(directory)
	if err != nil {
		if !os.IsExist(err) {
			err = os.MkdirAll(directory, os.ModePerm)
			if err != nil {
				logrus.Errorf("err1:%s\n", err)
			}
		} else {
			logrus.Errorf("err2:%s\n", err)
		}
	} else {
		if !s.IsDir() {
			// logrus.Errorf("err:%s\n", "The specified path is not a directory.")
		}
	}
	if !strings.HasSuffix(directory, "/") {
		directory = directory + "/"
	}
	filePath := directory + fileName

	return filePath
}

func UserUnmarshal(data []byte) User {
	var user User
	if len(data) == 0 {
		fmt.Println("User JSON is null....")
	}
	err := json.Unmarshal(data, &user)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	//fmt.Println("UserName:" + user.UserName)
	//fmt.Println("PrivateKey:" + user.PrivateKey)
	//fmt.Println("PublicKey:" + user.PublicKey)
	return user
}

func Md5SumFile(file string) string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err.Error()
	}
	md5 := md5.New()
	md5.Write(data)
	md5Data := md5.Sum([]byte(nil))
	return hex.EncodeToString(md5Data)
}
