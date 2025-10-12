package sms

import (
	"fmt"
	// "strconv"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	// util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/puoxiu/gogochat/config"
	"github.com/puoxiu/gogochat/internal/service/redis"
	"github.com/puoxiu/gogochat/pkg/constants"
	// "github.com/puoxiu/gogochat/pkg/random"
	"github.com/puoxiu/gogochat/pkg/zlog"
)

var smsClient *dysmsapi20170525.Client

// createClient 使用AK&SK初始化账号Client
func createClient() (result *dysmsapi20170525.Client, err error) {
	// 工程代码泄露可能会导致 AccessKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考。
	// 建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/378661.html。
	accessKeyID := config.GetConfig().AccessKeyID
	accessKeySecret := config.GetConfig().AccessKeySecret
	if smsClient == nil {
		config := &openapi.Config{
			// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_ID。
			AccessKeyId: tea.String(accessKeyID),
			// 必填，请确保代码运行环境设置了环境变量 ALIBABA_CLOUD_ACCESS_KEY_SECRET。
			AccessKeySecret: tea.String(accessKeySecret),
		}
		// Endpoint 请参考 https://api.aliyun.com/product/Dysmsapi
		config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
		smsClient, err = dysmsapi20170525.NewClient(config)
	}
	return smsClient, err
}

// func VerificationCode(telephone string) (string, int) {
// 	client, err := createClient()
// 	if err != nil {
// 		zlog.Error(err.Error())
// 		return constants.SYSTEM_ERROR, -1
// 	}
// 	key := "auth_code_" + telephone
// 	code, err := redis.GetKey(key)
// 	if err != nil {
// 		zlog.Error(err.Error())
// 		return constants.SYSTEM_ERROR, -1
// 	}

// 	if code != "" {
// 		// 直接返回，验证码还没过期，用户应该去输验证码
// 		message := "目前还不能发送验证码，请输入已发送的验证码"
// 		zlog.Info(message)
// 		return message, -2
// 	}
// 	// 验证码过期，重新生成
// 	code = strconv.Itoa(random.GetRandomInt(6))
// 	fmt.Println(code)
// 	err = redis.SetKeyEx(key, code, time.Minute) // 1分钟有效
// 	if err != nil {
// 		zlog.Error(err.Error())
// 		return constants.SYSTEM_ERROR, -1
// 	}
// 	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
// 		SignName:      tea.String("阿里云短信测试"),
// 		TemplateCode:  tea.String("SMS_154950909"), // 短信模板
// 		PhoneNumbers:  tea.String(telephone),
// 		TemplateParam: tea.String("{\"code\":\"" + code + "\"}"),
// 	}

// 	runtime := &util.RuntimeOptions{}
// 	// 目前使用的是测试专用签名，签名必须是“阿里云短信测试”，模板code为“SMS_154950909”
// 	rsp, err := client.SendSmsWithOptions(sendSmsRequest, runtime)
// 	if err != nil {
// 		zlog.Error(err.Error())
// 		return constants.SYSTEM_ERROR, -1
// 	}
// 	zlog.Info(*util.ToJSONString(rsp))
// 	return "验证码发送成功，请及时在对应电话查收短信", 0
// }



// VerificationCode 生成固定验证码（123456），替代真实短信发送
func VerificationCode(telephone string) (string, int) {
	// 5. 注释/删除阿里云客户端初始化（不再使用）
	// client, err := createClient()
	// if err != nil {
	// 	zlog.Error(err.Error())
	// 	return constants.SYSTEM_ERROR, -1
	// }

	// 保留 Redis 逻辑：防重复发送、验证码有效期（1分钟）
	key := "auth_code_" + telephone
	code, err := redis.GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 若验证码未过期，提示用户输入已发送的验证码
	if code != "" {
		message := "目前还不能发送验证码，请输入已发送的验证码（固定验证码：123456）"
		zlog.Info(message)
		return message, -2
	}

	// 6. 核心修改：使用固定验证码（123456），替代随机生成
	code = "123456" // 固定验证码，测试用
	fmt.Printf("当前手机号 %s 的验证码：%s（有效期1分钟）\n", telephone, code)

	// 保留 Redis 缓存：将固定验证码存入 Redis，设置1分钟过期
	err = redis.SetKeyEx(key, code, time.Minute)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 7. 修改返回信息，明确告知是测试用固定验证码
	return "测试环境：验证码已发送（固定验证码：123456，有效期1分钟）", 0
}
