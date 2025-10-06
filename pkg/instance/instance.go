package instance

import (
	"errors"
	"net"

	"github.com/jibitesh/request-response-manager/internal/config"
	"github.com/jibitesh/request-response-manager/internal/logger"
)

type Instance struct {
	Name string `json:"name"`
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

func getIp() (string, error) {
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, i := range interfaces {
		if ipnet, ok := i.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			logger.Info("ipnet: %v", ipnet)
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.To4()
				if ip.IsPrivate() {
					return ipnet.IP.String(), nil
				}
			}
		}
	}
	return "", errors.New("could not find a valid private IP address")
}

func GetInstance() (*Instance, error) {
	ip, err := getIp()
	if err != nil {
		logger.Error("Error: %v", err)
		return nil, err
	}
	port := config.AppConfig.Server.Port
	logger.Info("Local Private Ip Address: %s Port: %d \n", ip, port)
	return &Instance{
		Name: "none",
		Ip:   ip,
		Port: port,
	}, nil
}
