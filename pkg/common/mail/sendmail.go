/*
 * File: sendmail.go
 * Project: mail
 * File Created: Tuesday, 29th March 2022 5:45:50 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package mail

import (
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/log"
)

// MailService represents the interface for our mail service.
type PortalMailService interface {
	CreateMail(mailReq *PortalMail) []byte
	SendMail(mailReq *PortalMail) error
	NewMail(from string, to []string, subject string, mailType MailType, data *PortalMailData) *PortalMail
}

type ModelMailService interface {
	CreateMail(mailReq *ModelMail) []byte
	SendMail(mailReq *ModelMail) error
	NewMail(from string, to []string, subject string, mailType MailType, data *ModelMailData) *ModelMail
}

type MailType int

// List of Mail Types we are going to send.
const (
	MailConfirmation MailType = iota + 1
	PassReset
	TrainingConfirmation
)

// PortalMailData represents the data to be sent to the template of the mail.
type PortalMailData struct {
	Username string
	Code     string
}

// ModelMailData represents the data to be sent to the template of the mail.
type ModelMailData struct {
	Username string
	Status   string
	Project  string
	Model    string
}

// PortalMail represents an email request.
type PortalMail struct {
	from    string
	to      []string
	subject string
	mtype   MailType
	data    *PortalMailData
}

// ModelMail represents an email request.
type ModelMail struct {
	from    string
	to      []string
	subject string
	mtype   MailType
	data    *ModelMailData
}

// SGPortalMailService is the sendgrid implementation of our MailService.
type SGPortalMailService struct {
	sendGridApiKey      string
	mailVerifTemplateID string
	passResetTemplateID string

	PassResetCodeExpiration int
	PassResetEmail          string
	PassResetSubject        string
}

// SGModelMailService is the sendgrid implementation of our MailService.
type SGModelMailService struct {
	sendGridApiKey             string
	trainingCompleteTemplateID string

	TrainingCompleteEmail   string
	TrainingCompleteSubject string
}

// NewPortalSGMailService returns a new instance of SGPortalMailService
func NewPortalSGMailService(sendGridApiKey, mailVerifTemplateID, passResetTemplateID string, passResetCodeExpiration int, passResetEmail, passResetSubject string) *SGPortalMailService {
	return &SGPortalMailService{
		sendGridApiKey:          sendGridApiKey,
		mailVerifTemplateID:     mailVerifTemplateID,
		passResetTemplateID:     passResetTemplateID,
		PassResetCodeExpiration: passResetCodeExpiration,
		PassResetEmail:          passResetEmail,
		PassResetSubject:        passResetSubject,
	}
}

// NewModelSGMailService returns a new instance of SGModelMailService
func NewSGModelMailService(sendGridApiKey, trainingCompleteTemplateID, trainingCompleteEmail, trainingCompleteSubject string) *SGModelMailService {
	return &SGModelMailService{
		sendGridApiKey:             sendGridApiKey,
		trainingCompleteTemplateID: trainingCompleteTemplateID,
		TrainingCompleteEmail:      trainingCompleteEmail,
		TrainingCompleteSubject:    trainingCompleteSubject,
	}
}

// CreateMail takes in a mail request and constructs a sendgrid mail type.
func (ms *SGPortalMailService) CreateMail(mailReq *PortalMail) []byte {
	m := mail.NewV3Mail()

	from := mail.NewEmail("Emerald-AI", mailReq.from)
	m.SetFrom(from)

	if mailReq.mtype == MailConfirmation {
		m.SetTemplateID(ms.mailVerifTemplateID)
	} else if mailReq.mtype == PassReset {
		m.SetTemplateID(ms.passResetTemplateID)
	}

	p := mail.NewPersonalization()

	tos := make([]*mail.Email, 0)
	for _, to := range mailReq.to {
		tos = append(tos, mail.NewEmail("user", to))
	}

	p.AddTos(tos...)
	p.SetDynamicTemplateData("Username", mailReq.data.Username)
	p.SetDynamicTemplateData("Code", mailReq.data.Code)
	m.AddPersonalizations(p)
	return mail.GetRequestBody(m)
}

// CreateMail takes in a mail request and constructs a sendgrid mail type.
func (ms *SGModelMailService) CreateMail(mailReq *ModelMail) []byte {
	m := mail.NewV3Mail()

	from := mail.NewEmail("Emerald-AI", mailReq.from)
	m.SetFrom(from)

	if mailReq.mtype == TrainingConfirmation {
		m.SetTemplateID(ms.trainingCompleteTemplateID)
	}

	p := mail.NewPersonalization()

	tos := make([]*mail.Email, 0)
	for _, to := range mailReq.to {
		tos = append(tos, mail.NewEmail("user", to))
	}

	p.AddTos(tos...)
	p.SetDynamicTemplateData("Username", mailReq.data.Username)
	p.SetDynamicTemplateData("Status", mailReq.data.Status)
	p.SetDynamicTemplateData("Project", mailReq.data.Project)
	p.SetDynamicTemplateData("Model", mailReq.data.Model)
	m.AddPersonalizations(p)
	return mail.GetRequestBody(m)
}

// SendMail creates a sendgrid mail from the given mail request and sends it.
func (ms *SGPortalMailService) SendMail(mailReq *PortalMail) error {
	request := sendgrid.GetRequest(ms.sendGridApiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = ms.CreateMail(mailReq)
	request.Body = Body
	response, err := sendgrid.API(request)
	if err != nil {
		log.Errorf("unable to send mail", "error", err)
		return err
	}

	if response.StatusCode == 202 || response.StatusCode != 200 {
		log.Infof("mail sent successfully; sent status code=%d", response.StatusCode)
		return nil
	}

	log.Errorf("unable to send mail; unexpected sendgrid status code=%d", response.StatusCode)
	return fmt.Errorf("unexpected mail status code=%d", response.StatusCode)
}

// SendMail creates a sendgrid mail from the given mail request and sends it.
func (ms *SGModelMailService) SendMail(mailReq *ModelMail) error {
	request := sendgrid.GetRequest(ms.sendGridApiKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = ms.CreateMail(mailReq)
	request.Body = Body
	response, err := sendgrid.API(request)
	if err != nil {
		log.Errorf("unable to send mail", "error", err)
		return err
	}

	if response.StatusCode == 202 || response.StatusCode != 200 {
		log.Infof("mail sent successfully; sent status code=%d", response.StatusCode)
		return nil
	}

	log.Errorf("unable to send mail; unexpected sendgrid status code=%d", response.StatusCode)
	return fmt.Errorf("unexpected mail status code=%d", response.StatusCode)
}

// NewMail returns a new mail request.
func (ms *SGPortalMailService) NewMail(from string, to []string, subject string, mailType MailType, data *PortalMailData) *PortalMail {
	return &PortalMail{
		from:    from,
		to:      to,
		subject: subject,
		mtype:   mailType,
		data:    data,
	}
}

// NewMail returns a new mail request.
func (ms *SGModelMailService) NewMail(from string, to []string, subject string, mailType MailType, data *ModelMailData) *ModelMail {
	return &ModelMail{
		from:    from,
		to:      to,
		subject: subject,
		mtype:   mailType,
		data:    data,
	}
}
