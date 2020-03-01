package notifiers

import (
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"net/url"
	"strings"
	"time"
)

const defaultDingdingMsgType = "link"
const dingdingOptionsTemplate = `
      <h3 class="page-heading">DingDing settings</h3>
      <div class="gf-form">
        <span class="gf-form-label width-10">Url</span>
        <input type="text" required class="gf-form-input max-width-70" ng-model="ctrl.model.settings.url" placeholder="https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxxx"></input>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">MessageType</span>
        <select class="gf-form-input max-width-14" ng-model="ctrl.model.settings.msgType" ng-options="s for s in ['link','actionCard','markdown']" ng-init="ctrl.model.settings.msgType=ctrl.model.settings.msgType || '` + defaultDingdingMsgType + `'"></select>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">Mobiles</span>
        <input type="text" class="gf-form-input max-width-70" ng-model="ctrl.model.settings.mobiles" placeholder="markdown MessageType required; such as '186xxx1234,186xxx4321'"></input>
      </div>
`

const markdownTemplate = `
### $title
### $picUrl
### $msg
$items
### 报警时间：$startTime
### 持续时间：$remainTime
### 详情 [detail]($msgUrl)
### $atContent
`

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:            "dingding",
		Name:            "DingDing",
		Description:     "Sends HTTP POST request to DingDing",
		Factory:         newDingDingNotifier,
		OptionsTemplate: dingdingOptionsTemplate,
	})

}

func newDingDingNotifier(model *models.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "Could not find url property in settings"}
	}

	msgType := model.Settings.Get("msgType").MustString(defaultDingdingMsgType)

	mobilesStr := model.Settings.Get("mobiles").MustString("18612626214,15320347357")

	return &DingDingNotifier{
		NotifierBase: NewNotifierBase(model),
		MsgType:      msgType,
		URL:          url,
		AtMobiles:    strings.Split(mobilesStr, ","),
		log:          log.New("alerting.notifier.dingding"),
	}, nil
}

// DingDingNotifier is responsible for sending alert notifications to ding ding.
type DingDingNotifier struct {
	NotifierBase
	MsgType   string
	URL       string
	AtMobiles []string
	log       log.Logger
}

// Notify sends the alert notification to dingding.
func (dd *DingDingNotifier) Notify(evalContext *alerting.EvalContext) error {
	dd.log.Info("Sending dingding")

	messageURL, err := evalContext.GetRuleURL()
	if err != nil {
		dd.log.Error("Failed to get messageUrl", "error", err, "dingding", dd.Name)
		messageURL = ""
	}

	body, err := dd.genBody(evalContext, messageURL)
	if err != nil {
		return err
	}

	dd.log.Info("DingDingBody: ", string(body))

	cmd := &models.SendWebhookSync{
		Url:  dd.URL,
		Body: string(body),
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		dd.log.Error("Failed to send DingDing", "error", err, "dingding", dd.Name)
		return err
	}

	return nil
}

func (dd *DingDingNotifier) genBody(evalContext *alerting.EvalContext, messageURL string) ([]byte, error) {

	q := url.Values{
		"pc_slide": {"false"},
		"url":      {messageURL},
	}

	// Use special link to auto open the message url outside of Dingding
	// Refer: https://open-doc.dingtalk.com/docs/doc.htm?treeId=385&articleId=104972&docType=1#s9
	messageURL = "dingtalk://dingtalkclient/page/link?" + q.Encode()

	dd.log.Info("messageUrl:" + messageURL)

	message := evalContext.Rule.Message
	picURL := evalContext.ImagePublicURL
	title := evalContext.GetNotificationTitle()
	if message == "" {
		message = title
	}

	for i, match := range evalContext.EvalMatches {
		message += fmt.Sprintf("\n%2d. %s: %s", i+1, match.Metric, match.Value)
	}

	var bodyMsg map[string]interface{}
	if dd.MsgType == "actionCard" {
		// Embed the pic into the markdown directly because actionCard doesn't have a picUrl field
		if picURL != "" {
			message = "![](" + picURL + ")\n\n" + message
		}

		bodyMsg = map[string]interface{}{
			"msgtype": "actionCard",
			"actionCard": map[string]string{
				"text":        message,
				"title":       title,
				"singleTitle": "More",
				"singleURL":   messageURL,
			},
		}
	} else if dd.MsgType == "markdown" {
		markdownContent := dd.genMarkdownContent(evalContext, title, messageURL)
		bodyMsg = map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"text":  markdownContent,
				"title": title,
			},
			"at": map[string]interface{}{
				"atMobiles": dd.AtMobiles,
				"isAtAll":   false,
			},
		}
	} else {
		bodyMsg = map[string]interface{}{
			"msgtype": "link",
			"link": map[string]string{
				"text":       message,
				"title":      title,
				"picUrl":     picURL,
				"messageUrl": messageURL,
			},
		}
	}
	return json.Marshal(bodyMsg)
}

func (dd *DingDingNotifier) genMarkdownContent(evalContext *alerting.EvalContext, title, messageURL string) string {
	content := markdownTemplate
	message := evalContext.Rule.Message

	var atMobilesBuilder strings.Builder
	for _, mobile := range dd.AtMobiles {
		atMobilesBuilder.WriteString("@")
		atMobilesBuilder.WriteString(mobile)
		atMobilesBuilder.WriteString(" ")
	}

	items := ""
	for i, match := range evalContext.EvalMatches {
		items += fmt.Sprintf("\n%2d. %s: %s", i+1, match.Metric, match.Value)
	}

	content = strings.Replace(content, "$title", title, -1)
	content = strings.Replace(content, "$msg", message, -1)
	if len(evalContext.ImagePublicURL) > 0 {
		content = strings.Replace(content, "$picUrl", fmt.Sprintf("[picUrl](%s)", evalContext.ImagePublicURL), -1)

	} else {
		content = strings.Replace(content, "$picUrl", evalContext.ImagePublicURL, -1)
	}
	var cstZone = time.FixedZone("CST", 8*3600)
	content = strings.Replace(content, "$items", items, -1)
	content = strings.Replace(content, "$startTime", evalContext.Rule.LastStateChange.In(cstZone).Format("2006-01-02 15:04:05"), -1)
	content = strings.Replace(content, "$remainTime", evalContext.EndTime.Sub(evalContext.Rule.LastStateChange).String(), -1)
	content = strings.Replace(content, "$msgUrl", messageURL, -1)
	content = strings.Replace(content, "$atContent", atMobilesBuilder.String(), -1)
	return content
}
