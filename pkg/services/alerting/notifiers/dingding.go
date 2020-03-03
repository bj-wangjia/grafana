package notifiers

import (
	"encoding/json"
	"fmt"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/setting"
	"net/http"
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
      <gf-form-switch
          class="gf-form"
          label="Alert By Telephone"
          label-class="width-14"
          checked="ctrl.model.settings.telAlert"
          tooltip="钉钉报警持续一定时间会启动电话报警">
      </gf-form-switch>
      <div class="gf-form-inline">
        <div class="gf-form" ng-if="ctrl.model.settings.telAlert">
          <span class="gf-form-label width-12">Tel Alert after
            <info-popover mode="right-normal" position="top center">
              Specify how long  tel alert should be sent, e.g. after 30s, 1m, 10m, 30m or 1h etc.
            </info-popover>
          </span>
          <input type="text" placeholder="Select or specify custom" class="gf-form-input width-15" ng-model="ctrl.model.settings.afterTime"
            bs-typeahead="ctrl.getFrequencySuggestion" data-min-length=0 ng-required="ctrl.model.settings.afterTime">
        </div>
      </div>
`

const markdownTemplate = `
### $title
### $picUrl
### $msg
$items
### 报警时间：$startTime
### 持续时间：$remainTime
### 详情 [detail]($detailUrl)
$telAlertMsg
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
	telAlert := model.Settings.Get("telAlert").MustBool(false)

	afterTime, err := alerting.GetTimeDurationStringToSeconds(model.Settings.Get("afterTime").MustString())
	if err != nil {
		return nil, alerting.ValidationError{Reason: err.Error()}
	}

	return &DingDingNotifier{
		NotifierBase: NewNotifierBase(model),
		MsgType:      msgType,
		URL:          url,
		AtMobiles:    strings.Split(mobilesStr, ","),
		TelAlert:     telAlert,
		AfterTime:    afterTime,
		log:          log.New("alerting.notifier.dingding"),
	}, nil
}

// DingDingNotifier is responsible for sending alert notifications to ding ding.
type DingDingNotifier struct {
	NotifierBase
	MsgType   string
	URL       string
	AtMobiles []string
	TelAlert  bool
	AfterTime int64
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
		markdownContent := dd.genMarkdownContent(evalContext)
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

func (dd *DingDingNotifier) genMarkdownContent(evalContext *alerting.EvalContext) string {
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

	messageURL, _ := evalContext.GetRuleURL()
	title := StatusToString(evalContext.Rule.State)
	title += " " + evalContext.Rule.Name

	if len(evalContext.ImagePublicURL) > 0 {
		content = strings.Replace(content, "$picUrl", fmt.Sprintf("![picUrl](%s)", evalContext.ImagePublicURL), -1)
	} else {
		content = strings.Replace(content, "$picUrl", evalContext.ImagePublicURL, -1)
	}
	var cstZone = time.FixedZone("CST", 8*3600)

	remainTime := evalContext.EndTime.Sub(evalContext.Rule.LastStateChange)
	if remainTime < 0 {
		remainTime = -remainTime
	}
	telephoneMsg := ""
	if dd.TelAlert {
		d := time.Duration(dd.AfterTime)*time.Second - remainTime
		if d > time.Minute {
			telephoneMsg += "距离电话报警还有 " + d.String()
		} else {
			telephoneMsg += "正在电话报警...\n>" + dd.telAlert(title, message)
		}
	} else {
		telephoneMsg += "没有开启电话报警功能"
	}

	content = strings.Replace(content, "$title", title, -1)
	content = strings.Replace(content, "$msg", message, -1)
	content = strings.Replace(content, "$items", items, -1)
	content = strings.Replace(content, "$telAlertMsg", telephoneMsg, -1)
	content = strings.Replace(content, "$startTime", evalContext.Rule.LastStateChange.In(cstZone).Format("2006-01-02 15:04:05"), -1)
	content = strings.Replace(content, "$remainTime", remainTime.String(), -1)
	content = strings.Replace(content, "$detailUrl", messageURL, -1)
	content = strings.Replace(content, "$atContent", atMobilesBuilder.String(), -1)
	dd.log.Info("content:"+content, "messageUrl:"+messageURL+"-*-", "picUrl:"+evalContext.ImagePublicURL)
	return content
}

type TelCalResponse struct {
	Message string
	Code    string
}

func (dd *DingDingNotifier) telAlert(title, message string) string {
	if len(setting.TelAlertUrl) <= 0 {
		return "报警失败: 没有配置报警Url"
	}
	for _, tel := range dd.AtMobiles {
		v := url.Values{}
		v.Add("tel", tel)
		v.Add("platform", title)
		v.Add("msg", message)
		u := fmt.Sprintf("http://%s/?%s", setting.TelAlertUrl, v.Encode())
		dd.log.Info("TelAlert: ", u)
		res, err := http.Get(u)
		if err != nil {
			return "报警失败: " + err.Error()
		}
		if res.StatusCode != 200 {
			return "报警失败: " + res.Status
		}
	}

	return "电话报警成功."
}

func StatusToString(stateType models.AlertStateType) string {
	switch stateType {
	case models.AlertStateAlerting:
		return "[报警中]"
	case models.AlertStateOK:
		return "[报警已恢复]"
	case models.AlertStateNoData:
		return "[没有查到数据]"
	default:
		return "[未知错误]"
	}
}
