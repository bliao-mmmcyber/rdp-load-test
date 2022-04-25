package guac

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/appaegis/golang-common/pkg/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/sirupsen/logrus"
)

const (
	CharSet  = "UTF-8"
	Subject  = "You are invited to join %s's RDP session"
	HtmlBody = `
<html>
	<head></head>
	<body>
		<p>
			You are invited to join {{.Inviter}}'s RDP session'<br/>
			Click <a href="{{.Link}}">Here</a> to join.
		</p>
	</body>
</html>
`
)

type MailService interface {
	SendInvitation(to string, inviter string, link string) error
}

type ContentAttributes struct {
	Inviter string
	Link    string
}

type RdpMailService struct{}

func (s RdpMailService) SendInvitation(to string, inviter string, link string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.GetCeCogRegion()),
	},
	)
	if err != nil {
		logrus.Errorf("create aws session failed %v", err)
		return err
	}
	svc := ses.New(sess)
	tmpl, err := template.New("content").Parse(HtmlBody)
	if err != nil {
		return err
	}
	var contentBuilder strings.Builder
	err = tmpl.Execute(&contentBuilder, ContentAttributes{
		Inviter: inviter,
		Link:    link,
	})
	if err != nil {
		return err
	}

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(to),
			},
		},
		Source: aws.String("account@appaegis.com"),
		Message: &ses.Message{
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(fmt.Sprintf(Subject, inviter)),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(contentBuilder.String()),
				},
			},
		},
	}
	_, err = svc.SendEmail(input)
	return err
}
