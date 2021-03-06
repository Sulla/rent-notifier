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
	VkMessages       chan model.Message
	DB               *dbal.DBAL
	Prefix           string
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

	log.Printf("notify note {city: %d, type: %d, contact: %s, link: %s}", note.City, note.Type, note.Contact, note.Link)

	recipients, err := controller.DB.FindRecipientsByNote(note)

	if err != nil {
		log.Printf("error find recipients by note: %s", err)

		return err
	}

	vkIds := make([]int, 0)
	for _, recipient := range recipients {

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

func (controller ApiController) formatMessageTelegram(note dbal.Note) string {

	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("\n<b>Тип</b>: %s\n", model.FormatType(note.Type)))

	if note.Price > 0 {
		b.WriteString(fmt.Sprintf("<b>Цена</b>: %s руб/мес\n", model.FormatPrice(note.Price)))
	}

	if len(note.Subways) > 0 {
		b.WriteString(fmt.Sprintf("<b>Метро</b>: %s\n", model.FormatSubways(controller.DB, note.Subways)))
	}

	b.WriteString(fmt.Sprintf("<b>Ссылка</b>: <a href='%s'>Перейти к объявлению</a>\n\n", note.Link))

	return b.String()
}

func (controller ApiController) formatMessageVk(note dbal.Note) string {

	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("\nТип: %s\n", model.FormatType(note.Type)))

	if note.Price > 0 {
		b.WriteString(fmt.Sprintf("Цена: %s руб/мес\n", model.FormatPrice(note.Price)))
	}

	if len(note.Subways) > 0 {
		b.WriteString(fmt.Sprintf("Метро: %s\n", model.FormatSubways(controller.DB, note.Subways)))
	}

	b.WriteString(fmt.Sprintf("Ссылка: %s\n", note.Link))

	if note.Source == dbal.NOTE_VK_COMMENT {
		b.WriteString("Эта ссылка открывается верно только в браузере\n")
	} else {
		b.WriteString("\n")
	}

	return b.String()
}
