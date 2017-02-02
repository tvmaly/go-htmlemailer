package htmlemailer

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"testing"
)

func TestSendMailGun(t *testing.T) {

	fromemail := os.Getenv("FROMEMAIL")

	if fromemail == "" {
		t.Skip("please set env FROMEMAIL  to valid email in order to run the test")
	}

	toemail := os.Getenv("TOEMAIL")

	if toemail == "" {
		t.Skip("please set env TOEMAIL to valid email in order to run the test")
	}

	var emailTemplate = template.Must(template.ParseFiles("email.tmpl"))

	var emailbody bytes.Buffer

	emaildata := map[string]string{"host": "https://bestfoodnearme.com", "token": "12345"}

	err := emailTemplate.Execute(&emailbody, emaildata)

	if err != nil {
		t.Fatalf("Error executing template %s", err)
	}

	message := &EmailMessage{HTML: emailbody.String(), Text: "Please confirm email", Subject: "go htmlemailer test bestfoodnearme.com", FromName: "Bestfoodnearme", FromEmail: fromemail, ToEmail: toemail}

	emailconfig := &EmailConfig{}

	if emailconfig.LoadMailGunConfigFromEnv() == false {
		t.Skip("please set mailgun env variables to run this test")
	}

	response, err := emailconfig.Send(message)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("RESPONSE: %s\n", response)

}
