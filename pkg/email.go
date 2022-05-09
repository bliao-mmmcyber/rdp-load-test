package guac

import (
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
	Subject  = "Appaegis RDP application screen share"
	HtmlBody = `
<html>
	<head></head>
	<body>
      <div style="text-align: center;">
		<img
		  style="display: block; margin-left: auto; margin-right: auto;"
		  src="https://appaegis-public.s3.amazonaws.com/logo.png"
		  width="250"
		  height="auto"
		/>
		<span style="font-size: 12pt;">
		  <strong>
			<span style = "color: #373757; font-family: arial, helvetica, sans-serif;">
			</span>
		  </strong>
		</span>
		<br />
		<br />
		<table
		  style="
			min-height: 200px;
			width: 100%;
			border-collapse: collapse;
			background-color: #f2f5f9;
			border-color: #ffffff;
			border-style: none;"
		    border="1"
		>
		  <tbody >
			<tr>
			  <td style="width: 100%; text-align: center;">
				<span
				  style="
					color: #373757;
					font-family: arial, helvetica, sans-serif;
					font-size: 12pt;
				">
				  {{.Inviter}} was invite you to join a screen share. <br/>
				  Please click the below link, then you can join the screen share.
                  <br/><br/>
                  Here is the screen share link
				</span>
				<br />
				<br />
				<a
				  href="{{.Link}}"
				  target="_blank"
				  style="
					background-color: #00c2c6;
					color: #ffffff;
					border: 0px solid #000000;
					border-radius: 3px;
					box-sizing: border-box;
					font-family: arial, helvetica, sans-serif;
					font-size: 13px;
					font-weight: bold;
					line-height: 40px;
					padding: 12px 24px;
					text-align: center;
					text-decoration: none;
					vertical-align: middle;"
				  rel="noopener"
				>Application Name: {{.AppName}}</a>
			  </td>
			</tr>
		  </tbody>
		</table>
		<br />
	  </div>
	</body>
</html>
`
)

type MailService interface {
	SendInvitation(to string, inviter string, link string, appName string) error
}

type ContentAttributes struct {
	Inviter string
	Link    string
	AppName string
}

type RdpMailService struct{}

func (s RdpMailService) SendInvitation(to string, inviter string, link string, appName string) error {
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
		AppName: appName,
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
				Data:    aws.String(Subject),
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
