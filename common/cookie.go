package common

import (
	"bytes"
	"encoding/json"
	"github.com/robfig/cron"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Expiry   string `json:"expiry"`
	Token    string `json:"token"`
}

var (
	users                     []*User
	USER_INFO_ENV_NAME_PREFIX = "Go_Proxy_BingAI_USER_INFO"
	CRON_STR                  = "0 0 0 * * ?"
)

func init() {
	initUserInfo()
	cronAndUpdateToken()
}

func cronAndUpdateToken() {
	if os.Getenv("Go_Proxy_BingAI_CRON_STR") != "" {
		CRON_STR = os.Getenv("Go_Proxy_BingAI_CRON_STR")
	}

	c := cron.New()
	// 每天凌晨0点执行定时任务
	err := c.AddFunc(CRON_STR, func() {
		log.Println("[定时任务]开始-检查是否有token过期")
		for _, user := range users {
			if IsDateEqualToday(user.Expiry) {
				var oldToken = user.Token
				log.Printf("[定时任务] 用户：%s ，准备过期，过期时间：%s ,开始执行更新。。。", user.Username, user.Expiry)
				updateUserToken(user)
				updateUserTokenList(oldToken, user.Token)
			}
		}

	})

	if err != nil {
		log.Fatal("设置定时任务失败:", err)
	}

	c.Start()
	log.Println("成功开启定时任务cron：", CRON_STR)

}

func updateUserTokenList(oldToken string, newToken string) {
	log.Printf("[替换Token-准备替换] oldToken: %s , newToken: %s", oldToken, newToken)
	log.Println("[替换Token-替换前] USER_TOKEN_LIST：", USER_TOKEN_LIST)
	// 遍历切片，找到需要替换的元素的索引
	targetIndex := -1
	for i, token := range USER_TOKEN_LIST {
		if token == oldToken { // 假设你想替换 "token2" 这个元素
			targetIndex = i
			break
		}
	}

	// 如果找到了需要替换的元素，则进行替换
	if targetIndex != -1 {
		USER_TOKEN_LIST[targetIndex] = newToken // 假设你想替换为 "new_token"
	}

	log.Println("[替换Token-替换后] USER_TOKEN_LIST：", USER_TOKEN_LIST)

}

func initCookie() {
	// 打印解码后的数据
	for _, user := range users {
		updateUserToken(user)
		log.Printf("[初始化] 用户：%s,获取到的Token: %s,过期时间：%s ", user.Username, user.Token, user.Expiry)
		//添加到token池中
		USER_TOKEN_LIST = append(USER_TOKEN_LIST, user.Token)
	}
	log.Println("[初始化] USER_TOKEN_LIST为: ", USER_TOKEN_LIST)
}

// 获取token并更新
func updateUserToken(user *User) {
	jsonData, err := json.Marshal(user)
	if err != nil {
		log.Println("JSON编码失败:", err)
		return
	}

	resp, err := http.Post(BingAI_TOKEN_URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()

	// 解析响应的JSON数据
	var responseData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		log.Println("解析JSON失败:", err)
		return
	}

	if responseData["status"] == "fail" {
		log.Println("请求获取Token失败:", responseData["message"])
		return
	}

	// 提取data和expiry字段的值
	user.Token, _ = responseData["data"].(string)
	user.Expiry, _ = responseData["expiry"].(string)

}

func initUserInfo() {
	//初始化切片
	users = []*User{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, USER_INFO_ENV_NAME_PREFIX) {
			parts := strings.SplitN(env, "=", 2)
			//解析第一个用户信息
			parseUser(parts[1])
		}
	}
}

func parseUser(userInfo string) {
	// 解析JSON数据到User结构体
	var user User
	err := json.Unmarshal([]byte(userInfo), &user)
	if err != nil {
		log.Println("解析JSON失败: ", err)
		return
	}
	//添加
	users = append(users, &user)
}

// IsDateEqualToday 判断给定日期字符串是否等于今天的日期
func IsDateEqualToday(dateString string) bool {
	// 获取今天的日期字符串，格式为 "2006-01-02"
	todayString := time.Now().Format("2006-01-02")
	// 截取日期字符串的前10个字符（年、月、日部分）
	datePrefix := dateString[:10]

	/// 比较日期是否相等
	return datePrefix == todayString
}
