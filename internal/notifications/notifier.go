package notifications

import (
	"fmt"
	"log"
	"net/smtp"
)

type Notifier struct {
	SMTPServer string
	Port       string
	Username   string
	Password   string
	From       string
	To         []string
}

func NewNotifier(smtpServer, port, username, password, from string, to []string) *Notifier {
	return &Notifier{
		SMTPServer: smtpServer,
		Port:       port,
		Username:   username,
		Password:   password,
		From:       from,
		To:         to,
	}
}

func (n *Notifier) SendNotification(siteURL string) error {
	subject := "Сбой сайта"
	body := fmt.Sprintf("Сайт %s недоступен.", siteURL)
	message := []byte("Subject: " + subject + "\r\n" +
		"From: " + n.From + "\r\n" +
		"To: " + fmt.Sprintf("%v", n.To) + "\r\n" +
		"\r\n" +
		body)

	auth := smtp.PlainAuth("", n.Username, n.Password, n.SMTPServer)
	err := smtp.SendMail(n.SMTPServer+":"+n.Port, auth, n.From, n.To, message)
	if err != nil {
		log.Printf("Ошибка при отправке уведомления: %v", err)
		return err
	}

	log.Println("Уведомление отправлено успешно.")
	return nil
}