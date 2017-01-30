package htmlemailer

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"errors"
	"fmt"
	//"gopkg.in/mailgun/mailgun-go.v1"
	"github.com/mailgun/mailgun-go"
	"io"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
)

const crlf = "\r\n"

type EmailMessage struct {
	// the full HTML content to be sent
	HTML string `json:"html,omitempty"`
	// optional full text content to be sent
	Text string `json:"text,omitempty"`
	// the message subject
	Subject string `json:"subject,omitempty"`
	// the sender email address.
	FromEmail string `json:"from_email,omitempty"`
	// optional from name to be used
	FromName string `json:"from_name,omitempty"`
	// the email address of the recipient
	ToEmail string `json:"email"`
	// the optional display name to use for the recipient
	Name string `json:"name,omitempty"`
}

type EmailConfig struct {
	ServerName string `json:"servername"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
	Key        string `json:"-"`
	PublicKey  string `json:"publickey"`
	Domain     string `json:"domain"`
}

func (c *EmailConfig) Send(m *EmailMessage) (string, error) {

	if c.HasMailGunConfigExplicitlySet() || c.LoadMailGunConfigFromEnv() {
		return c.SendMailGun(m)
	}

	return c.SendSMTP(m)
}

func (c *EmailConfig) HasMailGunConfigExplicitlySet() bool {

	if c.Key != "" && c.PublicKey != "" && c.Domain != "" {
		return true
	}

	return false

}

func (c *EmailConfig) LoadMailGunConfigFromEnv() bool {

	apikey := os.Getenv("MG_API_KEY")

	domain := os.Getenv("MG_DOMAIN")

	pubkey := os.Getenv("MG_PUBLIC_API_KEY")

	if apikey != "" && domain != "" && pubkey != "" {
		c.Key = apikey
		c.PublicKey = pubkey
		c.Domain = domain
		return true
	}

	return false

}

func (c *EmailConfig) SendMailGun(m *EmailMessage) (string, error) {

	// see https://github.com/GoogleCloudPlatform/golang-samples/blob/master/docs/appengine/mail/mailgun/mailgun.go
	// see https://github.com/mailgun/mailgun-go

	client := mailgun.NewMailgun(c.Domain, c.Key, c.PublicKey)

	message := client.NewMessage(
		m.FromEmail,
		m.Subject,
		m.Text,
		m.ToEmail,
	)

	if m.HTML != "" {
		message.SetHtml(m.HTML)
	}

	msg, id, err := client.Send(message)

	if err != nil {
		return "", errors.New(fmt.Sprintf("Could not send message: %v, ID %v, %+v", err, id, msg))
	}

	return fmt.Sprintf("%v", id), nil
}

func (c *EmailConfig) SendSMTP(m *EmailMessage) (string, error) {

	if c.UserName == "" || c.Password == "" {
		return "", errors.New("UserName or Password is not set in call to SendSMTP")
	}

	from := mail.Address{m.FromName, m.FromEmail}
	to := mail.Address{m.Name, m.ToEmail}

	message, err := c.buildheader(m)

	if err != nil {
		return "", err
	}

	servername := c.ServerName

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", c.UserName, c.Password, host)

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	conn, err := tls.Dial("tcp", servername, tlsconfig)

	if err != nil {
		return "", err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return "", err
	}

	if err = client.Auth(auth); err != nil {
		return "", err
	}

	if err = client.Mail(from.Address); err != nil {
		return "", err
	}

	if err = client.Rcpt(to.Address); err != nil {
		return "", err
	}

	w, err := client.Data()

	if err != nil {
		return "", err
	}

	_, err = w.Write(message)

	if err != nil {
		return "", err
	}

	err = w.Close()

	if err != nil {
		return "", err
	}

	client.Quit()

	return "ok", nil

}

func (c *EmailConfig) buildheader(m *EmailMessage) ([]byte, error) {

	var buffer = &bytes.Buffer{}

	header := textproto.MIMEHeader{}

	from := mail.Address{m.FromName, m.FromEmail}
	to := mail.Address{m.Name, m.ToEmail}
	header.Add("To", to.String())
	header.Add("From", from.String())

	header.Add("Subject", m.Subject)

	mixedw := multipart.NewWriter(buffer)

	header.Add("MIME-Version", "1.0")
	header.Add("Content-Type", fmt.Sprintf("multipart/mixed;%s boundary=%s", crlf, mixedw.Boundary()))

	err := c.writeheader(buffer, header)

	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(buffer, "--%s%s", mixedw.Boundary(), crlf)

	if err != nil {
		return nil, err
	}

	if m.HTML != "" || m.Text != "" {

		altw := multipart.NewWriter(buffer)

		header = textproto.MIMEHeader{}
		header.Add("Content-Type", fmt.Sprintf("multipart/alternative;%s boundary=%s", crlf, altw.Boundary()))

		err := c.writeheader(buffer, header)

		if err != nil {
			return nil, err
		}

		if m.Text != "" {
			header = textproto.MIMEHeader{}
			header.Add("Content-Type", "text/plain; charset=utf-8")
			header.Add("Content-Transfer-Encoding", "quoted-printable")
			//header.Add("Content-Transfer-Encoding", "base64")

			partw, err := altw.CreatePart(header)
			if err != nil {
				return nil, err
			}

			bodyBytes := []byte(m.Text)

			encoder := b64.NewEncoder(b64.StdEncoding, partw)

			_, err = encoder.Write(bodyBytes)

			if err != nil {
				return nil, err
			}

			err = encoder.Close()

			if err != nil {
				return nil, err
			}
		}

		if m.HTML != "" {

			header = textproto.MIMEHeader{}

			header.Add("Content-Type", "text/html; charset=utf-8")
			header.Add("Content-Transfer-Encoding", "base64")

			partw, err := altw.CreatePart(header)

			if err != nil {
				return nil, err
			}

			htmlBodyBytes := []byte(m.HTML)

			encoder := b64.NewEncoder(b64.StdEncoding, partw)

			_, err = encoder.Write(htmlBodyBytes)

			if err != nil {
				return nil, err
			}

			err = encoder.Close()

			if err != nil {
				return nil, err
			}
		}

		altw.Close()
	}

	mixedw.Close()

	return buffer.Bytes(), nil

}

func (c *EmailConfig) writeheader(w io.Writer, header textproto.MIMEHeader) error {

	for k, vs := range header {
		_, err := fmt.Fprintf(w, "%s: ", k)
		if err != nil {
			return err
		}

		for i, v := range vs {

			v = textproto.TrimString(v)

			_, err := fmt.Fprintf(w, "%s", v)
			if err != nil {
				return err
			}

			if i < len(vs)-1 {
				return errors.New("Multiple header values are not supported.")
			}
		}

		_, err = fmt.Fprint(w, crlf)
		if err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(w, crlf)
	if err != nil {
		return err
	}

	return nil
}
