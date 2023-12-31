package web

import (
	"fmt"
	"github.com/ahmetson/datatype-lib/message"
	"github.com/valyala/fasthttp"
)

const (
	LayerRunning = "running"
	LayerClosed  = "closed"
)

var onlyPostMethod = (&message.Request{}).Fail("only POST method allowed").String()
var emptyBody = (&message.Request{}).Fail("empty body").String()

var errStr = func(err error) (replyStr string) {
	replyStr = (&message.Request{}).Fail(err.Error()).String()
	return
}

func (web *Handler) closeWeb() error {
	if web.status != nil {
		return nil
	}
	if !web.running {
		return nil
	}

	if web.layer == nil {
		return fmt.Errorf("layer not set")
	}

	if err := web.layer.Shutdown(); err != nil {
		return fmt.Errorf("server.Shutdown: %w", err)
	}

	web.running = false

	return nil
}

func (web *Handler) startWeb() {
	instanceConfig := web.Handler.Config()
	addr := fmt.Sprintf(":%d", instanceConfig.Port)

	go func() {
		web.running = true
		web.status = nil

		web.layer = &fasthttp.Server{
			Handler: web.handleWebRequest,
		}

		if err := web.layer.ListenAndServe(addr); err != nil {
			web.status = fmt.Errorf("error in ListenAndServe: %w at port %d", err, instanceConfig.Port)
			web.running = false
		}
	}()
}

func (web *Handler) handleWebRequest(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("json/application; charset=utf8")
	ctx.Response.Header.Set("X-Author", "Medet Ahmetson")

	if !ctx.IsPost() {
		ctx.SetStatusCode(405)
		_, _ = fmt.Fprintf(ctx, "%s", onlyPostMethod)
		return
	}

	body := ctx.PostBody()
	if len(body) == 0 {
		ctx.SetStatusCode(400)
		_, _ = fmt.Fprintf(ctx, "%s", emptyBody)
		return
	}

	// Just to add the Uuid
	request, err := message.NewReq([]string{string(body)})
	if err != nil {
		ctx.SetStatusCode(403)
		_, _ = fmt.Fprintf(ctx, "%s", errStr(err))
		return
	}
	request.SetUuid()

	requestMessage := request.String()

	resp, err := web.pairClient.RawRequest(requestMessage)

	if err != nil {
		ctx.SetStatusCode(403)
		_, _ = fmt.Fprintf(ctx, "%s", errStr(err))
		return
	}

	serverReply, err := message.NewRep(resp)
	if err != nil {
		reply := fmt.Errorf("failed to decode server data: %w", err)
		ctx.SetStatusCode(403)
		_, _ = fmt.Fprintf(ctx, "%s", errStr(reply))
	}

	if serverReply.IsOK() {
		ctx.SetStatusCode(200)
	} else {
		ctx.SetStatusCode(403)
	}
	replyMessage := serverReply.String()
	_, _ = fmt.Fprintf(ctx, "%s", replyMessage)
}
