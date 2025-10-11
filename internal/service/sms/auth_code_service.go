package sms

import (
	"fmt"
	"strconv"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/puoxiu/gogochat/config"
	"github.com/puoxiu/gogochat/internal/service/redis"
	"github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

var smsClient *dysmsapi20170525.Client
var accessKeyID string
var accessKeySecret string
var signName string
var templateCode string

// init 使用AK&SK初始化账号Client
func init() {
	// 工程代码泄露可能会导致 AccessKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考。
	// 建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/378661.html。
	conf := config.GetConfig()
	accessKeyID = conf.AccessKeyID
	accessKeySecret = conf.AccessKeySecret
	signName = conf.SignName
	templateCode = conf.TemplateCode
	config := &openapi.Config{
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_ID。
		AccessKeyId: tea.String(accessKeyID),
		// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_SECRET。
		AccessKeySecret: tea.String(accessKeySecret),
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Dysmsapi
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	client, err := dysmsapi20170525.NewClient(config)
	if err != nil {
		zlog.Error(err.Error())
	}
	smsClient = client
}

// VerificationCode 发送验证码
func VerificationCode(telephone string) error {
	code, _ := redis.GetKey("auth_code")
	if code != "" {
		// 直接返回，验证码还没过期，用户应该去输验证码
		zlog.Info("please input auth code")
		return nil
	}
	// 验证码过期，重新生成
	code = strconv.Itoa(random.GetRandomInt(6))
	fmt.Println(code)
	err := redis.SetKeyEx("auth_code", code, time.Minute*10) // 10分钟有效
	if err != nil {
		return err
	}
	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode), // 短信模板
		PhoneNumbers:  tea.String(telephone),
		TemplateParam: tea.String("{\"code\":\"" + code + "\"}"),
	}

	runtime := &util.RuntimeOptions{}
	// 目前使用的是测试专用签名，签名必须是“阿里云短信测试”，模板code为“SMS_154950909”
	rsp, err := smsClient.SendSmsWithOptions(sendSmsRequest, runtime)
	if err != nil {
		return err
	}
	zlog.Info(*util.ToJSONString(rsp))
	return nil
}
