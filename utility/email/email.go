package email

import (
	"crypto/tls"
	"github.com/iimeta/fastapi-admin/internal/config"
	"gopkg.in/gomail.v2"
)

type Option struct {
	To      []string // 收件人
	Subject string   // 邮件主题
	Body    string   // 邮件正文
}

type OptionFunc func(msg *gomail.Message)

func SendMail(email *Option, opt ...OptionFunc) error {

	m := gomail.NewMessage()

	// 这种方式可以添加别名, 即“XX官方”
	m.SetHeader("From", m.FormatAddress(config.Cfg.Email.UserName, config.Cfg.Email.FromName))

	if len(email.To) > 0 {
		m.SetHeader("To", email.To...)
	}

	if len(email.Subject) > 0 {
		m.SetHeader("Subject", email.Subject)
	}

	if len(email.Body) > 0 {
		m.SetBody("text/html", email.Body)
	}

	// m.SetHeader("Cc", m.FormatAddress("xxxx@foxmail.com", "收件人")) //抄送
	// m.SetHeader("Bcc", m.FormatAddress("xxxx@gmail.com", "收件人"))  //暗送

	for _, o := range opt {
		o(m)
	}

	return do(m)
}

func do(msg *gomail.Message) error {
	dialer := gomail.NewDialer(config.Cfg.Email.Host, config.Cfg.Email.Port, config.Cfg.Email.UserName, config.Cfg.Email.Password)

	// 自动开启SSL
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	return dialer.DialAndSend(msg)
}
