package controller

import (
	"github.com/valyala/fasthttp"
	"encoding/json"
	"rent-notifier/src/db"
	"rent-notifier/src/model"
	"log"
	"bytes"
	"fmt"
)

type ApiController struct {
	TelegramMessages chan model.Message
	VkMessages chan model.Message
	Db       *dbal.DBAL
	Prefix   string
}

func (controller ApiController) Notify(ctx *fasthttp.RequestCtx) error {

	ctx.SetContentType("application/json")

	body := string(ctx.PostBody())

	note := dbal.Note{}

	err := json.Unmarshal([]byte(body), &note)

	if nil != err {
		log.Printf("unmarshal error: %s", err)

		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte(`{"status": "err"}`))

		return err
	}

	vkIds := make([]int, 0)
	for _, recipient := range controller.Db.FindRecipientsByNote(note) {

		switch(recipient.ChatType) {
		case dbal.RECIPIENT_TELEGRAM:
			text := controller.formatMessageTelegram(note)
			controller.TelegramMessages <- model.Message{ChatId: recipient.ChatId, Text: text}
			break

		case dbal.RECIPIENT_VK:
			vkIds = append(vkIds, recipient.ChatId)
			break

		default:
			log.Printf("invalid recipient chat type: %s", recipient.ChatType)
		}
	}

	if len(vkIds) > 0 {
		text := controller.formatMessageVk(note)
		controller.VkMessages <- model.Message{ChatIds: vkIds, IsBulk: true, Text: text}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte(`{"status": "ok"}`))

	return nil
}

func (controller ApiController) formatMessageTelegram (note dbal.Note) string {

	var b bytes.Buffer

	b.WriteString("\n******socrent.ru******\n")
	b.WriteString(fmt.Sprintf("<b>%s</b>\n", model.FormatHeader(controller.Db, note)))
	b.WriteString(fmt.Sprintf("<a href='%s'>Перейти к объявлению</a>\n", note.Link))

	return b.String()
}

func (controller ApiController) formatMessageVk (note dbal.Note) string {

	var b bytes.Buffer

	b.WriteString("\n******socrent.ru******\n")
	b.WriteString(model.FormatHeader(controller.Db, note))
	b.WriteString(fmt.Sprintf("\n%s", note.Link))

	return b.String()
}