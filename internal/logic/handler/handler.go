package handler

import "github.com/iimeta/fastapi/internal/service"

type sHandler struct{}

func init() {
	service.RegisterHandler(New())
}

func New() service.IHandler {
	return &sHandler{}
}

func (s *sHandler) Init() error {
	return nil
}
