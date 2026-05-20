package game

type Service struct{}

type BootstrapResponse struct {
	GameName string   `json:"gameName"`
	Modules  []string `json:"modules"`
	Message  string   `json:"message"`
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Bootstrap() BootstrapResponse {
	return BootstrapResponse{
		GameName: "Hero3",
		Modules: []string{
			"player",
			"city",
			"resource",
			"military",
			"map",
			"combat",
			"save",
		},
		Message: "Hero3 后端基础服务已就绪，具体玩法逻辑待接入。",
	}
}
