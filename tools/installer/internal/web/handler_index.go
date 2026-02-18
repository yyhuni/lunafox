package web

import "net/http"

func (router *Router) handleIndex(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	page, err := renderIndexHTML(router.baseOptions)
	if err != nil {
		writeAPIError(writer, http.StatusInternalServerError, "INTERNAL", "页面渲染失败", nil)
		return
	}
	_, _ = writer.Write(page)
}
