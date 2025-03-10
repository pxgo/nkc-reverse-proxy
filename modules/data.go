package modules

import (
	"errors"
	"net/http"
)

var GlobalServices IPortHostServices

type IService struct {
	Server   IServer
	Location ILocation
	Global   IGlobal
	Template Template
}

func (service *IService) GetTemplateContentByStatus(status int) TemplateContent {
	switch status {
	case 500:
		return service.Template.Page500
	case 429:
		return service.Template.Page429
	case 503:
		return service.Template.Page503
	case 404:
		return service.Template.Page404
	case 403:
		return service.Template.Page403
	default:
		return service.Template.Page500
	}
}

type IHostService map[string][]IService

type IPortHostServices map[uint16]IHostService

type ILocation struct {
	Reg          string
	Pass         []string
	Balance      string
	ReqLimit     *[]*IReqLimit
	IpLimit      *[]*IIpLimit
	RedirectCode int
	RedirectUrl  string
	RootHandler  http.Handler
}

type IServer struct {
	Listen   uint16
	Name     []string
	SSLKey   string
	SSLCert  string
	ReqLimit *[]*IReqLimit
	IpLimit  *[]*IIpLimit
}

type IGlobal struct {
	ReqLimit *[]*IReqLimit
	IpLimit  *[]*IIpLimit
}

// InitGlobalServices 在这里准备所有可能需要的数据
// 读取YAML文件内容
// 加载原始配置数据
// 转换部分字段数据
// 缓存进配置字段
func InitGlobalServices() error {
	if GlobalServices == nil {
		GlobalServices = make(IPortHostServices)
	}
	GlobalReqLimit, err := GetReqLimitByString(GlobalConfigs.ReqLimit)
	if err != nil {
		return err
	}
	GlobalIpLimit, err := GetIpLimitByString(GlobalConfigs.IpLimit)
	if err != nil {
		return err
	}
	iGlobal := IGlobal{
		ReqLimit: &GlobalReqLimit,
		IpLimit:  &GlobalIpLimit,
	}

	for _, server := range GlobalConfigs.Servers {

		var services []IService

		if GlobalServices[server.Listen] == nil {
			GlobalServices[server.Listen] = make(IHostService)
		}
		serverReqLimit, err := GetReqLimitByString(server.ReqLimit)
		if err != nil {
			return err
		}
		serverIpLimit, err := GetIpLimitByString(server.IpLimit)
		if err != nil {
			return err
		}

		iServer := IServer{
			ReqLimit: &serverReqLimit,
			IpLimit:  &serverIpLimit,
		}

		var template = Template{
			Page404: server.Page404,
			Page429: server.Page429,
			Page500: server.Page500,
			Page503: server.Page503,
			Page403: server.Page403,
		}

		if template.Page429.Title == "" {
			template.Page429 = GlobalConfigs.Template.Page429
		}

		if template.Page404.Title == "" {
			template.Page404 = GlobalConfigs.Template.Page404
		}

		if template.Page500.Title == "" {
			template.Page500 = GlobalConfigs.Template.Page500
		}

		if template.Page503.Title == "" {
			template.Page503 = GlobalConfigs.Template.Page503
		}

		for _, location := range server.Location {
			locationReqLimit, err := GetReqLimitByString(location.ReqLimit)
			if err != nil {
				return err
			}
			locationIpLimit, err := GetIpLimitByString(location.IpLimit)
			if err != nil {
				return err
			}

			var RootHandler http.Handler

			if location.Root != "" {
				targetPath, err := GetAbsPath(location.Root)
				if err != nil {
					return err
				}
				if !IsDirValid(targetPath) {
					return errors.New("Invalid directory: " + location.Root)
				}
				RootHandler = http.FileServer(http.Dir(targetPath))
				if location.RootPrefix != "" {
					RootHandler = http.StripPrefix(location.RootPrefix, RootHandler)
				}
			}

			iLocation := ILocation{
				Reg:          location.Reg,
				Pass:         location.Pass,
				Balance:      location.Balance,
				ReqLimit:     &locationReqLimit,
				IpLimit:      &locationIpLimit,
				RedirectCode: location.RedirectCode,
				RedirectUrl:  location.RedirectUrl,
				RootHandler:  RootHandler,
			}

			services = append(services, IService{
				Global:   iGlobal,
				Server:   iServer,
				Template: template,
				Location: iLocation,
			})
		}

		for _, name := range server.Name {
			if GlobalServices[server.Listen][name] == nil {
				GlobalServices[server.Listen][name] = services
			} else {
				return errors.New("duplicate domain name or port")
			}
		}
	}
	return nil
}
